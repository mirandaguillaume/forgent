package spec

import (
	"fmt"
	"sort"
	"sync"

	"github.com/mirandaguillaume/forgent/pkg/model"
)

// TargetGenerator is the public interface for generating framework-specific output.
// Third parties can implement this interface to add new build targets.
type TargetGenerator interface {
	Target() string
	DefaultOutputDir() string
	GenerateSkill(skill model.SkillBehavior) string
	GenerateAgent(agent model.AgentComposition, skills []model.SkillBehavior, outputDir string) string
	GenerateInstructions(skills []model.SkillBehavior, agents []model.AgentComposition) *string
	SkillPath(name string) string
	AgentPath(name string) string
	InstructionsPath() *string
	ContextDir() string
}

// GeneratorFactory creates a new TargetGenerator instance.
type GeneratorFactory func() TargetGenerator

var (
	mu       sync.RWMutex
	registry = map[string]GeneratorFactory{}
)

// Register adds a generator factory for a build target.
func Register(name string, factory GeneratorFactory) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = factory
}

// Get returns a new TargetGenerator for the given target name.
func Get(name string) (TargetGenerator, error) {
	mu.RLock()
	defer mu.RUnlock()
	factory, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown build target: %q. Available targets: %v", name, availableLocked())
	}
	return factory(), nil
}

// Available returns sorted list of registered target names.
func Available() []string {
	mu.RLock()
	defer mu.RUnlock()
	return availableLocked()
}

func availableLocked() []string {
	keys := make([]string, 0, len(registry))
	for k := range registry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Reset clears the registry. Used only in tests.
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	registry = map[string]GeneratorFactory{}
}
