package reloader

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanActiveConversations2h_ThresholdFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now()

	// Thread A: mtime = now - 30m -> should be included
	fileA := filepath.Join(tmpDir, "thread-a.db")
	if err := os.WriteFile(fileA, []byte("data-a"), 0644); err != nil {
		t.Fatalf("failed writing fileA: %v", err)
	}
	_ = os.Chtimes(fileA, now.Add(-30*time.Minute), now.Add(-30*time.Minute))

	// Thread B: mtime = now - 1h50m -> should be included
	fileB := filepath.Join(tmpDir, "thread-b.db")
	if err := os.WriteFile(fileB, []byte("data-b"), 0644); err != nil {
		t.Fatalf("failed writing fileB: %v", err)
	}
	_ = os.Chtimes(fileB, now.Add(-110*time.Minute), now.Add(-110*time.Minute))

	// Thread C: .db mtime = now - 3h, but .db-wal mtime = now - 10m -> should be included (MAX logic)
	fileC := filepath.Join(tmpDir, "thread-c.db")
	fileCWal := filepath.Join(tmpDir, "thread-c.db-wal")
	if err := os.WriteFile(fileC, []byte("data-c"), 0644); err != nil {
		t.Fatalf("failed writing fileC: %v", err)
	}
	_ = os.Chtimes(fileC, now.Add(-3*time.Hour), now.Add(-3*time.Hour))
	if err := os.WriteFile(fileCWal, []byte("wal-c"), 0644); err != nil {
		t.Fatalf("failed writing fileCWal: %v", err)
	}
	_ = os.Chtimes(fileCWal, now.Add(-10*time.Minute), now.Add(-10*time.Minute))

	// Thread D: mtime = now - 2h10m -> should be excluded
	fileD := filepath.Join(tmpDir, "thread-d.db")
	if err := os.WriteFile(fileD, []byte("data-d"), 0644); err != nil {
		t.Fatalf("failed writing fileD: %v", err)
	}
	_ = os.Chtimes(fileD, now.Add(-130*time.Minute), now.Add(-130*time.Minute))

	// conversation_summaries.db (should be ignored by filter)
	fileIgnored := filepath.Join(tmpDir, "conversation_summaries.db")
	_ = os.WriteFile(fileIgnored, []byte("summary"), 0644)
	_ = os.Chtimes(fileIgnored, now, now)

	activeThreads, err := ScanActiveConversations2h(tmpDir, "", now)
	if err != nil {
		t.Fatalf("ScanActiveConversations2h failed: %v", err)
	}

	if len(activeThreads) != 3 {
		t.Fatalf("Expected exactly 3 active threads, got %d: %+v", len(activeThreads), activeThreads)
	}

	// Ordering must be descending by effectiveMtime:
	// 1st: thread-c (effective = now - 10m)
	// 2nd: thread-a (effective = now - 30m)
	// 3rd: thread-b (effective = now - 1h50m)
	if activeThreads[0].ThreadID != "thread-c" {
		t.Errorf("Expected 1st thread to be thread-c, got %s", activeThreads[0].ThreadID)
	}
	if !activeThreads[0].IsMainPane {
		t.Errorf("Expected 1st thread to have IsMainPane = true")
	}

	if activeThreads[1].ThreadID != "thread-a" {
		t.Errorf("Expected 2nd thread to be thread-a, got %s", activeThreads[1].ThreadID)
	}
	if activeThreads[1].IsMainPane {
		t.Errorf("Expected 2nd thread to have IsMainPane = false")
	}

	if activeThreads[2].ThreadID != "thread-b" {
		t.Errorf("Expected 3rd thread to be thread-b, got %s", activeThreads[2].ThreadID)
	}
	if activeThreads[2].IsMainPane {
		t.Errorf("Expected 3rd thread to have IsMainPane = false")
	}
}

func TestReadActiveThreadsSnapshot_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	snapshotFile := filepath.Join(tmpDir, "test_snapshot.json")

	original := ActiveThreadsSnapshot{
		Timestamp: "2026-10-18 12:00:00",
		ActiveThreads: []ActiveThread{
			{ThreadID: "t-1", Project: "C:/Projects/App1", IsMainPane: true},
			{ThreadID: "t-2", Project: "D:/Work/App2", IsMainPane: false},
		},
	}

	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("Failed marshaling: %v", err)
	}
	if err := os.WriteFile(snapshotFile, data, 0644); err != nil {
		t.Fatalf("Failed writing: %v", err)
	}

	readThreads, err := ReadActiveThreadsSnapshot(snapshotFile)
	if err != nil {
		t.Fatalf("ReadActiveThreadsSnapshot failed: %v", err)
	}

	if len(readThreads) != 2 {
		t.Fatalf("Expected 2 threads, got %d", len(readThreads))
	}
	if readThreads[0].ThreadID != "t-1" || readThreads[0].Project != "C:/Projects/App1" || !readThreads[0].IsMainPane {
		t.Errorf("Mismatch on thread 0: %+v", readThreads[0])
	}
	if readThreads[1].ThreadID != "t-2" || readThreads[1].Project != "D:/Work/App2" || readThreads[1].IsMainPane {
		t.Errorf("Mismatch on thread 1: %+v", readThreads[1])
	}

	// Test empty path error
	if _, err := ReadActiveThreadsSnapshot(""); err == nil {
		t.Errorf("Expected error for empty snapshotPath, got nil")
	}
}

func TestCaptureActiveThreadsSnapshot_SQLite(t *testing.T) {
	tmpDir := t.TempDir()
	convDir := filepath.Join(tmpDir, "conversations")
	_ = os.MkdirAll(convDir, 0755)
	historyDir := filepath.Join(tmpDir, "active_threads_history")

	now := time.Now()
	f1 := filepath.Join(convDir, "thread-snap-1.db")
	_ = os.WriteFile(f1, []byte("data1"), 0644)
	_ = os.Chtimes(f1, now.Add(-5*time.Minute), now.Add(-5*time.Minute))

	snapshotPath, activeThreads, err := CaptureActiveThreadsSnapshot(convDir, "", historyDir)
	if err != nil {
		t.Fatalf("CaptureActiveThreadsSnapshot failed: %v", err)
	}
	if snapshotPath == "" {
		t.Fatalf("Expected non-empty snapshotPath")
	}
	if _, err := os.Stat(snapshotPath); err != nil {
		t.Fatalf("Snapshot file does not exist on disk: %v", err)
	}
	if len(activeThreads) != 1 || activeThreads[0].ThreadID != "thread-snap-1" {
		t.Errorf("Unexpected active threads: %+v", activeThreads)
	}

	// Read it back via ReadActiveThreadsSnapshot
	diskThreads, err := ReadActiveThreadsSnapshot(snapshotPath)
	if err != nil {
		t.Fatalf("Failed reading snapshot back: %v", err)
	}
	if len(diskThreads) != 1 || diskThreads[0].ThreadID != "thread-snap-1" {
		t.Errorf("Unexpected disk threads: %+v", diskThreads)
	}
}

func TestExtractProjectFromText(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Active Workspaces pattern",
			input:    "Some text\nActive Workspaces:\n - G:/projects/antigravity\nNext line",
			expected: "G:/projects/antigravity",
		},
		{
			name:     "Command Working Directory pattern",
			input:    "Log header\nCommand Working Directory: C:\\repo\\backend\nFooter",
			expected: "C:\\repo\\backend",
		},
		{
			name:     "JSON Cwd pattern",
			input:    `{"status":"ok","Cwd":"D:\\work\\project"}`,
			expected: `D:\work\project`,
		},
		{
			name:     "File URI pattern",
			input:    `Workspace opened at file:///C:/Users/Tam/Workspace%201/repo`,
			expected: `C:/Users/Tam/Workspace 1/repo`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractProjectFromText(tc.input)
			if got != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestResolveProjectFromSummaries(t *testing.T) {
	tmpDir := t.TempDir()
	summariesDb := filepath.Join(tmpDir, "conversation_summaries.db")

	// Binary content containing thread-id and workspace uri
	content := []byte(`some header 12345678-abcd-1234-abcd-1234567890ab middle text ["file:///C:/Users/Tam/Workspace%20Proj"] trailing data`)
	if err := os.WriteFile(summariesDb, content, 0644); err != nil {
		t.Fatalf("Failed writing summaries db: %v", err)
	}

	got := resolveProjectFromSummaries(summariesDb, "12345678-abcd-1234-abcd-1234567890ab")
	expected := "C:/Users/Tam/Workspace Proj"
	if got != expected {
		t.Errorf("Expected %q, got %q", expected, got)
	}

	// Non-existent thread
	gotEmpty := resolveProjectFromSummaries(summariesDb, "non-existent-thread")
	if gotEmpty != "" {
		t.Errorf("Expected empty string for non-existent thread, got %q", gotEmpty)
	}
}
