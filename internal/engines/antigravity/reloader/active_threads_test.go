package reloader

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestScanActiveConversations_ThresholdFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now()

	// Thread A: valid UUID, mtime = now - 1m -> should be included (< 5m)
	idA := "11111111-1111-4111-8111-111111111111"
	fileA := filepath.Join(tmpDir, idA+".db")
	if err := os.WriteFile(fileA, []byte("data-a"), 0644); err != nil {
		t.Fatalf("failed writing fileA: %v", err)
	}
	_ = os.Chtimes(fileA, now.Add(-1*time.Minute), now.Add(-1*time.Minute))

	// Thread B: valid UUID, mtime = now - 3m -> should be included (< 5m)
	idB := "22222222-2222-4222-8222-222222222222"
	fileB := filepath.Join(tmpDir, idB+".db")
	if err := os.WriteFile(fileB, []byte("data-b"), 0644); err != nil {
		t.Fatalf("failed writing fileB: %v", err)
	}
	_ = os.Chtimes(fileB, now.Add(-3*time.Minute), now.Add(-3*time.Minute))

	// Thread C: valid UUID, .db mtime = now - 10m, but .db-wal mtime = now - 30s -> should be included (MAX logic, newest)
	idC := "33333333-3333-4333-8333-333333333333"
	fileC := filepath.Join(tmpDir, idC+".db")
	fileCWal := filepath.Join(tmpDir, idC+".db-wal")
	if err := os.WriteFile(fileC, []byte("data-c"), 0644); err != nil {
		t.Fatalf("failed writing fileC: %v", err)
	}
	_ = os.Chtimes(fileC, now.Add(-10*time.Minute), now.Add(-10*time.Minute))
	if err := os.WriteFile(fileCWal, []byte("wal-c"), 0644); err != nil {
		t.Fatalf("failed writing fileCWal: %v", err)
	}
	_ = os.Chtimes(fileCWal, now.Add(-30*time.Second), now.Add(-30*time.Second))

	// Thread D: valid UUID, mtime = now - 10m -> should be excluded (> 5m)
	idD := "44444444-4444-4444-8444-444444444444"
	fileD := filepath.Join(tmpDir, idD+".db")
	if err := os.WriteFile(fileD, []byte("data-d"), 0644); err != nil {
		t.Fatalf("failed writing fileD: %v", err)
	}
	_ = os.Chtimes(fileD, now.Add(-10*time.Minute), now.Add(-10*time.Minute))

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

	activeThreads, err := ScanActiveConversations(tmpDir, "", now)
	if err != nil {
		t.Fatalf("ScanActiveConversations failed: %v", err)
	}

	if len(activeThreads) != 3 {
		t.Fatalf("Expected exactly 3 active threads, got %d: %+v", len(activeThreads), activeThreads)
	}

	// Ordering must be descending by effectiveMtime:
	// 1st: thread-c (effective = now - 30s)
	// 2nd: thread-a (effective = now - 1m)
	// 3rd: thread-b (effective = now - 3m)
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
	_ = os.Chtimes(f1, now.Add(-2*time.Minute), now.Add(-2*time.Minute))

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

func TestPruneActiveThreadsHistory_UnderLimit(t *testing.T) {
	// 1. Zero files
	tmpDir0 := t.TempDir()
	deleted, err := PruneActiveThreadsHistory(tmpDir0, 2)
	if err != nil {
		t.Fatalf("PruneActiveThreadsHistory failed on empty dir: %v", err)
	}
	if deleted != 0 {
		t.Errorf("Expected 0 deleted files for empty dir, got %d", deleted)
	}

	// 2. Exactly 1 file
	tmpDir1 := t.TempDir()
	f1 := filepath.Join(tmpDir1, "2026-10-21-10-00-00_active_threads.json")
	_ = os.WriteFile(f1, []byte("{}"), 0644)
	deleted, err = PruneActiveThreadsHistory(tmpDir1, 2)
	if err != nil {
		t.Fatalf("PruneActiveThreadsHistory failed on 1-file dir: %v", err)
	}
	if deleted != 0 {
		t.Errorf("Expected 0 deleted files for 1 file, got %d", deleted)
	}
	if _, err := os.Stat(f1); err != nil {
		t.Errorf("File f1 was unexpectedly deleted: %v", err)
	}

	// 3. Exactly 2 files
	tmpDir2 := t.TempDir()
	f2a := filepath.Join(tmpDir2, "2026-10-21-10-00-00_active_threads.json")
	f2b := filepath.Join(tmpDir2, "2026-10-21-11-00-00_active_threads.json")
	_ = os.WriteFile(f2a, []byte("{}"), 0644)
	_ = os.WriteFile(f2b, []byte("{}"), 0644)
	deleted, err = PruneActiveThreadsHistory(tmpDir2, 2)
	if err != nil {
		t.Fatalf("PruneActiveThreadsHistory failed on 2-file dir: %v", err)
	}
	if deleted != 0 {
		t.Errorf("Expected 0 deleted files for 2 files, got %d", deleted)
	}
	if _, err := os.Stat(f2a); err != nil {
		t.Errorf("File f2a was unexpectedly deleted: %v", err)
	}
	if _, err := os.Stat(f2b); err != nil {
		t.Errorf("File f2b was unexpectedly deleted: %v", err)
	}
}

func TestPruneActiveThreadsHistory_OverLimit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create 10 files with simulated timestamps from 2026-10-18 to 2026-10-21
	filenames := []string{
		"2026-10-18-08-00-00_active_threads.json",
		"2026-10-18-12-00-00_active_threads.json",
		"2026-10-19-09-00-00_active_threads.json",
		"2026-10-19-15-00-00_active_threads.json",
		"2026-10-20-10-00-00_active_threads.json",
		"2026-10-20-14-00-00_active_threads.json",
		"2026-10-20-18-00-00_active_threads.json",
		"2026-10-21-08-00-00_active_threads.json",
		"2026-10-21-10-00-00_active_threads.json", // 2nd newest
		"2026-10-21-12-00-00_active_threads.json", // 1st newest
	}

	for _, name := range filenames {
		p := filepath.Join(tmpDir, name)
		if err := os.WriteFile(p, []byte(`{"test":true}`), 0644); err != nil {
			t.Fatalf("Failed creating file %s: %v", name, err)
		}
	}

	// Also add an unrelated file and a subdirectory to ensure they are NOT touched
	unrelatedFile := filepath.Join(tmpDir, "readme.txt")
	_ = os.WriteFile(unrelatedFile, []byte("preserve me"), 0644)
	subDir := filepath.Join(tmpDir, "subfolder")
	_ = os.MkdirAll(subDir, 0755)

	deleted, err := PruneActiveThreadsHistory(tmpDir, 2)
	if err != nil {
		t.Fatalf("PruneActiveThreadsHistory failed: %v", err)
	}

	if deleted != 8 {
		t.Errorf("Expected exactly 8 deleted files, got %d", deleted)
	}

	// Verify the 2 newest files exist
	expectedRetained := []string{
		"2026-10-21-12-00-00_active_threads.json",
		"2026-10-21-10-00-00_active_threads.json",
	}
	for _, name := range expectedRetained {
		p := filepath.Join(tmpDir, name)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("Expected newest file %s to be retained, but got error: %v", name, err)
		}
	}

	// Verify the 8 older files were deleted
	for _, name := range filenames[:8] {
		p := filepath.Join(tmpDir, name)
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("Expected older file %s to be deleted, but it still exists", name)
		}
	}

	// Verify unrelated file and subfolder are untouched
	if _, err := os.Stat(unrelatedFile); err != nil {
		t.Errorf("Unrelated file %s should be preserved, but got: %v", unrelatedFile, err)
	}
	if fi, err := os.Stat(subDir); err != nil || !fi.IsDir() {
		t.Errorf("Subdirectory %s should be preserved", subDir)
	}
}

func TestPruneActiveThreadsHistory_NonExistentDir(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "does_not_exist_dir")
	deleted, err := PruneActiveThreadsHistory(nonExistent, 2)
	if err != nil {
		t.Fatalf("Expected err == nil for non-existent dir, got %v", err)
	}
	if deleted != 0 {
		t.Errorf("Expected 0 deleted for non-existent dir, got %d", deleted)
	}
}

func TestSessionReloader_PruneActiveThreadsHistory(t *testing.T) {
	tmpDir := t.TempDir()

	for i := 1; i <= 5; i++ {
		name := filepath.Join(tmpDir, "2026-10-21-0"+string(rune('0'+i))+"-00-00_active_threads.json")
		_ = os.WriteFile(name, []byte("{}"), 0644)
	}

	rel := NewSessionReloader()
	rel.OutputDir = tmpDir

	// keepCount <= 0 should default to DefaultHistoryKeepCount (2)
	deleted, err := rel.PruneActiveThreadsHistory(0)
	if err != nil {
		t.Fatalf("rel.PruneActiveThreadsHistory failed: %v", err)
	}
	if deleted != 3 {
		t.Errorf("Expected 3 deleted files with default keepCount, got %d", deleted)
	}

	// Check 2 remaining
	entries, _ := os.ReadDir(tmpDir)
	if len(entries) != 2 {
		t.Errorf("Expected 2 files remaining in tmpDir, got %d", len(entries))
	}
}

func TestDefaultThreadsHistoryDir_LogsPath(t *testing.T) {
	historyDir := GetDefaultThreadsHistoryDir()
	expectedSubpath := filepath.Join("logs", "active_threads_history")
	if !strings.HasSuffix(historyDir, expectedSubpath) {
		t.Fatalf("expected GetDefaultThreadsHistoryDir to end with %q, got %q", expectedSubpath, historyDir)
	}
	if fi, err := os.Stat(historyDir); err != nil || !fi.IsDir() {
		t.Fatalf("expected historyDir to exist as directory: %v", err)
	}
}

func TestScanActiveConversations_FilterKilledThreads(t *testing.T) {
	tmpDir := t.TempDir()
	convDir := filepath.Join(tmpDir, "conversations")
	_ = os.MkdirAll(convDir, 0755)
	dbPath := filepath.Join(tmpDir, "conversation_summaries.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed opening sqlite: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE conversation_summaries (
		conversation_id TEXT PRIMARY KEY,
		workspace_uris TEXT NOT NULL DEFAULT "",
		parent_conversation_id TEXT NOT NULL DEFAULT "",
		killed INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("failed creating table: %v", err)
	}

	now := time.Now()

	// Thread Alive: killed = 0, mtime = now - 1m
	idAlive := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	fileAlive := filepath.Join(convDir, idAlive+".db")
	_ = os.WriteFile(fileAlive, []byte("alive"), 0644)
	_ = os.Chtimes(fileAlive, now.Add(-1*time.Minute), now.Add(-1*time.Minute))
	_, _ = db.Exec(`INSERT INTO conversation_summaries (conversation_id, workspace_uris, parent_conversation_id, killed)
		VALUES (?, ?, ?, ?)`, idAlive, `["file:///C:/AliveProject"]`, "", 0)

	// Thread Dead: killed = 1, mtime = now - 1m (very fresh, but killed)
	idDead := "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	fileDead := filepath.Join(convDir, idDead+".db")
	_ = os.WriteFile(fileDead, []byte("dead"), 0644)
	_ = os.Chtimes(fileDead, now.Add(-1*time.Minute), now.Add(-1*time.Minute))
	_, _ = db.Exec(`INSERT INTO conversation_summaries (conversation_id, workspace_uris, parent_conversation_id, killed)
		VALUES (?, ?, ?, ?)`, idDead, `["file:///C:/DeadProject"]`, "", 1)

	threads, err := ScanActiveConversations(convDir, dbPath, now)
	if err != nil {
		t.Fatalf("ScanActiveConversations failed: %v", err)
	}

	if len(threads) != 1 {
		t.Fatalf("Expected exactly 1 thread (killed filtered out), got %d: %+v", len(threads), threads)
	}
	if threads[0].ThreadID != idAlive {
		t.Errorf("Expected alive thread %s, got %s", idAlive, threads[0].ThreadID)
	}
	if threads[0].Project != "C:/AliveProject" {
		t.Errorf("Expected project C:/AliveProject, got %s", threads[0].Project)
	}
}

func TestScanActiveConversations_HierarchicalOrder(t *testing.T) {
	tmpDir := t.TempDir()
	convDir := filepath.Join(tmpDir, "conversations")
	_ = os.MkdirAll(convDir, 0755)
	dbPath := filepath.Join(tmpDir, "conversation_summaries.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed opening sqlite: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE conversation_summaries (
		conversation_id TEXT PRIMARY KEY,
		workspace_uris TEXT NOT NULL DEFAULT "",
		parent_conversation_id TEXT NOT NULL DEFAULT "",
		killed INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("failed creating table: %v", err)
	}

	now := time.Now()

	parentID := "11111111-1111-4111-8111-111111111111"
	child1ID := "22222222-2222-4222-8222-222222222222"
	child2ID := "33333333-3333-4333-8333-333333333333"

	// Parent has mtime = now - 10m (> 5m window, would normally be excluded if standalone)
	fileP := filepath.Join(convDir, parentID+".db")
	_ = os.WriteFile(fileP, []byte("parent"), 0644)
	_ = os.Chtimes(fileP, now.Add(-10*time.Minute), now.Add(-10*time.Minute))
	_, _ = db.Exec(`INSERT INTO conversation_summaries (conversation_id, workspace_uris, parent_conversation_id, killed)
		VALUES (?, ?, ?, ?)`, parentID, `["file:///C:/ParentProject"]`, "", 0)

	// Child 1 has mtime = now - 1m (< 5m)
	fileC1 := filepath.Join(convDir, child1ID+".db")
	_ = os.WriteFile(fileC1, []byte("child1"), 0644)
	_ = os.Chtimes(fileC1, now.Add(-1*time.Minute), now.Add(-1*time.Minute))
	_, _ = db.Exec(`INSERT INTO conversation_summaries (conversation_id, workspace_uris, parent_conversation_id, killed)
		VALUES (?, ?, ?, ?)`, child1ID, `["file:///C:/Child1Project"]`, parentID, 0)

	// Child 2 has mtime = now - 3m (< 5m)
	fileC2 := filepath.Join(convDir, child2ID+".db")
	_ = os.WriteFile(fileC2, []byte("child2"), 0644)
	_ = os.Chtimes(fileC2, now.Add(-3*time.Minute), now.Add(-3*time.Minute))
	_, _ = db.Exec(`INSERT INTO conversation_summaries (conversation_id, workspace_uris, parent_conversation_id, killed)
		VALUES (?, ?, ?, ?)`, child2ID, `["file:///C:/Child2Project"]`, parentID, 0)

	threads, err := ScanActiveConversations(convDir, dbPath, now)
	if err != nil {
		t.Fatalf("ScanActiveConversations failed: %v", err)
	}

	if len(threads) != 3 {
		t.Fatalf("Expected exactly 3 threads (2 children + 1 discovered parent), got %d: %+v", len(threads), threads)
	}

	// Order must be: Children first (by mtime desc), then Parent last
	// 1st: Child 1 (mtime -1m)
	// 2nd: Child 2 (mtime -3m)
	// 3rd: Parent (discovered via child, placed last)
	if threads[0].ThreadID != child1ID {
		t.Errorf("Expected 1st thread to be child1 %s, got %s", child1ID, threads[0].ThreadID)
	}
	if !threads[0].IsMainPane {
		t.Errorf("Expected 1st thread to have IsMainPane = true")
	}
	if threads[0].Project != "C:/Child1Project" {
		t.Errorf("Expected child1 project C:/Child1Project, got %s", threads[0].Project)
	}

	if threads[1].ThreadID != child2ID {
		t.Errorf("Expected 2nd thread to be child2 %s, got %s", child2ID, threads[1].ThreadID)
	}
	if threads[1].IsMainPane {
		t.Errorf("Expected 2nd thread to have IsMainPane = false")
	}

	if threads[2].ThreadID != parentID {
		t.Errorf("Expected 3rd thread to be parent %s, got %s", parentID, threads[2].ThreadID)
	}
	if threads[2].IsMainPane {
		t.Errorf("Expected 3rd thread to have IsMainPane = false")
	}
	if threads[2].Project != "C:/ParentProject" {
		t.Errorf("Expected parent project C:/ParentProject, got %s", threads[2].Project)
	}
}

func TestScanActiveConversations_ParentNotPromotedToStandalone(t *testing.T) {
	tmpDir := t.TempDir()
	convDir := filepath.Join(tmpDir, "conversations")
	_ = os.MkdirAll(convDir, 0755)
	dbPath := filepath.Join(tmpDir, "conversation_summaries.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed opening sqlite: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE conversation_summaries (
		conversation_id TEXT PRIMARY KEY,
		workspace_uris TEXT NOT NULL DEFAULT "",
		parent_conversation_id TEXT NOT NULL DEFAULT "",
		killed INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("failed creating table: %v", err)
	}

	now := time.Now()

	parentID := "11111111-1111-4111-8111-111111111111"
	childID := "22222222-2222-4222-8222-222222222222"
	standaloneID := "33333333-3333-4333-8333-333333333333"

	// Parent has mtime = now - 1m (very fresh, within 5m)
	fileP := filepath.Join(convDir, parentID+".db")
	_ = os.WriteFile(fileP, []byte("parent"), 0644)
	_ = os.Chtimes(fileP, now.Add(-1*time.Minute), now.Add(-1*time.Minute))
	_, _ = db.Exec(`INSERT INTO conversation_summaries (conversation_id, workspace_uris, parent_conversation_id, killed)
		VALUES (?, ?, ?, ?)`, parentID, `["file:///C:/ParentProject"]`, "", 0)

	// Child has mtime = now - 2m
	fileC := filepath.Join(convDir, childID+".db")
	_ = os.WriteFile(fileC, []byte("child"), 0644)
	_ = os.Chtimes(fileC, now.Add(-2*time.Minute), now.Add(-2*time.Minute))
	_, _ = db.Exec(`INSERT INTO conversation_summaries (conversation_id, workspace_uris, parent_conversation_id, killed)
		VALUES (?, ?, ?, ?)`, childID, `["file:///C:/ChildProject"]`, parentID, 0)

	// Pure Standalone has mtime = now - 3m
	fileS := filepath.Join(convDir, standaloneID+".db")
	_ = os.WriteFile(fileS, []byte("standalone"), 0644)
	_ = os.Chtimes(fileS, now.Add(-3*time.Minute), now.Add(-3*time.Minute))
	_, _ = db.Exec(`INSERT INTO conversation_summaries (conversation_id, workspace_uris, parent_conversation_id, killed)
		VALUES (?, ?, ?, ?)`, standaloneID, `["file:///C:/StandaloneProject"]`, "", 0)

	threads, err := ScanActiveConversations(convDir, dbPath, now)
	if err != nil {
		t.Fatalf("ScanActiveConversations failed: %v", err)
	}

	if len(threads) != 3 {
		t.Fatalf("Expected exactly 3 threads, got %d: %+v", len(threads), threads)
	}

	// Worker First invariant: Child -> Pure Standalone -> Parent
	// Parent must NOT be in Standalone group despite having mtime -1m (fresher than standalone)
	if threads[0].ThreadID != childID {
		t.Errorf("Expected 1st thread to be Child %s, got %s", childID, threads[0].ThreadID)
	}
	if threads[1].ThreadID != standaloneID {
		t.Errorf("Expected 2nd thread to be Standalone %s, got %s", standaloneID, threads[1].ThreadID)
	}
	if threads[2].ThreadID != parentID {
		t.Errorf("Expected 3rd thread to be Parent %s, got %s", parentID, threads[2].ThreadID)
	}

	// Verify Parent appears only once
	parentCount := 0
	for _, th := range threads {
		if th.ThreadID == parentID {
			parentCount++
		}
	}
	if parentCount != 1 {
		t.Errorf("Expected parent to appear exactly once, got %d", parentCount)
	}
}

func TestScanActiveConversations_FilterKilledParent(t *testing.T) {
	tmpDir := t.TempDir()
	convDir := filepath.Join(tmpDir, "conversations")
	_ = os.MkdirAll(convDir, 0755)
	dbPath := filepath.Join(tmpDir, "conversation_summaries.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed opening sqlite: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE conversation_summaries (
		conversation_id TEXT PRIMARY KEY,
		workspace_uris TEXT NOT NULL DEFAULT "",
		parent_conversation_id TEXT NOT NULL DEFAULT "",
		killed INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("failed creating table: %v", err)
	}

	now := time.Now()

	parentID := "11111111-1111-4111-8111-111111111111"
	childID := "22222222-2222-4222-8222-222222222222"

	// Parent is killed: killed = 1
	fileP := filepath.Join(convDir, parentID+".db")
	_ = os.WriteFile(fileP, []byte("parent"), 0644)
	_ = os.Chtimes(fileP, now.Add(-1*time.Minute), now.Add(-1*time.Minute))
	_, _ = db.Exec(`INSERT INTO conversation_summaries (conversation_id, workspace_uris, parent_conversation_id, killed)
		VALUES (?, ?, ?, ?)`, parentID, `["file:///C:/ParentProject"]`, "", 1)

	// Child is alive: killed = 0
	fileC := filepath.Join(convDir, childID+".db")
	_ = os.WriteFile(fileC, []byte("child"), 0644)
	_ = os.Chtimes(fileC, now.Add(-2*time.Minute), now.Add(-2*time.Minute))
	_, _ = db.Exec(`INSERT INTO conversation_summaries (conversation_id, workspace_uris, parent_conversation_id, killed)
		VALUES (?, ?, ?, ?)`, childID, `["file:///C:/ChildProject"]`, parentID, 0)

	threads, err := ScanActiveConversations(convDir, dbPath, now)
	if err != nil {
		t.Fatalf("ScanActiveConversations failed: %v", err)
	}

	if len(threads) != 1 {
		t.Fatalf("Expected exactly 1 thread (alive child only, killed parent excluded), got %d: %+v", len(threads), threads)
	}
	if threads[0].ThreadID != childID {
		t.Errorf("Expected alive child %s, got %s", childID, threads[0].ThreadID)
	}
}

func TestScanActiveConversations_Deduplication(t *testing.T) {
	tmpDir := t.TempDir()
	convDir := filepath.Join(tmpDir, "conversations")
	_ = os.MkdirAll(convDir, 0755)
	dbPath := filepath.Join(tmpDir, "conversation_summaries.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed opening sqlite: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE conversation_summaries (
		conversation_id TEXT PRIMARY KEY,
		workspace_uris TEXT NOT NULL DEFAULT "",
		parent_conversation_id TEXT NOT NULL DEFAULT "",
		killed INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("failed creating table: %v", err)
	}

	now := time.Now()

	parentID := "11111111-1111-4111-8111-111111111111"
	child1ID := "22222222-2222-4222-8222-222222222222"
	child2ID := "33333333-3333-4333-8333-333333333333"
	child3ID := "44444444-4444-4444-8444-444444444444"

	// Parent has mtime = now - 10m
	fileP := filepath.Join(convDir, parentID+".db")
	_ = os.WriteFile(fileP, []byte("parent"), 0644)
	_ = os.Chtimes(fileP, now.Add(-10*time.Minute), now.Add(-10*time.Minute))
	_, _ = db.Exec(`INSERT INTO conversation_summaries (conversation_id, workspace_uris, parent_conversation_id, killed)
		VALUES (?, ?, ?, ?)`, parentID, `["file:///C:/ParentProject"]`, "", 0)

	// Three children referencing the same parent
	children := []struct {
		id    string
		mtime time.Duration
	}{
		{child1ID, 1 * time.Minute},
		{child2ID, 2 * time.Minute},
		{child3ID, 3 * time.Minute},
	}

	for _, c := range children {
		f := filepath.Join(convDir, c.id+".db")
		_ = os.WriteFile(f, []byte("child"), 0644)
		_ = os.Chtimes(f, now.Add(-c.mtime), now.Add(-c.mtime))
		_, _ = db.Exec(`INSERT INTO conversation_summaries (conversation_id, workspace_uris, parent_conversation_id, killed)
			VALUES (?, ?, ?, ?)`, c.id, `["file:///C:/ChildProject"]`, parentID, 0)
	}

	threads, err := ScanActiveConversations(convDir, dbPath, now)
	if err != nil {
		t.Fatalf("ScanActiveConversations failed: %v", err)
	}

	if len(threads) != 4 {
		t.Fatalf("Expected exactly 4 threads (3 children + 1 deduplicated parent), got %d: %+v", len(threads), threads)
	}

	// Verify parent appears only once and at the end
	if threads[3].ThreadID != parentID {
		t.Errorf("Expected parent to be at index 3, got %s", threads[3].ThreadID)
	}

	parentOccurrences := 0
	for _, th := range threads {
		if th.ThreadID == parentID {
			parentOccurrences++
		}
	}
	if parentOccurrences != 1 {
		t.Errorf("Expected parent to appear exactly 1 time, got %d", parentOccurrences)
	}
}
