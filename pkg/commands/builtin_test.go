package commands

import (
	"context"
	"strings"
	"testing"
)

func findDefinitionByName(t *testing.T, defs []Definition, name string) Definition {
	t.Helper()
	for _, def := range defs {
		if def.Name == name {
			return def
		}
	}
	t.Fatalf("missing /%s definition", name)
	return Definition{}
}

func TestBuiltinHelpHandler_ReturnsFormattedMessage(t *testing.T) {
	defs := BuiltinDefinitions()
	helpDef := findDefinitionByName(t, defs, "help")
	if helpDef.Handler == nil {
		t.Fatalf("/help handler should not be nil")
	}

	var reply string
	err := helpDef.Handler(context.Background(), Request{
		Text: "/help",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	}, nil)
	if err != nil {
		t.Fatalf("/help handler error: %v", err)
	}
	if !strings.Contains(reply, "/show [model|channels]") {
		t.Fatalf("/help reply missing /show usage, got %q", reply)
	}
	if !strings.Contains(reply, "/stop") {
		t.Fatalf("/help reply missing /stop, got %q", reply)
	}
	if strings.Contains(reply, "/list") {
		t.Fatalf("/help reply should not contain /list, got %q", reply)
	}
}

func TestBuiltinShowChannels_UsesGetEnabledChannels(t *testing.T) {
	rt := &Runtime{
		GetEnabledChannels: func() []string {
			return []string{"telegram", "slack"}
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/show channels",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/show channels: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "telegram") || !strings.Contains(reply, "slack") {
		t.Fatalf("/show channels reply=%q, want telegram and slack", reply)
	}
}
