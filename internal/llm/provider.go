package llm

// Provider abstracts an LLM API for text completion.
type Provider interface {
	Complete(prompt string) (string, error)
}
