package providers

import "testing"

func TestErrProviderMessage_Error(t *testing.T) {
	err := &ErrProviderMessage{Message: "rate limit exceeded"}
	if err.Error() != "rate limit exceeded" {
		t.Errorf("Error() = %q, want %q", err.Error(), "rate limit exceeded")
	}
}
