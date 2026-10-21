package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	DefaultSwitchTimelineDir           = "logs/switch_timeline"
	DefaultSwitchTimelineRetentionDays = 7
)

// GetTodaySwitchTimelinePath returns the canonical path for today's switch timeline log file.
func GetTodaySwitchTimelinePath(baseDir ...string) string {
	dir := DefaultSwitchTimelineDir
	if len(baseDir) > 0 && strings.TrimSpace(baseDir[0]) != "" {
		dir = baseDir[0]
	}
	today := time.Now().Format("2006-01-02")
	return filepath.Join(dir, fmt.Sprintf("%s_switch_timeline.log", today))
}

// LogSwitchEvent appends a timestamped line to today's switch timeline log file.
// Format: [15:04:05.000] [+ 1234ms] [EVENT_TAG] Message
func LogSwitchEvent(timelineDir string, eventTag string, message string, elapsed ...time.Duration) {
	if strings.TrimSpace(timelineDir) == "" {
		timelineDir = DefaultSwitchTimelineDir
	}
	_ = os.MkdirAll(timelineDir, 0755)

	today := time.Now().Format("2006-01-02")
	filePath := filepath.Join(timelineDir, fmt.Sprintf("%s_switch_timeline.log", today))

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return // Fail-safe: ignore disk write errors
	}
	defer f.Close()

	nowStr := time.Now().Format("15:04:05.000")
	elapsedStr := ""
	if len(elapsed) > 0 {
		elapsedStr = fmt.Sprintf("[+ %6dms] ", elapsed[0].Milliseconds())
	}

	line := fmt.Sprintf("[%s] %s[%-16s] %s\n", nowStr, elapsedStr, eventTag, message)
	_, _ = f.WriteString(line)
}

// PruneDailySwitchTimelineLogs keeps only the most recent keepCount timeline log files.
func PruneDailySwitchTimelineLogs(timelineDir string, keepCount int) (int, error) {
	if keepCount <= 0 {
		keepCount = DefaultSwitchTimelineRetentionDays
	}
	dir := timelineDir
	if strings.TrimSpace(dir) == "" {
		dir = DefaultSwitchTimelineDir
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	var logFiles []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), "_switch_timeline.log") {
			logFiles = append(logFiles, entry)
		}
	}

	// Sắp xếp giảm dần theo tên: file mới nhất đứng đầu
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].Name() > logFiles[j].Name()
	})

	if len(logFiles) <= keepCount {
		return 0, nil
	}

	deleted := 0
	for i := keepCount; i < len(logFiles); i++ {
		targetFile := filepath.Join(dir, logFiles[i].Name())
		if err := os.Remove(targetFile); err == nil {
			deleted++
		}
	}

	return deleted, nil
}
