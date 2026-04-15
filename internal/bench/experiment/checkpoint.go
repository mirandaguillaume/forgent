package experiment

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// TaskResultJSON is a JSON-safe projection of TaskResult (error → string).
type TaskResultJSON struct {
	TaskID         string       `json:"task_id"`
	Condition      string       `json:"condition"`
	Completed      bool         `json:"completed"`
	StepResults    []StepResult `json:"step_results"`
	OrderingScore  float64      `json:"ordering_score"`
	GuardrailScore float64      `json:"guardrail_score"`
	TokensUsed     int          `json:"tokens_used"`
	WallClockMs    int64        `json:"wall_clock_ms"`
	LLMCalls       int          `json:"llm_calls"`
	Error          string       `json:"error,omitempty"`

	// Tool-based tasks (Story 5-4)
	ActionMetrics *ActionMetrics `json:"action_metrics,omitempty"`
}

// ToResultJSON converts a TaskResult to its JSON-safe form.
func ToResultJSON(r TaskResult) TaskResultJSON {
	j := TaskResultJSON{
		TaskID:         r.TaskID,
		Condition:      r.Condition,
		Completed:      r.Completed,
		StepResults:    r.StepResults,
		OrderingScore:  r.OrderingScore,
		GuardrailScore: r.GuardrailScore,
		TokensUsed:     r.TokensUsed,
		WallClockMs:    r.WallClockMs,
		LLMCalls:       r.LLMCalls,
	}
	if r.Error != nil {
		j.Error = r.Error.Error()
	}
	if r.ActionMetrics != nil {
		am := *r.ActionMetrics
		j.ActionMetrics = &am
	}
	return j
}

// ToTaskResult converts a TaskResultJSON back to a TaskResult.
func (j TaskResultJSON) ToTaskResult() TaskResult {
	r := TaskResult{
		TaskID:         j.TaskID,
		Condition:      j.Condition,
		Completed:      j.Completed,
		StepResults:    j.StepResults,
		OrderingScore:  j.OrderingScore,
		GuardrailScore: j.GuardrailScore,
		TokensUsed:     j.TokensUsed,
		WallClockMs:    j.WallClockMs,
		LLMCalls:       j.LLMCalls,
	}
	if j.Error != "" {
		r.Error = fmt.Errorf("%s", j.Error)
	}
	if j.ActionMetrics != nil {
		am := *j.ActionMetrics
		r.ActionMetrics = &am
	}
	return r
}

// CheckpointEntry is one line in the JSONL checkpoint file.
type CheckpointEntry struct {
	TaskID    string         `json:"task_id"`
	Condition string         `json:"condition"`
	Run       int            `json:"run"`
	Result    TaskResultJSON `json:"result"`
}

// CheckpointWriter appends entries to a JSONL file (thread-safe).
type CheckpointWriter struct {
	f  *os.File
	mu sync.Mutex
}

// NewCheckpointWriter opens (or creates) the file at path for appending.
func NewCheckpointWriter(path string) (*CheckpointWriter, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open checkpoint file: %w", err)
	}
	return &CheckpointWriter{f: f}, nil
}

// Append serialises entry as a JSON line, writes it, and fsyncs.
func (w *CheckpointWriter) Append(entry CheckpointEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal checkpoint entry: %w", err)
	}
	data = append(data, '\n')

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.f.Write(data); err != nil {
		return fmt.Errorf("write checkpoint entry: %w", err)
	}
	if err := w.f.Sync(); err != nil {
		return fmt.Errorf("fsync checkpoint file: %w", err)
	}
	return nil
}

// Close closes the underlying file.
func (w *CheckpointWriter) Close() error {
	return w.f.Close()
}

// LoadCheckpoint reads all entries from a JSONL checkpoint file.
func LoadCheckpoint(path string) ([]CheckpointEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open checkpoint file: %w", err)
	}
	defer f.Close()

	var entries []CheckpointEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20) // 1MB max line size
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e CheckpointEntry
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("parse checkpoint line: %w", err)
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan checkpoint file: %w", err)
	}
	return entries, nil
}

// BuildSkipMap groups entries by condition+taskID and returns taskID → run count
// for the given condition.
func BuildSkipMap(entries []CheckpointEntry, condition string) map[string]int {
	counts := make(map[string]int)
	for _, e := range entries {
		if e.Condition == condition {
			counts[e.TaskID]++
		}
	}
	return counts
}

// ExtractResults converts entries for a condition back to []TaskResult.
func ExtractResults(entries []CheckpointEntry, condition string) []TaskResult {
	var results []TaskResult
	for _, e := range entries {
		if e.Condition == condition {
			results = append(results, e.Result.ToTaskResult())
		}
	}
	return results
}
