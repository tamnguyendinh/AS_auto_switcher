package reloader

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"antigravity-switch/internal/store"
)

// ActiveThread represents a conversation thread extracted from Antigravity's layout state.
type ActiveThread struct {
	ThreadID   string `json:"thread_id"`
	Project    string `json:"project"`
	IsMainPane bool   `json:"is_main_pane"`
	Model      string `json:"model,omitempty"`
}

// ActiveThreadsSnapshot encapsulates the persisted state of active conversation threads.
type ActiveThreadsSnapshot struct {
	Timestamp     string         `json:"timestamp"`
	ActiveThreads []ActiveThread `json:"active_threads"`
}

var (
	reActiveWorkspaces = regexp.MustCompile(`(?i)Active Workspaces:\s*[
]+\s*-\s*([^
]+)`)
	reCommandCwd       = regexp.MustCompile(`(?i)Command Working Directory:\s*([^
]+)`)
	reJSONCwd          = regexp.MustCompile(`["']Cwd["']\s*:\s*["']([^"']+)["']`)
	reJSONProj         = regexp.MustCompile(`(?i)["'](?:project|workspace|workspace_path|workingDirectory|working_directory|active_workspace|activeWorkspace)["']\s*:\s*["']([^"']+)["']`)
	reFileURI          = regexp.MustCompile(`file:///([^
"'#]+)`)

	projectKeys = []string{
		"workingDirectory", "working_directory", "workspaceUri", "workspace_uri",
		"project", "workspace", "active_workspace", "activeWorkspace", "cwd", "Cwd",
	}
)

// GetDefaultThreadsHistoryDir returns the canonical accounts_store/active_threads_history directory.
func GetDefaultThreadsHistoryDir() string {
	storeDir := store.GetDefaultStoreDir()
	historyDir := filepath.Join(storeDir, "active_threads_history")
	_ = os.MkdirAll(historyDir, 0755)
	return historyDir
}

type threadCandidate struct {
	threadID       string
	dbMtime        time.Time
	walMtime       time.Time
	hasDb          bool
	hasWal         bool
	effectiveMtime time.Time
}

// ScanActiveConversations2h scans %USERPROFILE%/.gemini/antigravity/conversations/
// for all threads with MAX(mtime(.db), mtime(.db-wal)) >= startTime - 2h.
func ScanActiveConversations2h(conversationsDir string, summariesDbPath string, startTime time.Time) ([]ActiveThread, error) {
	cDir := conversationsDir
	if cDir == "" {
		home, _ := os.UserHomeDir()
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		if home != "" {
			cDir = filepath.Join(home, ".gemini", "antigravity", "conversations")
		}
	}

	threshold := startTime.Add(-2 * time.Hour)

	entries, err := os.ReadDir(cDir)
	if err != nil {
		return nil, fmt.Errorf("failed reading conversations dir: %w", err)
	}

	candidates := make(map[string]*threadCandidate)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		var tID string
		var isWal bool

		if strings.HasSuffix(name, ".db") {
			tID = strings.TrimSuffix(name, ".db")
		} else if strings.HasSuffix(name, ".db-wal") {
			tID = strings.TrimSuffix(name, ".db-wal")
			isWal = true
		} else {
			continue
		}

		tID = strings.TrimSpace(tID)
		if tID == "" || strings.EqualFold(tID, "conversation_summaries") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		cand, ok := candidates[tID]
		if !ok {
			cand = &threadCandidate{threadID: tID}
			candidates[tID] = cand
		}

		if isWal {
			cand.hasWal = true
			cand.walMtime = info.ModTime()
		} else {
			cand.hasDb = true
			cand.dbMtime = info.ModTime()
		}
	}

	var qualified []*threadCandidate
	for _, cand := range candidates {
		effective := cand.dbMtime
		if cand.hasWal && (cand.walMtime.After(effective) || !cand.hasDb) {
			effective = cand.walMtime
		}
		cand.effectiveMtime = effective

		if effective.Before(threshold) {
			continue
		}
		qualified = append(qualified, cand)
	}

	// Sort qualified by effectiveMtime descending (newest first)
	sort.Slice(qualified, func(i, j int) bool {
		return qualified[i].effectiveMtime.After(qualified[j].effectiveMtime)
	})

	result := make([]ActiveThread, 0, len(qualified))
	for i, cand := range qualified {
		proj := resolveProjectFromSummaries(summariesDbPath, cand.threadID)
		isMain := (i == 0)
		result = append(result, ActiveThread{
			ThreadID:   cand.threadID,
			Project:    proj,
			IsMainPane: isMain,
		})
	}

	return result, nil
}

func resolveProjectFromSummaries(summariesDbPath, threadID string) string {
	if threadID == "" {
		return ""
	}
	sDb := summariesDbPath
	if sDb == "" {
		home, _ := os.UserHomeDir()
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		if home != "" {
			sDb = filepath.Join(home, ".gemini", "antigravity", "conversation_summaries.db")
		}
	}

	if isFile(sDb) {
		if raw, err := os.ReadFile(sDb); err == nil {
			idBytes := []byte(threadID)
			idx := bytes.Index(raw, idBytes)
			if idx != -1 {
				end := idx + 2048
				if end > len(raw) {
					end = len(raw)
				}
				window := string(raw[idx:end])
				if extracted := extractProjectFromText(window); extracted != "" {
					return extracted
				}
			}
		}
	}

	home, _ := os.UserHomeDir()
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	if home != "" {
		brainDir := filepath.Join(home, ".gemini", "antigravity", "brain")
		if extracted := resolveThreadProject(threadID, nil, brainDir, nil); extracted != "" {
			return extracted
		}
	}

	return ""
}

// defaultCaptureActiveThreadsSnapshot extracts active conversation threads from
// conversations/*.db and *.db-wal with MAX(mtime) >= start - 2h, resolves projects,
// and saves an atomic snapshot JSON to outputDir.
func defaultCaptureActiveThreadsSnapshot(conversationsDir, summariesDbPath, outputDir string) (string, []ActiveThread, error) {
	start := time.Now()
	activeThreads, err := ScanActiveConversations2h(conversationsDir, summariesDbPath, start)
	if err != nil {
		return "", nil, err
	}

	outDir := outputDir
	if outDir == "" {
		outDir = GetDefaultThreadsHistoryDir()
	}
	_ = os.MkdirAll(outDir, 0755)

	timestampStr := start.Format("2006-01-02 15:04:05")
	fileTs := start.Format("2006-01-02-15-04-05")
	filename := fmt.Sprintf("%s_active_threads.json", fileTs)
	snapshotPath := filepath.Join(outDir, filename)

	snapshot := ActiveThreadsSnapshot{
		Timestamp:     timestampStr,
		ActiveThreads: activeThreads,
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", nil, err
	}

	tmpFile := snapshotPath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return "", nil, err
	}
	if err := os.Rename(tmpFile, snapshotPath); err != nil {
		_ = os.Remove(snapshotPath)
		if retryErr := os.Rename(tmpFile, snapshotPath); retryErr != nil {
			_ = os.Remove(tmpFile)
			return "", nil, fmt.Errorf("failed renaming snapshot: %w", retryErr)
		}
	}

	return snapshotPath, activeThreads, nil
}

// ReadActiveThreadsSnapshot reads and unmarshals an active threads snapshot JSON file.
func ReadActiveThreadsSnapshot(snapshotPath string) ([]ActiveThread, error) {
	if strings.TrimSpace(snapshotPath) == "" {
		return nil, errors.New("snapshotPath cannot be empty")
	}
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		return nil, fmt.Errorf("failed reading snapshot file: %w", err)
	}

	var snapshot ActiveThreadsSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed unmarshaling snapshot JSON: %w", err)
	}

	return snapshot.ActiveThreads, nil
}


func isFileViewTab(tab map[string]interface{}) bool {
	if tab == nil {
		return false
	}
	tabID := getMapString(tab, "id")
	if tabID == "" {
		tabID = getMapString(tab, "tabId")
	}
	if strings.HasPrefix(tabID, "file__") {
		return true
	}
	if content, ok := tab["content"].(map[string]interface{}); ok {
		if getMapString(content, "type") == "fileView" {
			return true
		}
	}
	if getMapString(tab, "content.type") == "fileView" || getMapString(tab, "type") == "fileView" {
		return true
	}
	return false
}

func cleanFileURI(raw string) string {
	decoded, err := url.PathUnescape(raw)
	if err == nil && decoded != "" {
		return decoded
	}
	return raw
}

func extractProjectFromText(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}

	if m := reActiveWorkspaces.FindStringSubmatch(text); len(m) > 1 {
		if p := strings.TrimSpace(m[1]); p != "" {
			return p
		}
	}

	if m := reCommandCwd.FindStringSubmatch(text); len(m) > 1 {
		if p := strings.TrimSpace(m[1]); p != "" {
			return p
		}
	}

	if m := reJSONCwd.FindStringSubmatch(text); len(m) > 1 {
		p := strings.TrimSpace(m[1])
		p = strings.ReplaceAll(p, "\\\\", "\\")
		if p != "" {
			return p
		}
	}

	if m := reJSONProj.FindStringSubmatch(text); len(m) > 1 {
		p := strings.TrimSpace(m[1])
		p = strings.ReplaceAll(p, "\\\\", "\\")
		if p != "" {
			return p
		}
	}

	if m := reFileURI.FindStringSubmatch(text); len(m) > 1 {
		raw := m[1]
		if p := cleanFileURI(raw); p != "" {
			return p
		}
	}

	return ""
}

func resolveThreadProject(threadID string, paneData map[string]interface{}, brainDir string, sdata map[string]interface{}) string {
	if paneData != nil {
		for _, k := range projectKeys {
			if val := getMapString(paneData, k); val != "" {
				if strings.HasPrefix(val, "file:///") {
					return cleanFileURI(val[len("file:///"):])
				}
				return val
			}
		}

		if tabs, ok := paneData["tabs"].([]interface{}); ok {
			for _, t := range tabs {
				if tabMap, ok := t.(map[string]interface{}); ok {
					meta, _ := tabMap["metadata"].(map[string]interface{})
					target := tabMap
					if meta != nil {
						target = meta
					}
					for _, k := range projectKeys {
						if val := getMapString(target, k); val != "" {
							if strings.HasPrefix(val, "file:///") {
								return cleanFileURI(val[len("file:///"):])
							}
							return val
						}
					}
				}
			}
		}

		if bytes, err := json.Marshal(paneData); err == nil {
			if extracted := extractProjectFromText(string(bytes)); extracted != "" {
				return extracted
			}
		}
	}

	if brainDir != "" && isDir(brainDir) && threadID != "" {
		threadDir := filepath.Join(brainDir, threadID)
		if isDir(threadDir) {
			metaPath := filepath.Join(threadDir, "metadata.json")
			if isFile(metaPath) {
				if raw, err := os.ReadFile(metaPath); err == nil {
					var meta map[string]interface{}
					if json.Unmarshal(raw, &meta) == nil && meta != nil {
						for _, k := range projectKeys {
							if val := getMapString(meta, k); val != "" {
								if strings.HasPrefix(val, "file:///") {
									return cleanFileURI(val[len("file:///"):])
								}
								return val
							}
						}
					}
				}
			}
		}
	}

	return ""
}

func isDir(p string) bool {
	stat, err := os.Stat(p)
	return err == nil && stat.IsDir()
}

func isFile(p string) bool {
	stat, err := os.Stat(p)
	return err == nil && !stat.IsDir()
}

func getMapString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
