package connector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Native state updates are patches. Keep the small execution/error projection;
// never retain message bodies, file contents or the full trajectory.
type nativeAgentUpdate struct {
	ConversationID   string  `json:"conversationId"`
	Status           *string `json:"status"`
	ExecutorMetadata *struct {
		ExecutionID       string `json:"executionId"`
		TerminationReason string `json:"terminationReason"`
	} `json:"executorMetadata"`
	MainTrajectoryUpdate *struct {
		LastStepError json.RawMessage `json:"lastStepError"`
		StepsUpdate   *struct {
			TotalLength int `json:"totalLength"`
		} `json:"stepsUpdate"`
	} `json:"mainTrajectoryUpdate"`
	ExecutionStatus json.RawMessage `json:"executionStatus"`
}

func decodeAgentStateStream(ctx context.Context, r io.Reader, out chan<- AgentStateUpdate) error {
	var state AgentStateUpdate
	for {
		frame, err := ReadConnectFrame(r)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if frame.Flags&ConnectFlagEndOfStream != 0 {
			var trailer struct {
				Error *struct {
					Code    string
					Message string
				} `json:"error"`
			}
			if err := json.Unmarshal(frame.Payload, &trailer); err != nil {
				return fmt.Errorf("invalid Connect trailer: %w", err)
			}
			if trailer.Error != nil {
				return fmt.Errorf("agent stream: %s: %s", trailer.Error.Code, trailer.Error.Message)
			}
			return nil
		}
		if frame.Flags != ConnectFlagMessage {
			return fmt.Errorf("unsupported Connect frame flags %d", frame.Flags)
		}
		var response struct {
			Update *nativeAgentUpdate `json:"update"`
		}
		if err := json.Unmarshal(frame.Payload, &response); err != nil {
			return fmt.Errorf("invalid agent state: %w", err)
		}
		if response.Update == nil {
			return errors.New("agent state frame has no update envelope")
		}
		u := response.Update
		if u.ConversationID != "" {
			if state.ConversationID != "" && state.ConversationID != u.ConversationID {
				state = AgentStateUpdate{}
			}
			state.ConversationID = u.ConversationID
		}
		if u.ExecutorMetadata != nil {
			if u.ExecutorMetadata.ExecutionID != "" && u.ExecutorMetadata.ExecutionID != state.ExecutionID {
				state.LastStepError = nil
				state.StepID = ""
				state.RetryAt = ""
				state.Attempt, state.MaxAttempts = 0, 0
			}
			state.ExecutionID = u.ExecutorMetadata.ExecutionID
			state.TerminationReason = strings.TrimPrefix(u.ExecutorMetadata.TerminationReason, "EXECUTOR_TERMINATION_REASON_")
		}
		if u.Status != nil {
			state.Status = strings.TrimPrefix(*u.Status, "CASCADE_RUN_STATUS_")
			if state.Status == "RUNNING" {
				state.TerminationReason = ""
				state.LastStepError = nil
			}
		}
		if t := u.MainTrajectoryUpdate; t != nil {
			if t.StepsUpdate != nil {
				state.StepID = strconv.Itoa(t.StepsUpdate.TotalLength)
			}
			if len(t.LastStepError) > 0 {
				var detail StepError
				if err := json.Unmarshal(t.LastStepError, &detail); err != nil {
					return fmt.Errorf("invalid last step error: %w", err)
				}
				state.LastStepError = nil
				if detail.Code != "" || detail.Message != "" || detail.Type != "" {
					state.LastStepError = &detail
				}
			}
		}
		if len(u.ExecutionStatus) > 0 {
			var execution struct {
				ModelRequest *struct {
					Attempt     int    `json:"attempt"`
					MaxAttempts int    `json:"maxAttempts"`
					RetryAt     string `json:"retryAt"`
				} `json:"modelRequest"`
			}
			if err := json.Unmarshal(u.ExecutionStatus, &execution); err != nil {
				return fmt.Errorf("invalid execution status: %w", err)
			}
			state.RetryAt = ""
			state.Attempt, state.MaxAttempts = 0, 0
			if execution.ModelRequest != nil {
				state.Attempt, state.MaxAttempts, state.RetryAt = execution.ModelRequest.Attempt, execution.ModelRequest.MaxAttempts, execution.ModelRequest.RetryAt
			}
		}
		if state.Status == "IDLE" {
			state.RetryAt = ""
		}
		select {
		case out <- state:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
