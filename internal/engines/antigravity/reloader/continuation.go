package reloader

import (
	"context"
	"fmt"
	"strings"
	"time"

	"as-auto-switcher/internal/engines/antigravity/connector"
)

// DefaultContinuationPrompt matches Python core/state_resolver.py:42 parity.
const DefaultContinuationPrompt = "Continue if you still have unfinished work in progress. " +
	"If all tasks are already completed, reply with: Currently there is no work to do."

// ContinuationSummary encapsulates the per-thread outcome of session continuation.
type ContinuationSummary struct {
	TotalThreads   int      `json:"total_threads"`
	SuccessCount   int      `json:"success_count"`
	FailedCount    int      `json:"failed_count"`
	SuccessThreads []string `json:"success_threads"`
	Errors         []string `json:"errors"`
}

// PerformSessionContinuation executes in-flight session continuation on a fresh Language Server instance.
// For each thread in activeThreads, it dispatches SmartFocusConversation, LoadTrajectory,
// and SendUserCascadeMessage with a 3-second pacing delay between successive threads.
// Errors on any thread do not abort the sequence; remaining threads continue to be processed.
func (r *SessionReloader) PerformSessionContinuation(
	ctx context.Context,
	client *connector.Client,
	activeThreads []ActiveThread,
	promptText string,
) (*ContinuationSummary, error) {
	if r == nil {
		return defaultReloader.PerformSessionContinuation(ctx, client, activeThreads, promptText)
	}

	r.mu.RLock()
	runner := r.ContinuationRunner
	sleeper := r.Sleeper
	r.mu.RUnlock()

	if runner != nil {
		err := runner(ctx, client, activeThreads, promptText)
		if err != nil {
			return &ContinuationSummary{
				TotalThreads: len(activeThreads),
				FailedCount:  len(activeThreads),
				Errors:       []string{err.Error()},
			}, err
		}
		var succ []string
		for _, th := range activeThreads {
			succ = append(succ, th.ThreadID)
		}
		return &ContinuationSummary{
			TotalThreads:   len(activeThreads),
			SuccessCount:   len(activeThreads),
			SuccessThreads: succ,
		}, nil
	}

	if client == nil || len(activeThreads) == 0 {
		return &ContinuationSummary{}, nil
	}

	pText := strings.TrimSpace(promptText)
	if pText == "" {
		pText = DefaultContinuationPrompt
	}

	if sleeper == nil {
		sleeper = defaultSleep
	}

	summary := &ContinuationSummary{
		TotalThreads:   len(activeThreads),
		SuccessThreads: make([]string, 0, len(activeThreads)),
		Errors:         make([]string, 0),
	}

	// Auto-detect active model
	activeModel := client.DiscoverActiveAgentModel(ctx, "")

	threadsSent := 0
	for _, thread := range activeThreads {
		tID := strings.TrimSpace(thread.ThreadID)
		if tID == "" {
			continue
		}

		// Step 4: Send thread 1 -> sleep 3s -> send thread 2 -> sleep 3s -> ...
		if threadsSent > 0 {
			_ = sleeper(ctx, 3*time.Second)
		}

		// Individual thread isolation: errors do not interrupt the continuation loop
		var threadErr error
		if err := client.SmartFocusConversation(ctx, tID); err != nil {
			threadErr = fmt.Errorf("SmartFocus error: %w", err)
		}
		if err := client.LoadTrajectory(ctx, tID); err != nil && threadErr == nil {
			threadErr = fmt.Errorf("LoadTrajectory error: %w", err)
		}
		if err := client.SendUserCascadeMessage(ctx, tID, pText, activeModel, nil); err != nil && threadErr == nil {
			threadErr = fmt.Errorf("SendUserCascadeMessage error: %w", err)
		}

		if threadErr != nil {
			summary.FailedCount++
			summary.Errors = append(summary.Errors, fmt.Sprintf("Thread %s: %v", tID, threadErr))
		} else {
			summary.SuccessCount++
			summary.SuccessThreads = append(summary.SuccessThreads, tID)
		}

		threadsSent++
	}

	return summary, nil
}

// PerformSessionContinuation is the package-level entry point delegating to defaultReloader.
func PerformSessionContinuation(
	ctx context.Context,
	client *connector.Client,
	activeThreads []ActiveThread,
	promptText string,
) (*ContinuationSummary, error) {
	return defaultReloader.PerformSessionContinuation(ctx, client, activeThreads, promptText)
}

// RestoreActiveSessions is an alias for PerformSessionContinuation.
func RestoreActiveSessions(
	ctx context.Context,
	client *connector.Client,
	activeThreads []ActiveThread,
	promptText string,
) (*ContinuationSummary, error) {
	return defaultReloader.PerformSessionContinuation(ctx, client, activeThreads, promptText)
}
