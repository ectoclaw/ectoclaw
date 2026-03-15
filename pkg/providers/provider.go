package providers

import "context"

// ErrProviderMessage is returned when the provider itself returns a human-readable error
// (e.g. rate limit, out of credits). The message is safe to forward directly to the user.
type ErrProviderMessage struct {
	Message string
}

func (e *ErrProviderMessage) Error() string { return e.Message }

// InvokeRequest describes a single invocation of a coding-agent CLI.
type InvokeRequest struct {
	LogKey       string // explicit log file key set by caller (e.g. "telegram_300185392", "heartbeat"); empty = no logging
	SessionKey   string // bridge session key used for session resume lookup (e.g. "telegram:300185392")
	SessionID    string // "" = new session; non-empty = resume existing
	SystemPrompt string // always sent
	UserMessage  string
	WorkDir      string
	Model        string // passed as --model; empty = let the CLI use its default
	Stateless    bool   // do not persist session state after invocation
}

// InvokeResult contains the output of a successful invocation.
type InvokeResult struct {
	SessionID string
	Text      string
	TokensIn  int
	TokensOut int
}

// Provider is the interface for coding-agent CLI backends.
type Provider interface {
	Invoke(ctx context.Context, req InvokeRequest) (InvokeResult, error)
	Name() string
}
