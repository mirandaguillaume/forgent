package experiment

import "sort"

// aggregateViolations walks the sandbox audit trail and returns a typed
// []NodeViolation slice sorted by (step declaration order, attempt number)
// for reproducible output.
//
// It is the single aggregation point for all four enforcement channels:
// file permission (10-3), output contract (10-4), network (10-5), and
// guardrail (10-6). Enforcement sites stay enforcement-only; only this
// function needs to know about NodeViolation.
func aggregateViolations(task TaskDefinition, sb *SandboxFS) []NodeViolation {
	if sb == nil {
		return nil
	}

	stepIndex := make(map[string]int, len(task.Steps))
	for i, s := range task.Steps {
		stepIndex[s.ID] = i
	}

	var violations []NodeViolation
	for _, e := range sb.Log() {
		if e.Allowed {
			continue
		}
		vt, prevented := classifyAction(e.Action)
		if vt == "" {
			continue
		}
		violations = append(violations, NodeViolation{
			NodeID:        e.StepID,
			ViolationType: vt,
			Attempted:     true,
			Prevented:     prevented,
			AttemptNumber: e.AttemptNumber,
			Reason:        e.Reason,
			Target:        e.Path,
		})
	}

	// Sort: step declaration order first, then attempt number.
	sort.SliceStable(violations, func(i, j int) bool {
		si := stepIndex[violations[i].NodeID]
		sj := stepIndex[violations[j].NodeID]
		if si != sj {
			return si < sj
		}
		return violations[i].AttemptNumber < violations[j].AttemptNumber
	})

	return violations
}

// classifyAction maps an AccessEntry.Action to a ViolationType constant and
// Prevented flag. Returns ("", false) for actions that don't represent
// violations (e.g. slot_clear which is always Allowed=true in practice).
//
// Canonical action values:
//
//	"read"/"write"/"delete" → file permission (prevented)
//	"slot_write"            → output contract (prevented)
//	"network"               → network proxy   (prevented)
//	"guardrail"             → guardrail check (post-hoc)
//	"loop_exceeded"         → loop bound      (post-hoc, story 10-7)
func classifyAction(action string) (violationType string, prevented bool) {
	switch action {
	case "read", "write", "delete":
		return ViolationTypeFilePermission, true
	case "slot_write":
		return ViolationTypeOutputContract, true
	case "network":
		return ViolationTypeNetwork, true
	case "guardrail":
		return ViolationTypeGuardrail, false
	case "loop_exceeded":
		return ViolationTypeLoopBoundExceeded, false
	case "call_budget_exceeded":
		return ViolationTypeCallBudgetExceeded, true
	default:
		return "", false
	}
}

// computeContractComplianceRate returns the fraction of steps (out of
// stepCount) that had zero violations on their first attempt
// (AttemptNumber == 1). Returns 1.0 for stepCount == 0 (defensive — no
// violations possible; Validate() already rejects zero-step tasks).
func computeContractComplianceRate(stepCount int, violations []NodeViolation) float64 {
	if stepCount == 0 {
		return 1.0
	}

	firstAttemptViolators := make(map[string]bool)
	for _, v := range violations {
		if v.AttemptNumber == 1 {
			firstAttemptViolators[v.NodeID] = true
		}
	}

	compliant := stepCount - len(firstAttemptViolators)
	if compliant < 0 {
		compliant = 0
	}
	return float64(compliant) / float64(stepCount)
}
