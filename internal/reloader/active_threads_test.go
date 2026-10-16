package reloader

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCaptureActiveThreadsSnapshot_DictFormat(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "app_storage.json")
	brainDir := filepath.Join(tmpDir, "brain")
	historyDir := filepath.Join(tmpDir, "active_threads_history")

	// Create thread directories in brainDir matching Python test parity
	_ = os.MkdirAll(filepath.Join(brainDir, "thread-main-456"), 0755)
	_ = os.MkdirAll(filepath.Join(brainDir, "thread-secondary-123"), 0755)

	// Prepare mock app_storage.json
	storageData := map[string]interface{}{
		"aux-pane-v2-session.conversationPanes": map[string]interface{}{
			"secondary-pane": map[string]interface{}{
				"paneId": "pane-2",
				"tabs": []interface{}{
					map[string]interface{}{
						"cascadeId": "thread-secondary-123",
						"metadata": map[string]interface{}{
							"workingDirectory": "G:/antigravity_auto_switch",
						},
					},
					map[string]interface{}{
						"id":   "file__viewer_tab",
						"type": "fileView",
					},
				},
			},
			"main-pane": map[string]interface{}{
				"paneId": "pane-main",
				"tabs": []interface{}{
					map[string]interface{}{
						"cascadeId": "thread-main-456",
						"metadata": map[string]interface{}{
							"workspaceUri": "file:///C:/Projects/AlphaApp",
						},
					},
				},
			},
		},
	}

	rawBytes, err := json.MarshalIndent(storageData, "", "  ")
	if err != nil {
		t.Fatalf("Failed marshaling storage data: %v", err)
	}
	if err := os.WriteFile(storagePath, rawBytes, 0644); err != nil {
		t.Fatalf("Failed writing storage file: %v", err)
	}

	snapshotPath, activeThreads, err := CaptureActiveThreadsSnapshot(storagePath, brainDir, historyDir)
	if err != nil {
		t.Fatalf("CaptureActiveThreadsSnapshot failed: %v", err)
	}

	if snapshotPath == "" {
		t.Fatal("Expected non-empty snapshot path")
	}
	if _, err := os.Stat(snapshotPath); err != nil {
		t.Fatalf("Snapshot file does not exist on disk: %v", err)
	}

	// Must exclude fileView tab and have 2 threads
	if len(activeThreads) != 2 {
		t.Fatalf("Expected 2 active threads, got %d: %+v", len(activeThreads), activeThreads)
	}

	// Main pane thread MUST be first (is_main_pane = true)
	if !activeThreads[0].IsMainPane || activeThreads[0].ThreadID != "thread-main-456" {
		t.Errorf("Expected first thread to be main pane (thread-main-456), got: %+v", activeThreads[0])
	}
	if activeThreads[0].Project != "C:/Projects/AlphaApp" {
		t.Errorf("Expected project C:/Projects/AlphaApp, got: %s", activeThreads[0].Project)
	}

	// Secondary thread
	if activeThreads[1].IsMainPane || activeThreads[1].ThreadID != "thread-secondary-123" {
		t.Errorf("Expected second thread to be secondary pane, got: %+v", activeThreads[1])
	}
	if activeThreads[1].Project != "G:/antigravity_auto_switch" {
		t.Errorf("Expected project G:/antigravity_auto_switch, got: %s", activeThreads[1].Project)
	}

	// Verify persisted snapshot file content
	fileBytes, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("Failed reading snapshot file: %v", err)
	}
	var snapshot ActiveThreadsSnapshot
	if err := json.Unmarshal(fileBytes, &snapshot); err != nil {
		t.Fatalf("Failed unmarshaling snapshot JSON: %v", err)
	}
	if snapshot.Timestamp == "" {
		t.Error("Expected non-empty timestamp in snapshot JSON")
	}
	if len(snapshot.ActiveThreads) != 2 {
		t.Errorf("Expected 2 active threads in JSON, got %d", len(snapshot.ActiveThreads))
	}
}

func TestCaptureActiveThreadsSnapshot_ListFormat(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "app_storage.json")
	brainDir := filepath.Join(tmpDir, "brain")
	historyDir := filepath.Join(tmpDir, "active_threads_history")

	_ = os.MkdirAll(filepath.Join(brainDir, "list-thread-1"), 0755)
	_ = os.MkdirAll(filepath.Join(brainDir, "list-thread-2"), 0755)

	storageData := map[string]interface{}{
		"conversationPanes": []interface{}{
			map[string]interface{}{
				"paneId": "main-pane",
				"tabs": []interface{}{
					map[string]interface{}{
						"cascadeId":         "list-thread-1",
						"working_directory": "D:/workspace/repo",
					},
				},
			},
			map[string]interface{}{
				"paneId":    "child-pane",
				"cascadeId": "list-thread-2",
			},
		},
	}

	rawBytes, _ := json.Marshal(storageData)
	_ = os.WriteFile(storagePath, rawBytes, 0644)

	snapshotPath, activeThreads, err := CaptureActiveThreadsSnapshot(storagePath, brainDir, historyDir)
	if err != nil {
		t.Fatalf("CaptureActiveThreadsSnapshot failed: %v", err)
	}

	if _, err := os.Stat(snapshotPath); err != nil {
		t.Fatalf("Snapshot file was not created: %v", err)
	}

	if len(activeThreads) != 2 {
		t.Fatalf("Expected 2 active threads, got %d", len(activeThreads))
	}

	if !activeThreads[0].IsMainPane || activeThreads[0].ThreadID != "list-thread-1" {
		t.Errorf("Expected first thread to be list-thread-1, got %+v", activeThreads[0])
	}
}

func TestCaptureActiveThreadsSnapshot_BrainDirFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "app_storage.json")
	brainDir := filepath.Join(tmpDir, "brain")
	historyDir := filepath.Join(tmpDir, "active_threads_history")

	_ = os.MkdirAll(filepath.Join(brainDir, "thread-exists"), 0755)

	storageData := map[string]interface{}{
		"aux-pane-v2-session.conversationPanes": map[string]interface{}{
			"pane-1": map[string]interface{}{
				"paneId": "main",
				"tabs": []interface{}{
					map[string]interface{}{
						"cascadeId": "thread-exists",
					},
					map[string]interface{}{
						"cascadeId": "thread-does-not-exist",
					},
				},
			},
		},
	}

	rawBytes, _ := json.Marshal(storageData)
	_ = os.WriteFile(storagePath, rawBytes, 0644)

	_, activeThreads, err := CaptureActiveThreadsSnapshot(storagePath, brainDir, historyDir)
	if err != nil {
		t.Fatalf("CaptureActiveThreadsSnapshot failed: %v", err)
	}

	if len(activeThreads) != 1 {
		t.Fatalf("Expected 1 active thread after brainDir filtering, got %d", len(activeThreads))
	}
	if activeThreads[0].ThreadID != "thread-exists" {
		t.Errorf("Expected thread-exists, got %s", activeThreads[0].ThreadID)
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
