//go:build windows

package connector

import (
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sys/windows/registry"
)

var (
	// ErrNoDefaultBrowser is returned when no default web browser could be identified.
	ErrNoDefaultBrowser = errors.New("default browser not found")

	// ErrNoBrowsersFound is returned when no installed web browsers were found.
	ErrNoBrowsersFound = errors.New("no installed browsers found")
)

// BrowserInfo represents metadata of an installed web browser on the host system.
type BrowserInfo struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	ExePath   string   `json:"exe_path"`
	IsDefault bool     `json:"is_default"`
	Args      []string `json:"-"`
}

// BrowserLauncher provides discovery and invocation capabilities for web browsers.
type BrowserLauncher interface {
	ListInstalledBrowsers() ([]BrowserInfo, error)
	GetDefaultBrowser() (*BrowserInfo, error)
	Launch(exePath string, targetURL string) error
}

// WindowsBrowserLauncher implements BrowserLauncher by querying Windows Registry.
type WindowsBrowserLauncher struct {
	mu       sync.Mutex
	commands map[string][]string
}

// NewWindowsBrowserLauncher creates a new WindowsBrowserLauncher.
func NewWindowsBrowserLauncher() *WindowsBrowserLauncher {
	return &WindowsBrowserLauncher{commands: make(map[string][]string)}
}

// NewBrowserLauncher returns a platform-standard BrowserLauncher instance.
func NewBrowserLauncher() BrowserLauncher {
	return NewWindowsBrowserLauncher()
}

// ParseBrowserCommand safely splits a Windows command line string into an executable path
// and its arguments. It properly handles both quoted and unquoted paths, preserves casing,
// and removes placeholder arguments such as %1 or "%1".
func ParseBrowserCommand(cmdStr string) (string, []string, error) {
	trimmed := strings.TrimSpace(cmdStr)
	if trimmed == "" {
		return "", nil, errors.New("empty browser command string")
	}

	var exePath string
	var remainder string

	if strings.HasPrefix(trimmed, "\"") {
		// Quoted path: find the closing quote
		endQuote := strings.Index(trimmed[1:], "\"")
		if endQuote == -1 {
			// Malformed quote: take the whole string after opening quote
			exePath = trimmed[1:]
			remainder = ""
		} else {
			exePath = trimmed[1 : endQuote+1]
			remainder = strings.TrimSpace(trimmed[endQuote+2:])
		}
	} else {
		// Unquoted path: could contain spaces (e.g., C:\Program Files\...) or be simple.
		// 1. If entire command directly exists on disk, it has no args.
		if isRegularExecutable(trimmed) {
			exePath = trimmed
			remainder = ""
		} else {
			// 2. Look for .exe suffix (case-insensitive)
			lower := strings.ToLower(trimmed)
			exeIdx := strings.Index(lower, ".exe")
			if exeIdx != -1 {
				candidate := trimmed[:exeIdx+4]
				candidateRemainder := strings.TrimSpace(trimmed[exeIdx+4:])
				exePath = candidate
				remainder = candidateRemainder
			} else {
				// 3. Fallback: split by first space
				fields := strings.Fields(trimmed)
				if len(fields) > 0 {
					exePath = fields[0]
					if len(fields) > 1 {
						remainder = strings.Join(fields[1:], " ")
					}
				}
			}
		}
	}

	exePath = strings.Trim(strings.TrimSpace(exePath), "\"")
	if exePath == "" {
		return "", nil, errors.New("cannot extract executable from command")
	}

	argv, err := windows.DecomposeCommandLine("browser.exe " + remainder)
	if err != nil {
		return "", nil, err
	}
	var filteredArgs []string
	for _, arg := range argv[1:] {
		if arg != "%1" && arg != "%L" && arg != "%l" {
			filteredArgs = append(filteredArgs, arg)
		}
	}

	return filepath.Clean(exePath), filteredArgs, nil
}

// GetDefaultBrowser resolves the default Windows web browser for HTTPS associations.
func (l *WindowsBrowserLauncher) GetDefaultBrowser() (*BrowserInfo, error) {
	defaultExe, progID, defaultArgs, err := l.resolveDefaultBrowserCommand()
	if err == nil && defaultExe != "" {
		name := friendlyBrowserNameFromExe(defaultExe, progID)
		return &BrowserInfo{
			ID:        cleanBrowserID(name, defaultExe),
			Name:      name,
			ExePath:   defaultExe,
			Args:      defaultArgs,
			IsDefault: true,
		}, nil
	}

	// Fallback to installed browsers list if UserChoice was not resolved
	browsers, listErr := l.ListInstalledBrowsers()
	if listErr == nil && len(browsers) > 0 {
		for i := range browsers {
			if browsers[i].IsDefault {
				return &browsers[i], nil
			}
		}

	}

	return nil, ErrNoDefaultBrowser
}

// ListInstalledBrowsers enumerates installed web browsers from HKCU and HKLM StartMenuInternet keys.
func (l *WindowsBrowserLauncher) ListInstalledBrowsers() ([]BrowserInfo, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	defaultExe, _, defaultArgs, _ := l.resolveDefaultBrowserCommand()
	if l.commands == nil {
		l.commands = make(map[string][]string)
	}
	if defaultExe != "" {
		l.commands[strings.ToLower(filepath.Clean(defaultExe))] = defaultArgs
	}

	seen := make(map[string]bool)
	var browsers []BrowserInfo

	hives := []struct {
		hive registry.Key
		path string
	}{
		{registry.CURRENT_USER, `Software\Clients\StartMenuInternet`},
		{registry.LOCAL_MACHINE, `SOFTWARE\Clients\StartMenuInternet`},
	}

	for _, h := range hives {
		k, err := registry.OpenKey(h.hive, h.path, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
		if err != nil {
			continue
		}

		subkeys, err := k.ReadSubKeyNames(-1)
		_ = k.Close()
		if err != nil {
			continue
		}

		for _, subkeyName := range subkeys {
			subPath := h.path + `\` + subkeyName
			subKey, err := registry.OpenKey(h.hive, subPath, registry.QUERY_VALUE)
			if err != nil {
				continue
			}

			// Read friendly name from default value
			displayName, _, _ := subKey.GetStringValue("")
			if strings.TrimSpace(displayName) == "" {
				displayName = subkeyName
			}

			// Read command from shell\open\command
			cmdKey, err := registry.OpenKey(subKey, `shell\open\command`, registry.QUERY_VALUE)
			_ = subKey.Close()
			if err != nil {
				continue
			}

			cmdStr, _, err := cmdKey.GetStringValue("")
			_ = cmdKey.Close()
			if err != nil || strings.TrimSpace(cmdStr) == "" {
				continue
			}

			exePath, args, err := ParseBrowserCommand(cmdStr)
			if err != nil || !isRegularExecutable(exePath) {
				continue
			}

			if l.commands == nil {
				l.commands = make(map[string][]string)
			}
			l.commands[strings.ToLower(filepath.Clean(exePath))] = append([]string(nil), args...)
			canonicalPath := filepath.Clean(exePath)
			keyLower := strings.ToLower(canonicalPath)
			if seen[keyLower] {
				continue
			}
			seen[keyLower] = true

			isDefault := false
			if defaultExe != "" && strings.EqualFold(canonicalPath, defaultExe) {
				isDefault = true
			}

			browsers = append(browsers, BrowserInfo{
				ID:        cleanBrowserID(displayName, canonicalPath),
				Name:      displayName,
				ExePath:   canonicalPath,
				Args:      args,
				IsDefault: isDefault,
			})
		}
	}

	// If defaultExe exists on disk but was not listed under StartMenuInternet, include it
	if defaultExe != "" && isRegularExecutable(defaultExe) && !seen[strings.ToLower(defaultExe)] {
		name := friendlyBrowserNameFromExe(defaultExe, "")
		browsers = append([]BrowserInfo{{
			ID:        cleanBrowserID(name, defaultExe),
			Name:      name,
			ExePath:   defaultExe,
			Args:      defaultArgs,
			IsDefault: true,
		}}, browsers...)
		seen[strings.ToLower(defaultExe)] = true
	}

	if len(browsers) == 0 {
		return nil, ErrNoBrowsersFound
	}

	return browsers, nil
}

// Launch executes the browser at exePath with targetURL without using a shell interpreter.
// If exePath is empty, it attempts to launch via the resolved default browser.
func (l *WindowsBrowserLauncher) Launch(exePath string, targetURL string) error {
	if targetURL == "" {
		return errors.New("empty browser URL")
	}
	resolvedPath := strings.TrimSpace(exePath)

	if resolvedPath != "" {
		if !isRegularExecutable(resolvedPath) {
			return fmt.Errorf("browser executable not found or not a file: %s", resolvedPath)
		}

		l.mu.Lock()
		args := append([]string(nil), l.commands[strings.ToLower(filepath.Clean(resolvedPath))]...)
		l.mu.Unlock()
		if targetURL != "" {
			args = append(args, targetURL)
		}
		cmd := exec.Command(resolvedPath, args...)

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("start browser (%s): %w", resolvedPath, err)
		}
		return nil
	}

	// Universal fallback if no browser executable could be resolved: rundll32 url.dll,FileProtocolHandler
	if targetURL != "" {
		cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", targetURL)
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("system browser fallback launch: %w", err)
		}
		return nil
	}

	return errors.New("cannot launch browser: empty executable and empty URL")
}

// resolveDefaultBrowserExe reads ProgId from UserChoice and looks up the shell open command.
func (l *WindowsBrowserLauncher) resolveDefaultBrowserCommand() (string, string, []string, error) {
	// 1. Read ProgId from UserChoice
	userChoiceKey := `Software\Microsoft\Windows\Shell\Associations\UrlAssociations\https\UserChoice`
	k, err := registry.OpenKey(registry.CURRENT_USER, userChoiceKey, registry.QUERY_VALUE)
	if err != nil {
		return "", "", nil, err
	}
	progID, _, err := k.GetStringValue("ProgId")
	_ = k.Close()
	if err != nil || strings.TrimSpace(progID) == "" {
		return "", "", nil, errors.New("no UserChoice ProgId")
	}

	progID = strings.TrimSpace(progID)

	// 2. Query shell\open\command for ProgId under HKCU\Software\Classes then HKLM\SOFTWARE\Classes
	classPaths := []struct {
		hive registry.Key
		path string
	}{
		{registry.CURRENT_USER, `Software\Classes\` + progID + `\shell\open\command`},
		{registry.LOCAL_MACHINE, `SOFTWARE\Classes\` + progID + `\shell\open\command`},
	}

	for _, cp := range classPaths {
		kCmd, err := registry.OpenKey(cp.hive, cp.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		cmdStr, _, err := kCmd.GetStringValue("")
		_ = kCmd.Close()
		if err != nil || strings.TrimSpace(cmdStr) == "" {
			continue
		}

		exePath, args, err := ParseBrowserCommand(cmdStr)
		if err == nil && isRegularExecutable(exePath) {
			return filepath.Clean(exePath), progID, args, nil
		}
	}

	return "", progID, nil, fmt.Errorf("command not found for ProgId %s", progID)
}

func friendlyBrowserNameFromExe(exePath, fallbackProgID string) string {
	base := strings.ToLower(filepath.Base(exePath))
	switch {
	case strings.Contains(base, "chrome"):
		return "Google Chrome"
	case strings.Contains(base, "msedge") || strings.Contains(base, "edge"):
		return "Microsoft Edge"
	case strings.Contains(base, "brave"):
		return "Brave"
	case strings.Contains(base, "firefox"):
		return "Mozilla Firefox"
	case strings.Contains(base, "opera"):
		return "Opera"
	case strings.Contains(base, "vivaldi"):
		return "Vivaldi"
	case strings.Contains(base, "coccoc"):
		return "Cốc Cốc"
	case strings.Contains(base, "waterfox"):
		return "Waterfox"
	case strings.Contains(base, "librewolf"):
		return "LibreWolf"
	case strings.Contains(base, "floorp"):
		return "Ablaze Floorp"
	default:
		if fallbackProgID != "" {
			return fallbackProgID
		}
		return strings.TrimSuffix(filepath.Base(exePath), filepath.Ext(exePath))
	}
}

func cleanBrowserID(name, exePath string) string {
	id := strings.ToLower(name)
	id = strings.ReplaceAll(id, " ", "_")
	if id == "" {
		id = strings.ToLower(filepath.Base(exePath))
	}
	return id
}
