// pkg/dag/execute.go
package dag

import (
	"context"
	"fmt"
	"sync"
)

// Option configures execution behaviour.
type Option func(*execConfig)

type execConfig struct {
	inputs         map[string]any
	concurrency    int
	onNodeComplete func(nodeID string)
}

// WithInputs seeds the data store with initial values before execution begins.
func WithInputs(inputs map[string]any) Option {
	return func(c *execConfig) { c.inputs = inputs }
}

// WithConcurrency limits how many nodes may run concurrently within a layer.
// 0 means unlimited (default).
func WithConcurrency(n int) Option {
	return func(c *execConfig) { c.concurrency = n }
}

// WithOnNodeComplete registers a callback invoked after each node completes successfully.
func WithOnNodeComplete(fn func(nodeID string)) Option {
	return func(c *execConfig) { c.onNodeComplete = fn }
}

// Execute runs the DAG to completion using layer-parallel execution.
//
// Algorithm:
//  1. Compute topological layers via Layers()
//  2. Seed data store with WithInputs values
//  3. For each layer, run all active (non-deactivated) nodes concurrently
//  4. KindRouter nodes: "__route" output deactivates all downstream except the chosen branch
//  5. First node error (after retries) cancels execution via context
//
// Nodes with nil Run are skipped (used for analysis-only nodes from formal.Graph).
func (d *DAG) Execute(ctx context.Context, opts ...Option) (map[string]any, error) {
	cfg := &execConfig{}
	for _, o := range opts {
		o(cfg)
	}

	store := &dataStore{data: make(map[string]any)}
	if cfg.inputs != nil {
		for k, v := range cfg.inputs {
			store.put(k, v)
		}
	}

	layers := d.Layers()
	deactivated := make(map[string]bool)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, layer := range layers {
		active := make([]string, 0, len(layer))
		for _, id := range layer {
			if !deactivated[id] {
				active = append(active, id)
			}
		}
		if len(active) == 0 {
			continue
		}

		// Check context before starting the layer
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		var wg sync.WaitGroup
		errCh := make(chan error, len(active))
		sem := makeSem(cfg.concurrency)

		// deactivation updates from this layer — collect then apply after WaitGroup
		var deMu sync.Mutex
		layerDeactivations := make(map[string]bool)

		for _, id := range active {
			wg.Add(1)
			go func(nodeID string) {
				defer wg.Done()

				if sem != nil {
					sem <- struct{}{}
					defer func() { <-sem }()
				}

				n := d.nodes[nodeID]
				inputs := store.gather(n.Consumes)
				out, err := runWithRetry(ctx, n, inputs)
				if err != nil {
					errCh <- fmt.Errorf("node %q: %w", nodeID, err)
					cancel()
					return
				}
				store.putAll(out)

				if n.Kind == KindRouter {
					chosen, _ := out["__route"].(string)
					deMu.Lock()
					for _, downstream := range d.adj[nodeID] {
						if downstream != chosen {
							layerDeactivations[downstream] = true
						}
					}
					deMu.Unlock()
				}

				if cfg.onNodeComplete != nil {
					cfg.onNodeComplete(nodeID)
				}
			}(id)
		}

		wg.Wait()
		close(errCh)

		if err := <-errCh; err != nil {
			return nil, err
		}

		// Apply deactivations after full layer completes
		for id := range layerDeactivations {
			deactivated[id] = true
		}
	}

	return store.terminalOutputs(d), nil
}

// runWithRetry executes a node's Run function, retrying up to n.MaxRetries times on error.
// If n.Timeout > 0, each attempt is cancelled after that duration.
// Nodes with nil Run return empty outputs immediately.
func runWithRetry(ctx context.Context, n *Node, inputs map[string]any) (map[string]any, error) {
	if n.Run == nil {
		return map[string]any{}, nil
	}

	attempt := func() (map[string]any, error) {
		runCtx := ctx
		var cancel context.CancelFunc
		if n.Timeout > 0 {
			runCtx, cancel = context.WithTimeout(ctx, n.Timeout)
			defer cancel()
		}
		return n.Run(runCtx, inputs)
	}

	out, err := attempt()
	for i := 0; i < n.MaxRetries && err != nil; i++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		out, err = attempt()
	}
	return out, err
}

// makeSem returns a buffered channel semaphore, or nil if n <= 0 (unlimited).
func makeSem(n int) chan struct{} {
	if n <= 0 {
		return nil
	}
	return make(chan struct{}, n)
}

// --- dataStore ---

type dataStore struct {
	mu   sync.RWMutex
	data map[string]any
}

func (s *dataStore) gather(consumes []string) map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]any, len(consumes))
	for _, k := range consumes {
		if v, ok := s.data[k]; ok {
			out[k] = v
		}
	}
	return out
}

func (s *dataStore) put(key string, val any) {
	s.mu.Lock()
	s.data[key] = val
	s.mu.Unlock()
}

func (s *dataStore) putAll(m map[string]any) {
	s.mu.Lock()
	for k, v := range m {
		s.data[k] = v
	}
	s.mu.Unlock()
}

// terminalOutputs returns values produced by terminal nodes (nodes with no downstream).
func (s *dataStore) terminalOutputs(d *DAG) map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]any)
	for id, n := range d.nodes {
		if len(d.adj[id]) == 0 {
			for _, p := range n.Produces {
				if v, ok := s.data[p]; ok {
					out[p] = v
				}
			}
		}
	}
	return out
}
