package reloader

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestScanActiveConversations2h_ThresholdFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now()

	// Thread A: valid UUID, mtime = now - 30m -> should be included
	idA := "11111111-1111-4111-8111-111111111111"
	fileA := filepath.Join(tmpDir, idA+".db")
	if err := os.WriteFile(fileA, []byte("data-a"), 0644); err != nil {
		t.Fatalf("failed writing fileA: %v", err)
	}
	_ = os.Chtimes(fileA, now.Add(-30*time.Minute), now.Add(-30*time.Minute))

	// Thread B: valid UUID, mtime = now - 1h50m -> should be included
	idB := "22222222-2222-4222-8222-222222222222"
	fileB := filepath.Join(tmpDir, idB+".db")
	if err := os.WriteFile(fileB, []byte("data-b"), 0644); err != nil {
		t.Fatalf("failed writing fileB: %v", err)
	}
	_ = os.Chtimes(fileB, now.Add(-110*time.Minute), now.Add(-110*time.Minute))

	// Thread C: valid UUID, .db mtime = now - 3h, but .db-wal mtime = now - 10m -> should be included (MAX logic)
	idC := "33333333-3333-4333-8333-333333333333"
	fileC := filepath.Join(tmpDir, idC+".db")
	fileCWal := filepath.Join(tmpDir, idC+".db-wal")
	if err := os.WriteFile(fileC, []byte("data-c"), 0644); err != nil {
		t.Fatalf("failed writing fileC: %v", err)
	}
	_ = os.Chtimes(fileC, now.Add(-3*time.Hour), now.Add(-3*time.Hour))
	if err := os.WriteFile(fileCWal, []byte("wal-c"), 0644); err != nil {
		t.Fatalf("failed writing fileCWal: %v", err)
	}
	_ = os.Chtimes(fileCWal, now.Add(-10*time.Minute), now.Add(-10*time.Minute))

	// Thread D: valid UUID, mtime = now - 2h10m -> should be excluded
	idD := "44444444-4444-4444-8444-444444444444"
	fileD := filepath.Join(tmpDir, idD+".db")
	if err := os.WriteFile(fileD, []byte("data-d"), 0644); err != nil {
		t.Fatalf("failed writing fileD: %v", err)
	}
	_ = os.Chtimes(fileD, now.Add(-130*time.Minute), now.Add(-130*time.Minute))

	// F3: Non-UUID databases must be strictly rejected
	fileIndex := filepath.Join(tmpDir, "index.db")
	_ = os.WriteFile(fileIndex, []byte("index-data"), 0644)
	_ = os.Chtimes(fileIndex, now, now)

	fileBackup := filepath.Join(tmpDir, "backup-copy.db-wal")
	_ = os.WriteFile(fileBackup, []byte("backup-wal"), 0644)
	_ = os.Chtimes(fileBackup, now, now)

	fileSummaries := filepath.Join(tmpDir, "conversation_summaries.db")
	_ = os.WriteFile(fileSummaries, []byte("summary"), 0644)
	_ = os.Chtimes(fileSummaries, now, now)

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
	if activeThreads[0].ThreadID != idC {
		t.Errorf("Expected 1st thread to be %s, got %s", idC, activeThreads[0].ThreadID)
	}
	if !activeThreads[0].IsMainPane {
		t.Errorf("Expected 1st thread to have IsMainPane = true")
	}

	if activeThreads[1].ThreadID != idA {
		t.Errorf("Expected 2nd thread to be %s, got %s", idA, activeThreads[1].ThreadID)
	}
	if activeThreads[1].IsMainPane {
		t.Errorf("Expected 2nd thread to have IsMainPane = false")
	}

	if activeThreads[2].ThreadID != idB {
		t.Errorf("Expected 3rd thread to be %s, got %s", idB, activeThreads[2].ThreadID)
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
			{ThreadID: "11111111-1111-4111-8111-111111111111", Project: "C:/Projects/App1", IsMainPane: true},
			{ThreadID: "22222222-2222-4222-8222-222222222222", Project: "D:/Work/App2", IsMainPane: false},
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
	if readThreads[0].ThreadID != "11111111-1111-4111-8111-111111111111" || readThreads[0].Project != "C:/Projects/App1" || !readThreads[0].IsMainPane {
		t.Errorf("Mismatch on thread 0: %+v", readThreads[0])
	}
	if readThreads[1].ThreadID != "22222222-2222-4222-8222-222222222222" || readThreads[1].Project != "D:/Work/App2" || readThreads[1].IsMainPane {
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
	tID := "12345678-1234-4234-8234-123456789abc"
	f1 := filepath.Join(convDir, tID+".db")
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
	if len(activeThreads) != 1 || activeThreads[0].ThreadID != tID {
		t.Errorf("Unexpected active threads: %+v", activeThreads)
	}

	// Read it back via ReadActiveThreadsSnapshot
	diskThreads, err := ReadActiveThreadsSnapshot(snapshotPath)
	if err != nil {
		t.Fatalf("Failed reading snapshot back: %v", err)
	}
	if len(diskThreads) != 1 || diskThreads[0].ThreadID != tID {
		t.Errorf("Unexpected disk threads: %+v", diskThreads)
	}
}

func TestResolveProjectFromSummaries_F2_RealSQLiteTable(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "conversation_summaries.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed creating sqlite db: %v", err)
	}
	defer db.Close()

	// Create real schema matching Antigravity
	_, err = db.Exec(`CREATE TABLE conversation_summaries (
		conversation_id TEXT PRIMARY KEY,
		title TEXT NOT NULL DEFAULT "",
		preview TEXT NOT NULL DEFAULT "",
		step_count INTEGER NOT NULL DEFAULT 0,
		last_modified_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		workspace_uris TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT ""
	)`)
	if err != nil {
		t.Fatalf("Failed creating table: %v", err)
	}

	targetID := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	// Preview contains a misleading "Command Working Directory: D:/UnrelatedPreview"
	// But workspace_uris contains the genuine project "C:/ActualProject"
	misleadingPreview := "Log header\nCommand Working Directory: D:/UnrelatedPreview\nSome chat text"
	realWorkspaceURIs := `["file:///C:/ActualProject"]`

	_, err = db.Exec(`INSERT INTO conversation_summaries (conversation_id, title, preview, workspace_uris, status)
		VALUES (?, ?, ?, ?, ?)`, targetID, "Title", misleadingPreview, realWorkspaceURIs, "IDLE")
	if err != nil {
		t.Fatalf("Failed inserting row: %v", err)
	}

	got := resolveProjectFromSummaries(dbPath, targetID)
	expected := "C:/ActualProject"
	if got != expected {
		t.Fatalf("F2 check failed: expected %q, got %q (did it erroneously match the preview?)", expected, got)
	}

	// Non-existent thread should return ""
	gotEmpty := resolveProjectFromSummaries(dbPath, "00000000-0000-4000-8000-000000000000")
	if gotEmpty != "" {
		t.Fatalf("Expected empty string for non-existent thread, got %q", gotEmpty)
	}
}

func TestSupervisorStoragePathDoesNotReplaceConversationsDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	convDir := filepath.Join(tmpDir, "conversations")
	_ = os.MkdirAll(convDir, 0755)
	historyDir := filepath.Join(tmpDir, "active_threads_history")
	_ = os.MkdirAll(historyDir, 0755)

	// Valid conversation file
	tID := "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	convFile := filepath.Join(convDir, tID+".db")
	_ = os.WriteFile(convFile, []byte("data"), 0644)

	// An existing app_storage.json file
	storagePath := filepath.Join(tmpDir, "app_storage.json")
	_ = os.WriteFile(storagePath, []byte(`{"test":true}`), 0644)

	rel := NewSessionReloader()
	rel.StoragePath = storagePath
	rel.OutputDir = historyDir
	rel.ConversationsDir = convDir

	snapPath, threads, err := rel.CaptureActiveThreadsSnapshot("", "", "")
	if err != nil {
		t.Fatalf("CaptureActiveThreadsSnapshot failed when StoragePath was configured: %v", err)
	}
	if snapPath == "" {
		t.Fatalf("Expected non-empty snapPath")
	}
	if len(threads) != 1 || threads[0].ThreadID != tID {
		t.Fatalf("Expected 1 thread with ID %s, got: %+v", tID, threads)
	}
}
