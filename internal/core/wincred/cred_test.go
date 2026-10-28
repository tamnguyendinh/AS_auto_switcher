package wincred

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestRawCredentialRoundTripAndDelete(t *testing.T) {
	target := fmt.Sprintf("auto-switcher-core-test-%d-%d", os.Getpid(), time.Now().UnixNano())
	t.Cleanup(func() {
		if err := DeleteRaw(target); err != nil {
			t.Errorf("cleanup test credential: %v", err)
		}
	})
	if _, err := ReadRaw(target); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing credential: expected ErrNotFound, got %v", err)
	}
	if HasRaw(target) {
		t.Fatal("missing credential reported present")
	}
	blob := []byte{0, 1, 127, 128, 255, 0, 42}
	if err := WriteRaw(target, "raw-api-test", blob); err != nil {
		t.Fatalf("write credential: %v", err)
	}
	if !HasRaw(target) {
		t.Fatal("written credential reported absent")
	}
	got, err := ReadRaw(target)
	if err != nil || !bytes.Equal(got, blob) {
		t.Fatalf("binary roundtrip failed: %v", err)
	}
	// Returned bytes must remain independent of the Windows-owned buffer.
	got[0] = 99
	again, err := ReadRaw(target)
	if err != nil || !bytes.Equal(again, blob) {
		t.Fatalf("mutating returned bytes changed stored credential: %v", err)
	}
	for i := 0; i < 2; i++ {
		if err := DeleteRaw(target); err != nil {
			t.Fatalf("delete %d: %v", i, err)
		}
	}
	if HasRaw(target) {
		t.Fatal("deleted credential reported present")
	}
}

func TestRawCredentialValidation(t *testing.T) {
	for _, target := range []string{"", " \t"} {
		if _, err := ReadRaw(target); err == nil {
			t.Error("read must reject an empty target")
		}
		if err := WriteRaw(target, "test", []byte{1}); err == nil {
			t.Error("write must reject an empty target")
		}
		if err := DeleteRaw(target); err != nil {
			t.Errorf("empty-target delete must be idempotent: %v", err)
		}
	}
	if err := WriteRaw("auto-switcher-core-validation-test", "test", nil); !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("empty blob: expected ErrInvalidPayload, got %v", err)
	}
}
