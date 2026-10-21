package reloader

import (
	"database/sql"
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

	_ "modernc.org/sqlite"
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

var reUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// DefaultHistoryKeepCount defines the maximum number of active threads snapshot files retained.
const DefaultHistoryKeepCount = 2


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
// It strictly requires basenames to be valid UUIDs (F3).
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
		// F3: Strict UUID validation on basename
		if !reUUID.MatchString(tID) {
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

// resolveProjectFromSummaries queries workspace_uris directly from SQLite (F2).
// It avoids reading chat preview text or raw binary patterns.
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
		dsn := fmt.Sprintf("file:%s?mode=ro", filepath.ToSlash(sDb))
		db, err := sql.Open("sqlite", dsn)
		if err == nil {
			defer db.Close()
			var rawURIs string
			err = db.QueryRow("SELECT workspace_uris FROM conversation_summaries WHERE conversation_id = ? LIMIT 1", threadID).Scan(&rawURIs)
			if err == nil && strings.TrimSpace(rawURIs) != "" {
				var uris []string
				if json.Unmarshal([]byte(rawURIs), &uris) == nil && len(uris) > 0 {
					uri := uris[0]
					if strings.HasPrefix(uri, "file:///") {
						return cleanFileURI(uri[len("file:///"):])
					}
					return uri
				}
				if strings.HasPrefix(rawURIs, "file:///") {
					return cleanFileURI(rawURIs[len("file:///"):])
				}
				return rawURIs
			}
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

func cleanFileURI(raw string) string {
	decoded, err := url.PathUnescape(raw)
	if err == nil && decoded != "" {
		return decoded
	}
	return raw
}

func isFile(p string) bool {
	stat, err := os.Stat(p)
	return err == nil && !stat.IsDir()
}

// PruneActiveThreadsHistory prunes active threads snapshot files in historyDir,
// retaining only the newest keepCount files (sorted descending by timestamp in filename).
// If keepCount <= 0, DefaultHistoryKeepCount (2) is used.
// If historyDir is empty, GetDefaultThreadsHistoryDir() is used.
// If historyDir does not exist, it safely returns (0, nil).
func PruneActiveThreadsHistory(historyDir string, keepCount int) (int, error) {
	if keepCount <= 0 {
		keepCount = DefaultHistoryKeepCount
	}
	if historyDir == "" {
		historyDir = GetDefaultThreadsHistoryDir()
	}

	entries, err := os.ReadDir(historyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed reading history dir %s: %w", historyDir, err)
	}

	var snapshots []os.DirEntry
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), "_active_threads.json") {
			snapshots = append(snapshots, entry)
		}
	}

	if len(snapshots) <= keepCount {
		return 0, nil
	}

	// Filename format YYYY-MM-DD-HH-mm-ss_active_threads.json sorts descending alphabetically
	// identical to chronological descending (newest first).
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Name() > snapshots[j].Name()
	})

	deleted := 0
	for k := keepCount; k < len(snapshots); k++ {
		filePath := filepath.Join(historyDir, snapshots[k].Name())
		if err := os.Remove(filePath); err == nil {
			deleted++
		}
	}

	return deleted, nil
}
