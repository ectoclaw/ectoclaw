package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"

	"github.com/ectoclaw/ectoclaw/cmd/ectoclaw/internal"
	"github.com/ectoclaw/ectoclaw/pkg/bridge"
	"github.com/ectoclaw/ectoclaw/pkg/logger"
	"github.com/ectoclaw/ectoclaw/pkg/providers"
)

func agentCmd(message, sessionKey, model string, debug bool) error {
	if debug {
		logger.SetLevel(logger.DEBUG)
		logger.DebugCF("agent", "Debug mode enabled", nil)
	}

	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	if model != "" {
		cfg.Bridge.Model = model
	}

	provider, err := providers.NewProvider(cfg)
	if err != nil {
		return fmt.Errorf("error creating provider: %w", err)
	}

	sessionsPath := filepath.Join(cfg.WorkspacePath(), "sessions.json")
	sessions := bridge.NewSessions(sessionsPath)
	if err := sessions.Load(); err != nil {
		logger.WarnCF("agent", "Failed to load sessions", map[string]any{"error": err})
	}

	invoke := func(ctx context.Context, msg string) (string, error) {
		prompt, _ := bridge.AssembleSystemPrompt(cfg.WorkspacePath())
		existingID, _ := sessions.Get(sessionKey)

		result, err := provider.Invoke(ctx, providers.InvokeRequest{
			LogKey:       sessionKey,
			SessionKey:   sessionKey,
			SessionID:    existingID,
			SystemPrompt: prompt,
			UserMessage:  msg,
			WorkDir:      cfg.WorkspacePath(),
			Model:        cfg.Bridge.Model,
		})
		if err != nil {
			return "", err
		}

		if result.SessionID != "" {
			sessions.Set(sessionKey, result.SessionID)
			_ = sessions.Save()
		}

		text, _ := bridge.ParseOutput(result.Text)
		return strings.TrimSpace(text), nil
	}

	if message != "" {
		ctx := context.Background()
		response, err := invoke(ctx, message)
		if err != nil {
			return fmt.Errorf("error: %w", err)
		}
		fmt.Printf("\n%s %s\n", internal.Logo, response)
		return nil
	}

	fmt.Printf("%s Interactive mode (Ctrl+C to exit)\n\n", internal.Logo)
	interactiveMode(invoke, sessionKey)
	return nil
}

func interactiveMode(invoke func(context.Context, string) (string, error), sessionKey string) {
	prompt := fmt.Sprintf("%s You: ", internal.Logo)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          prompt,
		HistoryFile:     filepath.Join(os.TempDir(), ".ectoclaw_history"),
		HistoryLimit:    100,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Printf("Error initializing readline: %v\n", err)
		simpleInteractiveMode(invoke)
		return
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt || err == io.EOF {
				fmt.Println("\nGoodbye!")
				return
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}
		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			return
		}

		ctx := context.Background()
		response, err := invoke(ctx, input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		fmt.Printf("\n%s %s\n\n", internal.Logo, response)
	}
}

func simpleInteractiveMode(invoke func(context.Context, string) (string, error)) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("You: ")
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nGoodbye!")
				return
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}
		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			return
		}

		ctx := context.Background()
		response, err := invoke(ctx, input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		fmt.Printf("\n%s\n\n", response)
	}
}
