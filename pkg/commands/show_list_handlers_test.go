package commands

import (
	"context"
	"strings"
	"testing"
)

func TestShowChannels_AllChannelsReturned(t *testing.T) {
	rt := &Runtime{
		GetEnabledChannels: func() []string {
			return []string{"telegram", "slack"}
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Channel: "telegram",
		Text:    "/show channels",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/show channels outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "telegram") || !strings.Contains(reply, "slack") {
		t.Fatalf("/show channels reply=%q, want both channels", reply)
	}
}

func TestShowChannels_HandledOnAllChannels(t *testing.T) {
	rt := &Runtime{
		GetEnabledChannels: func() []string {
			return []string{"telegram"}
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Channel: "whatsapp",
		Text:    "/show channels",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("whatsapp /show channels outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if res.Command != "show" {
		t.Fatalf("command=%q, want=%q", res.Command, "show")
	}
	if !strings.Contains(reply, "telegram") {
		t.Fatalf("reply=%q, want enabled channels content", reply)
	}
}

func TestUnknownCommand_Passthrough(t *testing.T) {
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), nil)

	passthrough := ex.Execute(context.Background(), Request{
		Channel: "whatsapp",
		Text:    "/foo",
	})
	if passthrough.Outcome != OutcomePassthrough {
		t.Fatalf("whatsapp /foo outcome=%v, want=%v", passthrough.Outcome, OutcomePassthrough)
	}
	if passthrough.Command != "foo" {
		t.Fatalf("whatsapp /foo command=%q, want=%q", passthrough.Command, "foo")
	}
}
