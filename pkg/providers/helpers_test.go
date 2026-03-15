package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestHelperProcess is not a real test. It runs as a fake claude/codex subprocess
// when invoked with GO_WANT_HELPER_PROCESS=1. The FAKE_SCENARIO env var selects
// what output to emit.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	switch os.Getenv("FAKE_SCENARIO") {
	case "claude_success":
		fmt.Println(`{"type":"result","subtype":"success","is_error":false,"result":"hello world","session_id":"sess-abc","usage":{"input_tokens":10,"output_tokens":5}}`)
		os.Exit(0)
	case "claude_is_error":
		fmt.Println(`{"type":"result","subtype":"success","is_error":true,"result":"Rate limit exceeded.","session_id":"","usage":{"input_tokens":0,"output_tokens":0}}`)
		os.Exit(0)
	case "claude_session_expired":
		fmt.Println(`{"type":"result","subtype":"error_during_execution","is_error":true,"errors":["No conversation found with id old-session-123"],"session_id":"","usage":{"input_tokens":0,"output_tokens":0}}`)
		os.Exit(0)
	case "claude_exit_failure":
		os.Exit(1)
	case "codex_success":
		fmt.Println(`{"type":"thread.started","thread_id":"thread-xyz"}`)
		fmt.Println(`{"type":"item.completed","item":{"type":"agent_message","text":"I will help."}}`)
		fmt.Println(`{"type":"item.completed","item":{"type":"agent_message","text":"hello from codex"}}`)
		fmt.Println(`{"type":"turn.completed","usage":{"input_tokens":20,"output_tokens":8}}`)
		os.Exit(0)
	case "codex_stderr_failure":
		fmt.Fprintln(os.Stderr, "codex: authentication required")
		os.Exit(1)
	case "claude_idle_hang":
		// Emit nothing and block until killed by the idle timeout.
		time.Sleep(time.Hour)
	case "codex_idle_hang":
		// Emit nothing and block until killed by the idle timeout.
		time.Sleep(time.Hour)
	case "claude_idle_then_succeed":
		// On retry (detected via --resume flag in args), return a successful result.
		// On first call, emit a partial result event with a session_id then hang.
		isRetry := false
		for _, arg := range os.Args {
			if arg == "--resume" {
				isRetry = true
				break
			}
		}
		if isRetry {
			fmt.Println(`{"type":"result","subtype":"success","is_error":false,"result":"recovered after idle","session_id":"sess-recovered","usage":{"input_tokens":5,"output_tokens":3}}`)
			os.Exit(0)
		}
		// Emit a result event so the session_id is captured before hanging.
		fmt.Println(`{"type":"result","subtype":"success","is_error":false,"result":"","session_id":"sess-partial","usage":{"input_tokens":0,"output_tokens":0}}`)
		time.Sleep(time.Hour)
	case "codex_idle_then_succeed":
		// On retry (detected via "resume" subcommand in args), return a successful result.
		// On first call, emit a thread.started event with a thread_id then hang.
		isRetry := false
		for _, arg := range os.Args {
			if arg == "resume" {
				isRetry = true
				break
			}
		}
		if isRetry {
			fmt.Println(`{"type":"thread.started","thread_id":"thread-recovered"}`)
			fmt.Println(`{"type":"item.completed","item":{"type":"agent_message","text":"recovered after idle"}}`)
			fmt.Println(`{"type":"turn.completed","usage":{"input_tokens":5,"output_tokens":3}}`)
			os.Exit(0)
		}
		// Emit a thread.started event so the session_id is captured before hanging.
		fmt.Println(`{"type":"thread.started","thread_id":"thread-partial"}`)
		time.Sleep(time.Hour)
	default:
		fmt.Fprintf(os.Stderr, "unknown FAKE_SCENARIO=%q\n", os.Getenv("FAKE_SCENARIO"))
		os.Exit(2)
	}
}

// fakeInPath writes a shell script named binName into a temp dir and prepends
// that dir to PATH. The script invokes this test binary as a fake subprocess,
// emitting output for the given scenario.
func fakeInPath(t *testing.T, binName, scenario string) {
	t.Helper()
	dir := t.TempDir()
	// Use 'exec env' so the shell is replaced by the test binary directly.
	// Without 'exec', the shell forks the test binary as a child; killing the
	// shell leaves the child alive with the stdout pipe still open, which
	// deadlocks the idle-timeout tests that rely on killing the subprocess.
	script := fmt.Sprintf(
		"#!/bin/sh\nexec env GO_WANT_HELPER_PROCESS=1 FAKE_SCENARIO=%s %s -test.run=TestHelperProcess -- \"$@\"\n",
		scenario, os.Args[0],
	)
	if err := os.WriteFile(filepath.Join(dir, binName), []byte(script), 0o755); err != nil {
		t.Fatalf("write fake %s binary: %v", binName, err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
}

// sliceContains reports whether s appears in args.
func sliceContains(args []string, s string) bool {
	for _, a := range args {
		if a == s {
			return true
		}
	}
	return false
}

// sliceIndexOf returns the index of s in args, or -1.
func sliceIndexOf(args []string, s string) int {
	for i, a := range args {
		if a == s {
			return i
		}
	}
	return -1
}
