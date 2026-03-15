package providers

import (
	"bufio"
	"context"
	"errors"
	"io"
	"time"

	"github.com/ectoclaw/ectoclaw/pkg/logger"
)

// errIdleTimeout is returned when no output arrives within the idle deadline.
var errIdleTimeout = errors.New("provider: no output received within idle timeout")

// scannerMaxTokenSize is the maximum size of a single JSON line read from the provider subprocess.
// The default bufio.Scanner limit is 64 KB, which is too small for tool output events that embed
// file contents. A 4 MB limit prevents ErrTooLong from causing a deadlock (the subprocess blocks
// writing to its stdout pipe while cmd.Wait() blocks waiting for the subprocess to exit).
const scannerMaxTokenSize = 4 * 1024 * 1024 // 4 MB

// scanWithIdleTimeout reads lines from r, sending each non-empty line to the returned channel.
// If idleTimeout > 0 and no line arrives within that duration, cancel is called (which kills
// the subprocess via exec.CommandContext) and the channel is closed. Callers detect the hang
// by checking runCtx.Err() != nil && parentCtx.Err() == nil after the channel closes.
//
// Two goroutines are used internally. bufio.Scanner.Scan() is a blocking call that cannot
// participate in a select directly, so a dedicated scanner goroutine owns it and forwards
// lines over rawCh. The managing goroutine races rawCh against the timer. After cancel() the
// managing goroutine drains rawCh; the scanner goroutine exits promptly once the subprocess
// stdout pipe is closed by exec.CommandContext.
func scanWithIdleTimeout(cancel context.CancelFunc, r io.Reader, idleTimeout time.Duration) <-chan string {
	lineCh := make(chan string, 8)
	go func() {
		defer close(lineCh)
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, scannerMaxTokenSize), scannerMaxTokenSize)

		// Fast path: no idle timeout configured — drain directly without timer overhead.
		if idleTimeout <= 0 {
			for scanner.Scan() {
				if line := scanner.Text(); line != "" {
					lineCh <- line
				}
			}
			if err := scanner.Err(); err != nil {
				logger.WarnCF("providers", "stdout scanner error", map[string]any{"error": err.Error()})
			}
			return
		}

		// rawCh bridges the blocking scanner goroutine to the timer select below.
		rawCh := make(chan string)
		go func() {
			defer close(rawCh)
			for scanner.Scan() {
				rawCh <- scanner.Text()
			}
			if err := scanner.Err(); err != nil {
				logger.WarnCF("providers", "stdout scanner error", map[string]any{"error": err.Error()})
			}
		}()

		timer := time.NewTimer(idleTimeout)
		defer timer.Stop()

		for {
			select {
			case line, ok := <-rawCh:
				if !ok {
					return // EOF or pipe closed
				}
				if line == "" {
					continue // blank lines don't reset the idle timer
				}
				// Reset the idle timer on each non-empty line received.
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(idleTimeout)
				lineCh <- line

			case <-timer.C:
				cancel()
				// Drain the scanner goroutine until it exits. exec.CommandContext kills
				// the subprocess when the context is done, which closes stdout and
				// unblocks scanner.Scan(), causing rawCh to close.
				for range rawCh {
				}
				return
			}
		}
	}()
	return lineCh
}
