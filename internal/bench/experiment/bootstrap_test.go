package experiment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBootstrapCI_ZeroDelta(t *testing.T) {
	// Identical conditions → CI should straddle 0
	n := 100
	resolved := make([]bool, n)
	for i := range resolved {
		resolved[i] = i%2 == 0 // 50% resolve rate in both
	}
	lo, hi := BootstrapCI(resolved, resolved, 10_000)
	assert.LessOrEqual(t, lo, 0.0, "lower CI bound should be ≤ 0 for identical conditions")
	assert.GreaterOrEqual(t, hi, 0.0, "upper CI bound should be ≥ 0 for identical conditions")
}

func TestBootstrapCI_ClearPositiveDelta(t *testing.T) {
	// condB resolves 100%, condA resolves 0% → CI should be clearly > 0
	n := 50
	resolvedA := make([]bool, n) // all false
	resolvedB := make([]bool, n)
	for i := range resolvedB {
		resolvedB[i] = true // all true
	}
	lo, hi := BootstrapCI(resolvedA, resolvedB, 10_000)
	assert.Greater(t, lo, 0.5, "lower CI bound should be well above 0 for 100% vs 0%")
	assert.Greater(t, hi, 0.5, "upper CI bound should be above 0")
}

func TestBootstrapCI_EmptyInput(t *testing.T) {
	// Empty slices → (0, 0) without panic
	lo, hi := BootstrapCI(nil, nil, 10_000)
	assert.Equal(t, 0.0, lo)
	assert.Equal(t, 0.0, hi)
}

func TestBootstrapCI_ZeroN(t *testing.T) {
	// n=0 → (0, 0) without panic
	a := []bool{true, false}
	b := []bool{true, true}
	lo, hi := BootstrapCI(a, b, 0)
	assert.Equal(t, 0.0, lo)
	assert.Equal(t, 0.0, hi)
}

func TestBootstrapCI_MismatchedLength(t *testing.T) {
	// Mismatched lengths → (0, 0) without panic
	a := []bool{true, false}
	b := []bool{true}
	lo, hi := BootstrapCI(a, b, 10_000)
	assert.Equal(t, 0.0, lo)
	assert.Equal(t, 0.0, hi)
}

func TestBootstrapCI_Reproducible(t *testing.T) {
	// Same seed → same results every run
	a := []bool{true, false, true, false, false}
	b := []bool{true, true, true, false, true}
	lo1, hi1 := BootstrapCI(a, b, 1000)
	lo2, hi2 := BootstrapCI(a, b, 1000)
	assert.Equal(t, lo1, lo2)
	assert.Equal(t, hi1, hi2)
}
