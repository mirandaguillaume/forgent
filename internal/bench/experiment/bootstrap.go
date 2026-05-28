package experiment

import (
	"math/rand"
	"sort"
)

// BootstrapCI computes a 95% bootstrap confidence interval for the difference
// in resolve rate between two conditions (condB minus condA).
//
// resolved[i] == true means task i was resolved (patch applied + tests passed).
// n is the number of bootstrap resamples (10_000 is standard for 95% CI).
//
// Returns (lo, hi) such that there is 95% confidence the true delta is in [lo, hi].
// A CI that does not include 0 indicates statistical significance.
//
// Bootstrap resampling is done with replacement (paired: same index drawn for both
// conditions to account for task-level correlation between conditions).
func BootstrapCI(resolvedA, resolvedB []bool, n int) (lo, hi float64) {
	size := len(resolvedA)
	if n <= 0 || size == 0 || size != len(resolvedB) {
		return 0, 0
	}

	r := rand.New(rand.NewSource(42)) // fixed seed for reproducibility
	deltas := make([]float64, n)

	for i := range n {
		var sumA, sumB float64
		for range size {
			idx := r.Intn(size)
			if resolvedA[idx] {
				sumA++
			}
			if resolvedB[idx] {
				sumB++
			}
		}
		deltas[i] = (sumB - sumA) / float64(size)
	}

	sort.Float64s(deltas)

	// 2.5th and 97.5th percentiles (clamped to valid index range)
	lo = deltas[int(0.025*float64(n))]
	hi = deltas[min(int(0.975*float64(n)), n-1)]
	return lo, hi
}
