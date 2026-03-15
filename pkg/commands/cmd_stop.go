package commands

import "context"

func stopCommand() Definition {
	return Definition{
		Name:        "stop",
		Description: "Cancel the in-progress response",
		Usage:       "/stop",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.CancelSession == nil {
				return req.Reply("No active session to stop.")
			}
			if rt.CancelSession() {
				return req.Reply("Stopped.")
			}
			return req.Reply("No active response to stop.")
		},
	}
}
