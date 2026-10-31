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

// DefaultActiveThreadsWindow defines the time window for detecting active conversation threads.
const DefaultActiveThreadsWindow = 5 * time.Minute


// GetDefaultThreadsHistoryDir returns the canonical logs/active_threads_history directory.
func GetDefaultThreadsHistoryDir() string {
	historyDir := filepath.Join("logs", "active_threads_history")
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

// ScanActiveConversations scans %USERPROFILE%/.gemini/antigravity/conversations/
// for active threads within DefaultActiveThreadsWindow (5m) and arranges them
// hierarchically: children first, standalone root next, and active parents last.
// It strictly requires basenames to be valid UUIDs and ignores killed threads.
func ScanActiveConversations(conversationsDir string, summariesDbPath string, startTime time.Time) ([]ActiveThread, error) {
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

	threshold := startTime.Add(-DefaultActiveThreadsWindow)

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

	var rawCandidates []*threadCandidate
	for _, cand := range candidates {
		effective := cand.dbMtime
		if cand.hasWal && (cand.walMtime.After(effective) || !cand.hasDb) {
			effective = cand.walMtime
		}
		cand.effectiveMtime = effective

		if effective.Before(threshold) {
			continue
		}
		rawCandidates = append(rawCandidates, cand)
	}

	if len(rawCandidates) == 0 {
		return []ActiveThread{}, nil
	}

	// Sort rawCandidates by effectiveMtime descending (newest first)
	sort.Slice(rawCandidates, func(i, j int) bool {
		return rawCandidates[i].effectiveMtime.After(rawCandidates[j].effectiveMtime)
	})

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

	var db *sql.DB
	if isFile(sDb) {
		dsn := fmt.Sprintf("file:%s?mode=ro", filepath.ToSlash(sDb))
		openedDB, err := sql.Open("sqlite", dsn)
		if err == nil && openedDB != nil {
			db = openedDB
			defer db.Close()
		}
	}

	var childrenList []ActiveThread
	parentIDsSet := make(map[string]bool)
	var parentOrder []string
	var candidateStandalone []ActiveThread

	for _, cand := range rawCandidates {
		pID, uris, killed := queryConversationSummaryMeta(db, cand.threadID)
		if killed {
			continue
		}
		proj := parseWorkspaceProject(uris)
		if pID != "" && reUUID.MatchString(pID) {
			childrenList = append(childrenList, ActiveThread{
				ThreadID: cand.threadID,
				Project:  proj,
			})
			if !parentIDsSet[pID] {
				parentIDsSet[pID] = true
				parentOrder = append(parentOrder, pID)
			}
		} else {
			candidateStandalone = append(candidateStandalone, ActiveThread{
				ThreadID: cand.threadID,
				Project:  proj,
			})
		}
	}

	// Step 4: Tách lọc và Gom cụm Gia đình
	orderedChildren := childrenList

	var orderedStandalone []ActiveThread
	for _, cand := range candidateStandalone {
		if !parentIDsSet[cand.ThreadID] {
			orderedStandalone = append(orderedStandalone, cand)
		}
	}

	var orderedParents []ActiveThread
	for _, pID := range parentOrder {
		_, parentURIs, parentKilled := queryConversationSummaryMeta(db, pID)
		if !parentKilled {
			orderedParents = append(orderedParents, ActiveThread{
				ThreadID: pID,
				Project:  parseWorkspaceProject(parentURIs),
			})
		}
	}

	// Step 5: Ghép danh sách: Children -> Standalone -> Parents & Khử trùng lặp
	fullList := make([]ActiveThread, 0, len(orderedChildren)+len(orderedStandalone)+len(orderedParents))
	fullList = append(fullList, orderedChildren...)
	fullList = append(fullList, orderedStandalone...)
	fullList = append(fullList, orderedParents...)

	seen := make(map[string]bool)
	result := make([]ActiveThread, 0, len(fullList))
	for _, th := range fullList {
		cleanID := strings.TrimSpace(th.ThreadID)
		if cleanID == "" || seen[cleanID] {
			continue
		}
		seen[cleanID] = true
		th.IsMainPane = (len(result) == 0)
		result = append(result, th)
	}

	return result, nil
}

// queryConversationSummaryMeta retrieves metadata for a thread from conversation_summaries.
// It checks parent_conversation_id, workspace_uris, and killed status.
// Invariant 7: If the SQLite table lacks parent_conversation_id or killed columns,
// it falls back defensively to querying only workspace_uris.
func queryConversationSummaryMeta(db *sql.DB, threadID string) (parentID string, workspaceURIs string, killed bool) {
	if db == nil || threadID == "" {
		return "", "", false
	}
	var pIDVal, urisVal, kVal any
	err := db.QueryRow("SELECT parent_conversation_id, workspace_uris, killed FROM conversation_summaries WHERE conversation_id = ? LIMIT 1", threadID).Scan(&pIDVal, &urisVal, &kVal)
	if err != nil {
		// Fallback phòng thủ: nếu mock table thiếu cột parent_conversation_id hoặc killed
		var fallbackURIs any
		_ = db.QueryRow("SELECT workspace_uris FROM conversation_summaries WHERE conversation_id = ? LIMIT 1", threadID).Scan(&fallbackURIs)
		if fallbackURIs != nil {
			if s, ok := fallbackURIs.(string); ok {
				return "", s, false
			}
		}
		return "", "", false
	}

	var pID, uris string
	if pIDVal != nil {
		if s, ok := pIDVal.(string); ok {
			pID = s
		}
	}
	if urisVal != nil {
		if s, ok := urisVal.(string); ok {
			uris = s
		}
	}

	isKilled := false
	switch v := kVal.(type) {
	case int64:
		isKilled = (v != 0)
	case int:
		isKilled = (v != 0)
	case bool:
		isKilled = v
	case string:
		isKilled = (v == "1" || strings.EqualFold(v, "true"))
	}
	return strings.TrimSpace(pID), uris, isKilled
}

// parseWorkspaceProject extracts the root project path from raw workspace_uris JSON or string.
func parseWorkspaceProject(rawURIs string) string {
	raw := strings.TrimSpace(rawURIs)
	if raw == "" {
		return ""
	}
	var uris []string
	if json.Unmarshal([]byte(raw), &uris) == nil && len(uris) > 0 {
		uri := uris[0]
		if strings.HasPrefix(uri, "file:///") {
			return cleanFileURI(uri[len("file:///"):])
		}
		return uri
	}
	if strings.HasPrefix(raw, "file:///") {
		return cleanFileURI(raw[len("file:///"):])
	}
	return raw
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
			_, uris, _ := queryConversationSummaryMeta(db, threadID)
			return parseWorkspaceProject(uris)
		}
	}

	return ""
}

// defaultCaptureActiveThreadsSnapshot extracts active conversation threads from
// conversations/*.db and *.db-wal with MAX(mtime) >= start - DefaultActiveThreadsWindow (5m), resolves projects,
// and saves an atomic snapshot JSON to outputDir.
func defaultCaptureActiveThreadsSnapshot(conversationsDir, summariesDbPath, outputDir string) (string, []ActiveThread, error) {
	start := time.Now()
	activeThreads, err := ScanActiveConversations(conversationsDir, summariesDbPath, start)
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
