package model

import (
	"errors"
	"sync"
)

var ErrMutationBusy = errors.New("Codex operation in progress")
var ErrRecoveryRequired = errors.New("Codex recovery required")

// Coordinator is shared by OAuth, account commands and session transactions.
// Callers acquire it before their own state locks; store locks are innermost.
type Coordinator struct {
	sync.Mutex
	AttemptID        string
	RecoveryRequired bool
}

func (c *Coordinator) Check(attempt string) error {
	if c.RecoveryRequired {
		return ErrRecoveryRequired
	}
	if c.AttemptID != "" && c.AttemptID != attempt {
		return ErrMutationBusy
	}
	return nil
}
