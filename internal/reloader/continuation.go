package reloader

import (
	"context"
	"strings"
	"time"

	"antigravity-switch/internal/connector"
)

// DefaultContinuationPrompt matches Python core/state_resolver.py:42 parity.
const DefaultContinuationPrompt = "Continue if you still have unfinished work in progress. " +
	"If all tasks are already completed, reply with: Currently there is no work to do."

// PerformSessionContinuation executes in-flight session continuation on a fresh Language Server instance.
// For each thread in activeThreads, it dispatches SmartFocusConversation, LoadTrajectory,
// and SendUserCascadeMessage with a 3-second pacing delay between successive threads.
func (r *SessionReloader) PerformSessionContinuation(
	ctx context.Context,
	client *connector.Client,
	activeThreads []ActiveThread,
	promptText string,
) error {
	if r == nil {
		return defaultReloader.PerformSessionContinuation(ctx, client, activeThreads, promptText)
	}

	r.mu.RLock()
	runner := r.ContinuationRunner
	sleeper := r.Sleeper
	r.mu.RUnlock()

	if runner != nil {
		return runner(ctx, client, activeThreads, promptText)
	}

	if client == nil || len(activeThreads) == 0 {
		return nil
	}

	pText := strings.TrimSpace(promptText)
	if pText == "" {
		pText = DefaultContinuationPrompt
	}

	if sleeper == nil {
		sleeper = defaultSleep
	}

	threadsSent := 0
	for _, thread := range activeThreads {
		tID := strings.TrimSpace(thread.ThreadID)
		if tID == "" {
			continue
		}

		if threadsSent > 0 {
			_ = sleeper(ctx, 3*time.Second)
		}

		_ = client.SmartFocusConversation(ctx, tID)
		_ = client.LoadTrajectory(ctx, tID)
		_ = client.SendUserCascadeMessage(ctx, tID, pText, "", nil)

		threadsSent++
	}

	return nil
}

// PerformSessionContinuation is the package-level entry point delegating to defaultReloader.
func PerformSessionContinuation(
	ctx context.Context,
	client *connector.Client,
	activeThreads []ActiveThread,
	promptText string,
) error {
	return defaultReloader.PerformSessionContinuation(ctx, client, activeThreads, promptText)
}

// RestoreActiveSessions is an alias for PerformSessionContinuation.
func RestoreActiveSessions(
	ctx context.Context,
	client *connector.Client,
	activeThreads []ActiveThread,
	promptText string,
) error {
	return defaultReloader.PerformSessionContinuation(ctx, client, activeThreads, promptText)
}
