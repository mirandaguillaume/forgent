// pkg/dag/execute.go
package dag

import (
	"context"
	"fmt"
	"sort"
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

// Execute runs the graph to completion.
//
// For acyclic graphs (no back-edges), uses layer-parallel execution.
// For graphs with back-edges, uses an event-driven scheduler that tracks
// per-back-edge iteration counters and re-enqueues cycle nodes.
//
// Nodes with nil Run are skipped (used for analysis-only nodes).
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

	var err error
	if d.HasBackEdges() {
		err = d.executeEventDriven(ctx, store, cfg)
	} else {
		err = d.executeLayerBased(ctx, store, cfg)
	}
	if err != nil {
		return nil, err
	}
	return store.terminalOutputs(d), nil
}

// executeLayerBased runs the graph layer-by-layer (fast path for acyclic graphs).
func (d *DAG) executeLayerBased(ctx context.Context, store *dataStore, cfg *execConfig) error {
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

		if ctx.Err() != nil {
			return ctx.Err()
		}

		var wg sync.WaitGroup
		errCh := make(chan error, len(active))
		sem := makeSem(cfg.concurrency)

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
				out, err := runNode(ctx, d, n, inputs)
				if err != nil {
					errCh <- fmt.Errorf("node %q: %w", nodeID, err)
					cancel()
					return
				}
				store.putAll(out)

				if n.Kind == KindRouter {
					chosen, _ := out["__route"].(string)
					deMu.Lock()
					for _, e := range d.adj[nodeID] {
						if e.To != chosen {
							layerDeactivations[e.To] = true
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
			return err
		}

		for id := range layerDeactivations {
			deactivated[id] = true
		}
	}

	return nil
}

// nodeState tracks execution state in the event-driven scheduler.
type nodeState int

const (
	statePending nodeState = iota
	stateRunning
	stateDone
)

// executeEventDriven runs graphs with back-edges using a ready-node scheduler.
//
// Algorithm:
//  1. All nodes start as pending
//  2. Nodes with all forward predecessors done are ready
//  3. Run all ready nodes concurrently
//  4. When a back-edge source completes and iterations < MaxIterations,
//     reset all nodes in the cycle path to pending for re-execution
//  5. Repeat until no more ready nodes
func (d *DAG) executeEventDriven(ctx context.Context, store *dataStore, cfg *execConfig) error {
	states := make(map[string]nodeState, len(d.nodes))
	for id := range d.nodes {
		states[id] = statePending
	}
	iterations := make(map[EdgeKey]int)
	deactivated := make(map[string]bool)

	sem := makeSem(cfg.concurrency)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Find ready nodes: pending, not deactivated, all forward predecessors done
		var ready []string
		for id := range d.nodes {
			if states[id] != statePending || deactivated[id] {
				continue
			}
			allReady := true
			for _, e := range d.rev[id] {
				if e.IsBackEdge() {
					continue
				}
				if states[e.From] != stateDone {
					allReady = false
					break
				}
			}
			if allReady {
				ready = append(ready, id)
			}
		}

		if len(ready) == 0 {
			break
		}

		sort.Strings(ready) // deterministic ordering

		for _, id := range ready {
			states[id] = stateRunning
		}

		var wg sync.WaitGroup
		errCh := make(chan error, len(ready))
		var mu sync.Mutex

		for _, id := range ready {
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

				mu.Lock()
				states[nodeID] = stateDone

				if n.Kind == KindRouter {
					chosen, _ := out["__route"].(string)
					for _, e := range d.adj[nodeID] {
						if e.To != chosen {
							deactivated[e.To] = true
						}
					}
				}

				// Back-edge re-enqueue: reset cycle nodes to pending
				for _, e := range d.adj[nodeID] {
					if !e.IsBackEdge() {
						continue
					}
					key := EdgeKey{From: e.From, To: e.To}
					if iterations[key] < e.MaxIterations {
						iterations[key]++
						path := d.forwardPath(e.To, e.From)
						for _, pid := range path {
							states[pid] = statePending
						}
					}
				}
				mu.Unlock()

				if cfg.onNodeComplete != nil {
					cfg.onNodeComplete(nodeID)
				}
			}(id)
		}

		wg.Wait()
		close(errCh)

		if err := <-errCh; err != nil {
			return err
		}
	}

	return nil
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

// runNode dispatches to kind-specific handlers.
func runNode(ctx context.Context, d *DAG, n *Node, inputs map[string]any) (map[string]any, error) {
	switch n.Kind {
	case KindMap:
		return runMap(ctx, n, inputs)
	case KindFilter:
		return runFilter(ctx, n, inputs)
	case KindFlatMap:
		return runFlatMap(ctx, n, inputs)
	case KindReduce:
		return runReduce(ctx, n, inputs)
	case KindBatch:
		return runBatch(ctx, n, inputs)
	case KindZip:
		return runZip(n, inputs)
	case KindGate:
		return runGate(ctx, n, inputs)
	case KindFallback:
		return runFallback(ctx, d, n, inputs)
	case KindRace:
		return runRace(ctx, d, n, inputs)
	case KindCache:
		return runCache(ctx, n, inputs)
	default:
		return runWithRetry(ctx, n, inputs)
	}
}

// findListInput finds the first []any value in inputs.
func findListInput(inputs map[string]any) (string, []any, bool) {
	for k, v := range inputs {
		if l, ok := v.([]any); ok {
			return k, l, true
		}
	}
	return "", nil, false
}

// runMap runs n.Run in parallel for each item in the first []any input.
// Output: first Produces key → []any of results.
func runMap(ctx context.Context, n *Node, inputs map[string]any) (map[string]any, error) {
	listKey, list, ok := findListInput(inputs)
	if !ok {
		return nil, fmt.Errorf("map node %q: no []any input found", n.ID)
	}

	results := make([]any, len(list))
	var wg sync.WaitGroup
	errCh := make(chan error, len(list))

	for i, item := range list {
		wg.Add(1)
		go func(idx int, it any) {
			defer wg.Done()
			itemInputs := make(map[string]any, len(inputs))
			for k, v := range inputs {
				itemInputs[k] = v
			}
			itemInputs[listKey] = it
			out, err := runWithRetry(ctx, n, itemInputs)
			if err != nil {
				errCh <- err
				return
			}
			for _, v := range out {
				results[idx] = v
				break
			}
		}(i, item)
	}

	wg.Wait()
	close(errCh)
	if err := <-errCh; err != nil {
		return nil, err
	}

	out := make(map[string]any)
	if len(n.Produces) > 0 {
		out[n.Produces[0]] = results
	}
	return out, nil
}

// runFilter runs n.Run per item; keeps items where output contains __keep=true.
func runFilter(ctx context.Context, n *Node, inputs map[string]any) (map[string]any, error) {
	listKey, list, ok := findListInput(inputs)
	if !ok {
		return nil, fmt.Errorf("filter node %q: no []any input found", n.ID)
	}

	type result struct {
		idx  int
		keep bool
	}
	results := make([]result, len(list))
	var wg sync.WaitGroup
	errCh := make(chan error, len(list))

	for i, item := range list {
		wg.Add(1)
		go func(idx int, it any) {
			defer wg.Done()
			itemInputs := make(map[string]any, len(inputs))
			for k, v := range inputs {
				itemInputs[k] = v
			}
			itemInputs[listKey] = it
			out, err := runWithRetry(ctx, n, itemInputs)
			if err != nil {
				errCh <- err
				return
			}
			keep, _ := out["__keep"].(bool)
			results[idx] = result{idx: idx, keep: keep}
		}(i, item)
	}

	wg.Wait()
	close(errCh)
	if err := <-errCh; err != nil {
		return nil, err
	}

	var filtered []any
	for _, r := range results {
		if r.keep {
			filtered = append(filtered, list[r.idx])
		}
	}

	out := make(map[string]any)
	if len(n.Produces) > 0 {
		out[n.Produces[0]] = filtered
	}
	return out, nil
}

// runFlatMap runs n.Run per item; each call returns []any, results are concatenated.
func runFlatMap(ctx context.Context, n *Node, inputs map[string]any) (map[string]any, error) {
	listKey, list, ok := findListInput(inputs)
	if !ok {
		return nil, fmt.Errorf("flat-map node %q: no []any input found", n.ID)
	}

	expanded := make([][]any, len(list))
	var wg sync.WaitGroup
	errCh := make(chan error, len(list))

	for i, item := range list {
		wg.Add(1)
		go func(idx int, it any) {
			defer wg.Done()
			itemInputs := make(map[string]any, len(inputs))
			for k, v := range inputs {
				itemInputs[k] = v
			}
			itemInputs[listKey] = it
			out, err := runWithRetry(ctx, n, itemInputs)
			if err != nil {
				errCh <- err
				return
			}
			for _, v := range out {
				if sub, ok := v.([]any); ok {
					expanded[idx] = sub
				}
				break
			}
		}(i, item)
	}

	wg.Wait()
	close(errCh)
	if err := <-errCh; err != nil {
		return nil, err
	}

	var flat []any
	for _, sub := range expanded {
		flat = append(flat, sub...)
	}

	out := make(map[string]any)
	if len(n.Produces) > 0 {
		out[n.Produces[0]] = flat
	}
	return out, nil
}

// runReduce folds a []any sequentially using n.Run with __accumulator.
func runReduce(ctx context.Context, n *Node, inputs map[string]any) (map[string]any, error) {
	listKey, list, ok := findListInput(inputs)
	if !ok {
		return nil, fmt.Errorf("reduce node %q: no []any input found", n.ID)
	}

	var acc any
	for _, item := range list {
		itemInputs := make(map[string]any, len(inputs))
		for k, v := range inputs {
			itemInputs[k] = v
		}
		itemInputs[listKey] = item
		itemInputs["__accumulator"] = acc
		out, err := runWithRetry(ctx, n, itemInputs)
		if err != nil {
			return nil, err
		}
		if v, ok := out["__accumulator"]; ok {
			acc = v
		}
	}

	out := make(map[string]any)
	if len(n.Produces) > 0 {
		out[n.Produces[0]] = acc
	}
	return out, nil
}

// runBatch chunks a []any into batches of Config.BatchSize, runs each in parallel.
func runBatch(ctx context.Context, n *Node, inputs map[string]any) (map[string]any, error) {
	listKey, list, ok := findListInput(inputs)
	if !ok {
		return nil, fmt.Errorf("batch node %q: no []any input found", n.ID)
	}

	batchSize := 1
	if n.Config != nil && n.Config.BatchSize > 0 {
		batchSize = n.Config.BatchSize
	}

	// Chunk
	var chunks [][]any
	for i := 0; i < len(list); i += batchSize {
		end := i + batchSize
		if end > len(list) {
			end = len(list)
		}
		chunks = append(chunks, list[i:end])
	}

	chunkResults := make([][]any, len(chunks))
	var wg sync.WaitGroup
	errCh := make(chan error, len(chunks))

	for i, chunk := range chunks {
		wg.Add(1)
		go func(idx int, c []any) {
			defer wg.Done()
			chunkInputs := make(map[string]any, len(inputs))
			for k, v := range inputs {
				chunkInputs[k] = v
			}
			chunkInputs[listKey] = any(c)
			out, err := runWithRetry(ctx, n, chunkInputs)
			if err != nil {
				errCh <- err
				return
			}
			for _, v := range out {
				if sub, ok := v.([]any); ok {
					chunkResults[idx] = sub
				}
				break
			}
		}(i, chunk)
	}

	wg.Wait()
	close(errCh)
	if err := <-errCh; err != nil {
		return nil, err
	}

	var flat []any
	for _, sub := range chunkResults {
		flat = append(flat, sub...)
	}

	out := make(map[string]any)
	if len(n.Produces) > 0 {
		out[n.Produces[0]] = flat
	}
	return out, nil
}

// runZip pairs two []any inputs element-wise. No Run function needed.
// Uses Consumes order to determine which list is first vs second.
func runZip(n *Node, inputs map[string]any) (map[string]any, error) {
	var lists [][]any
	for _, key := range n.Consumes {
		if v, ok := inputs[key]; ok {
			if l, ok := v.([]any); ok {
				lists = append(lists, l)
			}
		}
		if len(lists) == 2 {
			break
		}
	}
	if len(lists) < 2 {
		return nil, fmt.Errorf("zip node %q: requires 2 []any inputs, got %d", n.ID, len(lists))
	}

	minLen := len(lists[0])
	if len(lists[1]) < minLen {
		minLen = len(lists[1])
	}

	pairs := make([]any, minLen)
	for i := 0; i < minLen; i++ {
		pairs[i] = []any{lists[0][i], lists[1][i]}
	}

	out := make(map[string]any)
	if len(n.Produces) > 0 {
		out[n.Produces[0]] = pairs
	}
	return out, nil
}

// --- Control-flow handlers ---

// runGate runs n.Run; if output contains __halt=true, returns a haltError.
func runGate(ctx context.Context, n *Node, inputs map[string]any) (map[string]any, error) {
	out, err := runWithRetry(ctx, n, inputs)
	if err != nil {
		return nil, err
	}
	if halt, _ := out["__halt"].(bool); halt {
		return out, &HaltError{NodeID: n.ID, Outputs: out}
	}
	return out, nil
}

// HaltError signals that a KindGate node halted the pipeline.
type HaltError struct {
	NodeID  string
	Outputs map[string]any
}

func (e *HaltError) Error() string {
	return fmt.Sprintf("pipeline halted by gate node %q", e.NodeID)
}

// runFallback tries the primary Run, then Config.Fallbacks in order.
func runFallback(ctx context.Context, _ *DAG, n *Node, inputs map[string]any) (map[string]any, error) {
	out, err := runWithRetry(ctx, n, inputs)
	if err == nil {
		return out, nil
	}

	if n.Config == nil {
		return nil, err
	}

	for _, fn := range n.Config.Fallbacks {
		out, err = fn(ctx, inputs)
		if err == nil {
			return out, nil
		}
	}

	return nil, fmt.Errorf("fallback node %q: all fallbacks exhausted: %w", n.ID, err)
}

// runRace runs n.Run and competitor functions in parallel; first success wins.
func runRace(ctx context.Context, _ *DAG, n *Node, inputs map[string]any) (map[string]any, error) {
	type raceResult struct {
		out map[string]any
		err error
	}

	raceCtx, raceCancel := context.WithCancel(ctx)
	defer raceCancel()

	if n.Config != nil && n.Config.RaceTimeout > 0 {
		var cancel context.CancelFunc
		raceCtx, cancel = context.WithTimeout(raceCtx, n.Config.RaceTimeout)
		defer cancel()
	}

	fns := []RunFunc{RunFunc(n.Run)}
	if n.Config != nil {
		fns = append(fns, n.Config.Competitors...)
	}

	resultCh := make(chan raceResult, len(fns))

	for _, fn := range fns {
		go func(f RunFunc) {
			out, err := f(raceCtx, inputs)
			resultCh <- raceResult{out: out, err: err}
		}(fn)
	}

	var lastErr error
	for i := 0; i < len(fns); i++ {
		select {
		case res := <-resultCh:
			if res.err == nil {
				raceCancel()
				return res.out, nil
			}
			lastErr = res.err
		case <-raceCtx.Done():
			return nil, fmt.Errorf("race node %q: %w", n.ID, raceCtx.Err())
		}
	}
	return nil, fmt.Errorf("race node %q: all competitors failed: %w", n.ID, lastErr)
}

// cacheStore is a global in-memory cache for KindCache nodes.
var cacheStore = struct {
	mu    sync.RWMutex
	items map[string]map[string]any
}{items: make(map[string]map[string]any)}

// runCache checks cache by key; on miss, runs n.Run and stores result.
func runCache(ctx context.Context, n *Node, inputs map[string]any) (map[string]any, error) {
	var key string
	if n.Config != nil && n.Config.CacheKeyFunc != nil {
		key = n.Config.CacheKeyFunc(inputs)
	} else {
		key = fmt.Sprintf("%s:%v", n.ID, inputs)
	}

	// Check cache
	cacheStore.mu.RLock()
	if cached, ok := cacheStore.items[key]; ok {
		cacheStore.mu.RUnlock()
		return cached, nil
	}
	cacheStore.mu.RUnlock()

	// Cache miss — run
	out, err := runWithRetry(ctx, n, inputs)
	if err != nil {
		return nil, err
	}

	// Store
	cacheStore.mu.Lock()
	cacheStore.items[key] = out
	cacheStore.mu.Unlock()

	return out, nil
}

// ClearCache clears the global cache used by KindCache nodes.
func ClearCache() {
	cacheStore.mu.Lock()
	cacheStore.items = make(map[string]map[string]any)
	cacheStore.mu.Unlock()
}

// makeSem returns a buffered channel semaphore, or nil if n <= 0 (unlimited).
func makeSem(n int) chan struct{} {
	if n <= 0 {
		return nil
	}
	return make(chan struct{}, n)
}
