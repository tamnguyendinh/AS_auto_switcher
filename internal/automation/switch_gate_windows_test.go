//go:build windows

package automation

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSwitchGate_ExecuteAndTryExecute(t *testing.T) {
	uniqueName := fmt.Sprintf("Local\\TestSwitchGate_Basic_%d", time.Now().UnixNano())
	gate := NewSwitchGate(uniqueName)

	if gate.MutexName() != uniqueName {
		t.Errorf("expected mutex name %s, got %s", uniqueName, gate.MutexName())
	}

	executed := false
	err := gate.Execute(context.Background(), func(wasAbandoned bool) error {
		executed = true
		if wasAbandoned {
			t.Errorf("expected wasAbandoned=false")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !executed {
		t.Errorf("action was not executed")
	}

	// Test TryExecute while gate is held
	gateHeld := make(chan struct{})
	releaseGate := make(chan struct{})

	go func() {
		_ = gate.Execute(context.Background(), func(wasAbandoned bool) error {
			close(gateHeld)
			<-releaseGate
			return nil
		})
	}()

	<-gateHeld

	// While gate is held, TryExecute should fail with ErrGateBusy
	ok, tryErr := gate.TryExecute(func(wasAbandoned bool) error {
		return nil
	})
	if ok {
		t.Errorf("expected TryExecute to fail while gate is held, got ok=true")
	}
	if !errors.Is(tryErr, ErrGateBusy) {
		t.Errorf("expected ErrGateBusy, got %v", tryErr)
	}

	// Release gate
	close(releaseGate)

	// Wait briefly and verify TryExecute now succeeds
	time.Sleep(100 * time.Millisecond)
	ok, tryErr = gate.TryExecute(func(wasAbandoned bool) error {
		return nil
	})
	if !ok || tryErr != nil {
		t.Errorf("expected TryExecute to succeed after gate release, got ok=%v err=%v", ok, tryErr)
	}
}

func TestSwitchGate_ContextCancellation(t *testing.T) {
	uniqueName := fmt.Sprintf("Local\\TestSwitchGate_Cancel_%d", time.Now().UnixNano())
	gate := NewSwitchGate(uniqueName)

	gateHeld := make(chan struct{})
	releaseGate := make(chan struct{})

	go func() {
		_ = gate.Execute(context.Background(), func(wasAbandoned bool) error {
			close(gateHeld)
			<-releaseGate
			return nil
		})
	}()

	<-gateHeld

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := gate.Execute(ctx, func(wasAbandoned bool) error {
		t.Errorf("should not be reached due to context timeout")
		return nil
	})
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
	if elapsed < 100*time.Millisecond {
		t.Errorf("expected timeout wait of at least 100ms, got %v", elapsed)
	}

	close(releaseGate)
}

func TestSwitchGate_ConcurrentInProcess(t *testing.T) {
	uniqueName := fmt.Sprintf("Local\\TestSwitchGate_Concurrent_%d", time.Now().UnixNano())
	gate := NewSwitchGate(uniqueName)

	const goroutines = 20
	var activeCount atomic.Int32
	var maxActive atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := gate.Execute(context.Background(), func(wasAbandoned bool) error {
				curr := activeCount.Add(1)
				for {
					oldMax := maxActive.Load()
					if curr <= oldMax || maxActive.CompareAndSwap(oldMax, curr) {
						break
					}
				}
				time.Sleep(5 * time.Millisecond)
				activeCount.Add(-1)
				return nil
			})
			if err != nil {
				t.Errorf("Execute error: %v", err)
			}
		}()
	}

	wg.Wait()

	if maxActive.Load() != 1 {
		t.Errorf("expected strictly 1 concurrent execution inside gate, got max=%d", maxActive.Load())
	}
}

// TestHelperProcessNamedMutexHolder is a helper test invoked as a separate Windows process.
func TestHelperProcessNamedMutexHolder(t *testing.T) {
	if os.Getenv("GO_WANT_SWITCH_GATE_HELPER") != "1" {
		return
	}
	mutexName := os.Getenv("TEST_GATE_MUTEX_NAME")
	if mutexName == "" {
		os.Exit(2)
	}

	gate := NewSwitchGate(mutexName)
	_ = gate.Execute(context.Background(), func(wasAbandoned bool) error {
		_, _ = os.Stdout.WriteString("GATE_HELD_READY\n")
		_ = os.Stdout.Sync()
		time.Sleep(1500 * time.Millisecond)
		return nil
	})
	os.Exit(0)
}

func TestSwitchGate_ProcessContention(t *testing.T) {
	uniqueName := fmt.Sprintf("Local\\TestSwitchGate_ProcContention_%d", time.Now().UnixNano())

	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcessNamedMutexHolder$", "-test.v")
	cmd.Env = append(os.Environ(),
		"GO_WANT_SWITCH_GATE_HELPER=1",
		fmt.Sprintf("TEST_GATE_MUTEX_NAME=%s", uniqueName),
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed creating stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed starting helper process: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	scanner := bufio.NewScanner(stdout)
	ready := false
	for scanner.Scan() {
		if scanner.Text() == "GATE_HELD_READY" {
			ready = true
			break
		}
	}

	if !ready {
		t.Fatalf("helper process did not output GATE_HELD_READY")
	}

	// Parent attempts TryExecute on the same named mutex; must fail because child holds it
	parentGate := NewSwitchGate(uniqueName)
	acquired, tryErr := parentGate.TryExecute(func(wasAbandoned bool) error {
		return nil
	})

	if acquired {
		t.Errorf("parent should not acquire mutex while child process is holding it")
	}
	if !errors.Is(tryErr, ErrGateBusy) {
		t.Errorf("expected ErrGateBusy, got %v", tryErr)
	}
}