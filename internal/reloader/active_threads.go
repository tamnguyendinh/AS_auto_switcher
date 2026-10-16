package reloader

import (
	"bufio"
	"encoding/json"
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
}

// ActiveThreadsSnapshot encapsulates the persisted state of active conversation threads.
type ActiveThreadsSnapshot struct {
	Timestamp     string         `json:"timestamp"`
	ActiveThreads []ActiveThread `json:"active_threads"`
}

var (
	reActiveWorkspaces = regexp.MustCompile(`(?i)Active Workspaces:\s*[\r\n]+\s*-\s*([^\r\n]+)`)
	reCommandCwd       = regexp.MustCompile(`(?i)Command Working Directory:\s*([^\r\n]+)`)
	reJSONCwd          = regexp.MustCompile(`["']Cwd["']\s*:\s*["']([^"']+)["']`)
	reJSONProj         = regexp.MustCompile(`(?i)["'](?:project|workspace|workspace_path|workingDirectory|working_directory|active_workspace|activeWorkspace)["']\s*:\s*["']([^"']+)["']`)
	reFileURI          = regexp.MustCompile(`file:///([^\r\n"'#]+)`)

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

// defaultCaptureActiveThreadsSnapshot extracts active conversation threads strictly from
// aux-pane-v2-session.conversationPanes in app_storage.json, resolves workspace projects,
// identifies is_main_pane, sorts with main pane first, and writes an atomic timestamped snapshot JSON.
func defaultCaptureActiveThreadsSnapshot(storagePath, brainDir, outputDir string) (string, []ActiveThread, error) {
	sPath := storagePath
	if sPath == "" {
		sPath = GetDefaultAppStoragePath()
	}

	bDir := brainDir
	if bDir == "" {
		home, _ := os.UserHomeDir()
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		if home != "" {
			bDir = filepath.Join(home, ".gemini", "antigravity", "brain")
		}
	}

	outDir := outputDir
	if outDir == "" {
		outDir = GetDefaultThreadsHistoryDir()
	}
	_ = os.MkdirAll(outDir, 0755)

	threadsMap := make(map[string]ActiveThread)
	sdata := make(map[string]interface{})

	if sPath != "" && isFile(sPath) {
		if raw, err := os.ReadFile(sPath); err == nil {
			_ = json.Unmarshal(raw, &sdata)
		}
	}

	var convPanes interface{}
	if val, ok := sdata["aux-pane-v2-session.conversationPanes"]; ok && val != nil {
		convPanes = val
	}
	if convPanes == nil {
		auxSession := sdata["aux-pane-v2-session"]
		if auxSession == nil {
			auxSession = sdata["aux-pane-session"]
		}
		if strAux, ok := auxSession.(string); ok && strings.TrimSpace(strAux) != "" {
			var parsed interface{}
			if json.Unmarshal([]byte(strAux), &parsed) == nil {
				auxSession = parsed
			}
		}
		if mapAux, ok := auxSession.(map[string]interface{}); ok {
			convPanes = mapAux["conversationPanes"]
		}
		if convPanes == nil {
			if cp, ok := sdata["conversationPanes"]; ok && cp != nil {
				convPanes = cp
			}
		}
	}

	if strVal, ok := convPanes.(string); ok && strings.TrimSpace(strVal) != "" {
		var parsed interface{}
		if json.Unmarshal([]byte(strVal), &parsed) == nil {
			convPanes = parsed
		}
	}

	if cpMap, ok := convPanes.(map[string]interface{}); ok {
		var sortedKeys []string
		for k := range cpMap {
			sortedKeys = append(sortedKeys, k)
		}
		sort.Strings(sortedKeys)

		for _, kID := range sortedKeys {
			paneVal := cpMap[kID]
			if strings.HasPrefix(kID, "file__") {
				continue
			}
			paneInfo, _ := paneVal.(map[string]interface{})
			if paneInfo == nil {
				paneInfo = make(map[string]interface{})
			}
			if isFileViewTab(paneInfo) {
				continue
			}
			paneID := getMapString(paneInfo, "paneId")
			isMain := strings.Contains(strings.ToLower(kID), "main") ||
				strings.Contains(strings.ToLower(paneID), "main") ||
				len(threadsMap) == 0

			tabs, _ := paneInfo["tabs"].([]interface{})
			type threadTab struct {
				id  string
				tab map[string]interface{}
			}
			var threadIDsInPane []threadTab
			for _, tabVal := range tabs {
				tab, ok := tabVal.(map[string]interface{})
				if !ok || isFileViewTab(tab) {
					continue
				}
				meta, _ := tab["metadata"].(map[string]interface{})
				tID := getMapString(meta, "cascadeId")
				if tID == "" {
					tID = getMapString(tab, "cascadeId")
				}
				if tID == "" {
					tID = getMapString(tab, "childCascadeId")
				}
				tID = strings.TrimSpace(tID)
				if tID != "" {
					threadIDsInPane = append(threadIDsInPane, threadTab{id: tID, tab: tab})
				}
			}

			if len(threadIDsInPane) > 0 {
				for idx, item := range threadIDsInPane {
					threadID := item.id
					if bDir != "" && isDir(bDir) && !isDir(filepath.Join(bDir, threadID)) {
						continue
					}
					if _, exists := threadsMap[threadID]; !exists {
						proj := resolveThreadProject(threadID, item.tab, bDir, sdata)
						if proj == "" {
							proj = resolveThreadProject(threadID, paneInfo, bDir, sdata)
						}
						threadsMap[threadID] = ActiveThread{
							ThreadID:   threadID,
							Project:    proj,
							IsMainPane: (idx == 0 && isMain),
						}
					}
				}
			} else if strings.TrimSpace(kID) != "" && strings.TrimSpace(kID) != "index" {
				cid := getMapString(paneInfo, "cascadeId")
				if cid == "" {
					cid = getMapString(paneInfo, "activeThreadId")
				}
				if cid == "" {
					cid = getMapString(paneInfo, "threadId")
				}
				if cid == "" {
					cid = strings.TrimSpace(kID)
				}
				cid = strings.TrimSpace(cid)
				if cid != "" && !strings.HasPrefix(cid, "file__") {
					if bDir == "" || !isDir(bDir) || isDir(filepath.Join(bDir, cid)) {
						if _, exists := threadsMap[cid]; !exists {
							proj := resolveThreadProject(cid, paneInfo, bDir, sdata)
							threadsMap[cid] = ActiveThread{
								ThreadID:   cid,
								Project:    proj,
								IsMainPane: isMain,
							}
						}
					}
				}
			}
		}
	} else if cpList, ok := convPanes.([]interface{}); ok {
		for idx, itemVal := range cpList {
			paneInfo, ok := itemVal.(map[string]interface{})
			if !ok || isFileViewTab(paneInfo) {
				continue
			}
			paneID := getMapString(paneInfo, "paneId")
			if paneID == "" {
				paneID = getMapString(paneInfo, "id")
			}
			if strings.HasPrefix(paneID, "file__") {
				continue
			}
			isMain := (idx == 0 || strings.Contains(strings.ToLower(paneID), "main") || len(threadsMap) == 0)

			tabs, _ := paneInfo["tabs"].([]interface{})
			tabFound := false
			for tabIdx, tabVal := range tabs {
				tab, ok := tabVal.(map[string]interface{})
				if !ok || isFileViewTab(tab) {
					continue
				}
				meta, _ := tab["metadata"].(map[string]interface{})
				tID := getMapString(meta, "cascadeId")
				if tID == "" {
					tID = getMapString(tab, "cascadeId")
				}
				if tID == "" {
					tID = getMapString(tab, "childCascadeId")
				}
				tID = strings.TrimSpace(tID)
				if tID != "" {
					if bDir != "" && isDir(bDir) && !isDir(filepath.Join(bDir, tID)) {
						continue
					}
					if _, exists := threadsMap[tID]; !exists {
						proj := resolveThreadProject(tID, tab, bDir, sdata)
						if proj == "" {
							proj = resolveThreadProject(tID, paneInfo, bDir, sdata)
						}
						threadsMap[tID] = ActiveThread{
							ThreadID:   tID,
							Project:    proj,
							IsMainPane: (isMain && tabIdx == 0),
						}
						tabFound = true
					}
				}
			}
			if !tabFound {
				cid := getMapString(paneInfo, "cascadeId")
				if cid == "" {
					cid = getMapString(paneInfo, "activeThreadId")
				}
				if cid == "" {
					cid = getMapString(paneInfo, "threadId")
				}
				if cid == "" {
					cid = paneID
				}
				cid = strings.TrimSpace(cid)
				if cid != "" && cid != "index" && !strings.HasPrefix(cid, "file__") {
					if bDir == "" || !isDir(bDir) || isDir(filepath.Join(bDir, cid)) {
						if _, exists := threadsMap[cid]; !exists {
							proj := resolveThreadProject(cid, paneInfo, bDir, sdata)
							threadsMap[cid] = ActiveThread{
								ThreadID:   cid,
								Project:    proj,
								IsMainPane: isMain,
							}
						}
					}
				}
			}
		}
	}

	activeThreads := make([]ActiveThread, 0, len(threadsMap))
	for _, th := range threadsMap {
		activeThreads = append(activeThreads, th)
	}

	sort.Slice(activeThreads, func(i, j int) bool {
		if activeThreads[i].IsMainPane != activeThreads[j].IsMainPane {
			return activeThreads[i].IsMainPane
		}
		return activeThreads[i].ThreadID < activeThreads[j].ThreadID
	})

	now := time.Now()
	timestampStr := now.Format("2006-01-02 15:04:05")
	fileTs := now.Format("2006-01-02-15-04-05")
	filename := fmt.Sprintf("%s_active_threads.json", fileTs)
	snapshotPath := filepath.Join(outDir, filename)

	payload := ActiveThreadsSnapshot{
		Timestamp:     timestampStr,
		ActiveThreads: activeThreads,
	}

	tmpFile, err := os.CreateTemp(outDir, "snapshot_tmp_*.json")
	if err != nil {
		return "", activeThreads, fmt.Errorf("failed creating temp snapshot file: %w", err)
	}
	tmpName := tmpFile.Name()

	enc := json.NewEncoder(tmpFile)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
		return "", activeThreads, fmt.Errorf("failed writing snapshot JSON: %w", err)
	}
	_ = tmpFile.Sync()
	_ = tmpFile.Close()

	if err := os.Rename(tmpName, snapshotPath); err != nil {
		_ = os.Remove(snapshotPath)
		if retryErr := os.Rename(tmpName, snapshotPath); retryErr != nil {
			_ = os.Remove(tmpName)
			return "", activeThreads, fmt.Errorf("failed moving snapshot file: %w", retryErr)
		}
	}

	return snapshotPath, activeThreads, nil
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
		p = strings.ReplaceAll(p, `\\`, `\`)
		if p != "" {
			return p
		}
	}

	if m := reJSONProj.FindStringSubmatch(text); len(m) > 1 {
		p := strings.TrimSpace(m[1])
		p = strings.ReplaceAll(p, `\\`, `\`)
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

			tCandidates := []string{
				filepath.Join(threadDir, "transcript.jsonl"),
				filepath.Join(threadDir, ".system_generated", "logs", "transcript.jsonl"),
			}
			for _, tFile := range tCandidates {
				if isFile(tFile) {
					if f, err := os.Open(tFile); err == nil {
						scanner := bufio.NewScanner(f)
						count := 0
						for scanner.Scan() && count < 40 {
							line := scanner.Text()
							if extracted := extractProjectFromText(line); extracted != "" {
								_ = f.Close()
								return extracted
							}
							count++
						}
						_ = f.Close()
					}
				}
			}
		}
	}

	if sdata != nil {
		layoutKey1 := fmt.Sprintf("antigravity-multi-conversation-layout-v3-%s", threadID)
		layoutKey2 := fmt.Sprintf("antigravity-multi-conversation-layout-v3-%s+_new", threadID)
		for _, lk := range []string{layoutKey1, layoutKey2} {
			if val := getMapString(sdata, lk); val != "" {
				if extracted := extractProjectFromText(val); extracted != "" {
					return extracted
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
