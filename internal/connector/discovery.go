package connector

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sync/singleflight"
	"golang.org/x/sys/windows"
)

var (
	ErrLanguageServerNotFound = errors.New("language_server.exe process not found or not ready")
	ErrNoListeningPort        = errors.New("no loopback listening port found for language_server.exe")
	ErrInvalidCSRFToken       = errors.New("invalid or missing CSRF token in language_server command line")
)

var (
	modNtdll                      = windows.NewLazySystemDLL("ntdll.dll")
	procNtQueryInformationProcess = modNtdll.NewProc("NtQueryInformationProcess")
)

const (
	processCommandLineInformation = 60
)

type unicodeString struct {
	Length        uint16
	MaximumLength uint16
	Buffer        *uint16
}

// getProcessCommandLine đọc dòng lệnh của tiến trình bằng NtQueryInformationProcess class 60.
func getProcessCommandLine(hProcess windows.Handle) (string, error) {
	var returnLength uint32
	// Lần gọi 1 để lấy độ dài buffer cần cấp phát
	procNtQueryInformationProcess.Call(
		uintptr(hProcess),
		uintptr(processCommandLineInformation),
		0,
		0,
		uintptr(unsafe.Pointer(&returnLength)),
	)
	if returnLength == 0 {
		return "", errors.New("không xác định được kích thước buffer dòng lệnh")
	}

	buf := make([]byte, returnLength)
	r0, _, _ := procNtQueryInformationProcess.Call(
		uintptr(hProcess),
		uintptr(processCommandLineInformation),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(returnLength),
		uintptr(unsafe.Pointer(&returnLength)),
	)
	if int32(r0) < 0 {
		return "", fmt.Errorf("NtQueryInformationProcess lỗi NTSTATUS 0x%X", r0)
	}

	us := (*unicodeString)(unsafe.Pointer(&buf[0]))
	if us.Length == 0 || us.Buffer == nil {
		return "", nil
	}

	charCount := int(us.Length / 2)
	slice := unsafe.Slice(us.Buffer, charCount)
	return windows.UTF16ToString(slice), nil
}

// CSRFTokenRegex matches --csrf_token in command line arguments:
//   --csrf_token <UUID>
//   --csrf_token=<UUID>
//   --csrf_token "<UUID>"
//   --csrf_token='<UUID>'
var CSRFTokenRegex = regexp.MustCompile(`--csrf_token(?:=|\s+)(?:["']?)([a-zA-Z0-9_-]+)(?:["']?)`)

// ServerInfo encapsulates the parameters required to connect to Antigravity Language Server.
type ServerInfo struct {
	PID       int    `json:"pid"`
	CSRFToken string `json:"csrf_token"`
	Port      int    `json:"port"`
}

// BaseURL returns the loopback base URL for HTTP communication.
func (s *ServerInfo) BaseURL() string {
	if s == nil || s.Port <= 0 {
		return ""
	}
	return fmt.Sprintf("http://127.0.0.1:%d", s.Port)
}

// ExtractCSRFToken parses the CSRF token UUID from the process command-line arguments.
func ExtractCSRFToken(cmdline string) string {
	if strings.TrimSpace(cmdline) == "" {
		return ""
	}
	matches := CSRFTokenRegex.FindStringSubmatch(cmdline)
	if len(matches) > 1 {
		return strings.Trim(matches[1], `"' `)
	}
	return ""
}

// ProcessCandidate holds PID and CommandLine for candidate processes.
type ProcessCandidate struct {
	PID         int    `json:"ProcessId"`
	CommandLine string `json:"CommandLine"`
}

// QueryLanguageServerProcesses tìm các tiến trình language_server.exe qua Win32 Toolhelp32 Snapshot thuần (< 1ms).
func QueryLanguageServerProcesses() ([]ProcessCandidate, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("CreateToolhelp32Snapshot failed: %w", err)
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	if err := windows.Process32First(snapshot, &entry); err != nil {
		return nil, fmt.Errorf("Process32First failed: %w", err)
	}

	var candidates []ProcessCandidate
	for {
		name := windows.UTF16ToString(entry.ExeFile[:])
		if strings.EqualFold(name, "language_server.exe") {
			pid := int(entry.ProcessID)
			hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, entry.ProcessID)
			if err != nil {
				hProcess, err = windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, entry.ProcessID)
			}
			if err == nil {
				cmdLine, err := getProcessCommandLine(hProcess)
				_ = windows.CloseHandle(hProcess)
				if err == nil && cmdLine != "" {
					candidates = append(candidates, ProcessCandidate{
						PID:         pid,
						CommandLine: cmdLine,
					})
				}
			}
		}

		if err := windows.Process32Next(snapshot, &entry); err != nil {
			break
		}
	}
	return candidates, nil
}

const (
	afInet                   = 2 // AF_INET
	tcpTableOwnerPIDListener = 3 // TCP_TABLE_OWNER_PID_LISTENER
	mibTCPStateListen        = 2 // MIB_TCP_STATE_LISTEN
)

var (
	modIphlpapi             = windows.NewLazySystemDLL("iphlpapi.dll")
	procGetExtendedTcpTable = modIphlpapi.NewProc("GetExtendedTcpTable")
)

// discoverPortsViaWin32 queries iphlpapi.dll::GetExtendedTcpTable in pure Go without CGo.
func discoverPortsViaWin32(pid int) ([]int, error) {
	if pid <= 0 {
		return nil, nil
	}

	var size uint32
	// 1. First call to retrieve required buffer size
	r0, _, _ := procGetExtendedTcpTable.Call(
		0,
		uintptr(unsafe.Pointer(&size)),
		0,
		uintptr(afInet),
		uintptr(tcpTableOwnerPIDListener),
		0,
	)
	if windows.Errno(r0) != windows.ERROR_INSUFFICIENT_BUFFER {
		return nil, fmt.Errorf("GetExtendedTcpTable size query returned: %v", windows.Errno(r0))
	}

	buf := make([]byte, size)
	// 2. Second call to retrieve table
	r0, _, _ = procGetExtendedTcpTable.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
		0,
		uintptr(afInet),
		uintptr(tcpTableOwnerPIDListener),
		0,
	)
	if r0 != 0 {
		return nil, fmt.Errorf("GetExtendedTcpTable failed: %v", windows.Errno(r0))
	}

	if len(buf) < 4 {
		return nil, errors.New("GetExtendedTcpTable returned buffer too small")
	}

	numEntries := binary.LittleEndian.Uint32(buf[0:4])
	rowSize := 24 // 6 * sizeof(DWORD)
	portsMap := make(map[int]struct{})

	for i := uint32(0); i < numEntries; i++ {
		offset := 4 + i*uint32(rowSize)
		if int(offset)+rowSize > len(buf) {
			break
		}

		state := binary.LittleEndian.Uint32(buf[offset : offset+4])
		localAddr := binary.LittleEndian.Uint32(buf[offset+4 : offset+8])
		owningPID := binary.LittleEndian.Uint32(buf[offset+20 : offset+24])

		if int(owningPID) != pid {
			continue
		}
		if state != mibTCPStateListen {
			continue
		}

		// Check loopback or wildcard (127.0.0.1 = 0x0100007F in little-endian, 0.0.0.0 = 0)
		ip := net.IPv4(byte(localAddr), byte(localAddr>>8), byte(localAddr>>16), byte(localAddr>>24))
		if !ip.IsLoopback() && !ip.IsUnspecified() {
			continue
		}

		// Port is stored in network byte order (big endian) in the low 16 bits
		port := int(binary.BigEndian.Uint16(buf[offset+8 : offset+10]))
		if port > 0 {
			portsMap[port] = struct{}{}
		}
	}

	ports := make([]int, 0, len(portsMap))
	for p := range portsMap {
		ports = append(ports, p)
	}
	sort.Ints(ports)
	return ports, nil
}

// DiscoverListeningPorts tìm tất cả các cổng TCP listening loopback bằng Win32 GetExtendedTcpTable thuần
func DiscoverListeningPorts(pid int) ([]int, error) {
	if pid <= 0 {
		return nil, errors.New("invalid PID")
	}
	ports, err := discoverPortsViaWin32(pid)
	if err != nil || len(ports) == 0 {
		return nil, fmt.Errorf("không tìm thấy cổng lắng nghe cho PID %d: %w", pid, err)
	}
	return ports, nil
}

// ProbeHTTPPort probes a loopback port to check if it responds with HTTP status 200.
func ProbeHTTPPort(port int, timeout time.Duration) bool {
	if port <= 0 {
		return false
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/", port)
	client := &http.Client{
		Timeout: timeout,
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// DiscoverListeningPort finds the primary Connect-RPC HTTP listening port for the PID.
// When multiple ports are listening, it probes each port and selects the one returning HTTP 200.
func DiscoverListeningPort(pid int) (int, error) {
	ports, err := DiscoverListeningPorts(pid)
	if err != nil || len(ports) == 0 {
		return 0, ErrNoListeningPort
	}

	if len(ports) == 1 {
		return ports[0], nil
	}

	// Probe each port to find HTTP 200 listener (probe timeout 1s)
	for _, p := range ports {
		if ProbeHTTPPort(p, 1*time.Second) {
			return p, nil
		}
	}

	// In Antigravity, the lower port is the HTTPS listener (which rejects plain HTTP with 400),
	// while the higher port is always the Connect-RPC HTTP listener. Default to the highest port!
	return ports[len(ports)-1], nil
}

// CheckProcessLiveness kiểm tra nhanh tính sống sót của PID bằng Win32 API thuần (< 0.02ms).
// Sử dụng PROCESS_QUERY_LIMITED_INFORMATION và GetExitCodeProcess.
func CheckProcessLiveness(pid int) bool {
	if pid <= 0 {
		return false
	}
	hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(hProcess)

	var exitCode uint32
	if err := windows.GetExitCodeProcess(hProcess, &exitCode); err != nil {
		return false
	}
	// STILL_ACTIVE = 259 trong Win32 API
	return exitCode == 259
}

// CheckServerLiveness xác minh toàn diện cả tiến trình OS lẫn tính sẵn sàng của cổng HTTP loopback.
func CheckServerLiveness(info *ServerInfo) bool {
	if info == nil || info.PID <= 0 || info.Port <= 0 {
		return false
	}
	if !CheckProcessLiveness(info.PID) {
		return false
	}
	// Thăm dò nhanh cổng HTTP (timeout 200ms)
	return ProbeHTTPPort(info.Port, 200*time.Millisecond)
}

type serverDiscoveryCache struct {
	mu           sync.RWMutex
	info         *ServerInfo
	creationTime windows.Filetime
	cachedAt     time.Time
}

var (
	discCache  serverDiscoveryCache
	discFlight singleflight.Group
)

// InvalidateDiscoveryCache xóa cache nếu thông tin kết nối bị lỗi
func InvalidateDiscoveryCache(failedPort int, failedCSRF string) {
	discCache.mu.Lock()
	defer discCache.mu.Unlock()
	if discCache.info != nil {
		if (failedPort <= 0 || discCache.info.Port == failedPort) &&
			(failedCSRF == "" || discCache.info.CSRFToken == failedCSRF) {
			discCache.info = nil
			discCache.cachedAt = time.Time{}
		}
	}
}

// GetCachedServerInfo returns the current cached ServerInfo (or nil if cache is empty).
func GetCachedServerInfo() *ServerInfo {
	discCache.mu.RLock()
	defer discCache.mu.RUnlock()
	return discCache.info
}

// SetDiscoveryCacheForTesting sets the discovery cache for unit test validation.
func SetDiscoveryCacheForTesting(info *ServerInfo, creationTime windows.Filetime) {
	discCache.mu.Lock()
	defer discCache.mu.Unlock()
	discCache.info = info
	discCache.creationTime = creationTime
	discCache.cachedAt = time.Now()
}

// DiscoverLanguageServer quét tiến trình language_server.exe với bộ nhớ đệm 10s và singleflight gom luồng
func DiscoverLanguageServer() (*ServerInfo, error) {
	// 1. Kiểm tra cache còn hạn (TTL 10s)
	discCache.mu.Lock()
	cached := discCache.info
	cachedTime := discCache.cachedAt
	cachedCreation := discCache.creationTime
	if cached != nil && time.Since(cachedTime) <= 10*time.Second {
		isAlive := false
		if hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(cached.PID)); err == nil {
			var ct, et, kt, ut windows.Filetime
			var exitCode uint32
			if err := windows.GetProcessTimes(hProcess, &ct, &et, &kt, &ut); err == nil {
				if errExit := windows.GetExitCodeProcess(hProcess, &exitCode); errExit == nil {
					// STILL_ACTIVE = 259 trong Win32 API
					if exitCode == 259 && et.LowDateTime == 0 && et.HighDateTime == 0 && ct == cachedCreation {
						isAlive = true
					}
				}
			}
			_ = windows.CloseHandle(hProcess)
		}

		if isAlive && ProbeHTTPPort(cached.Port, 300*time.Millisecond) {
			discCache.mu.Unlock()
			return cached, nil
		}

		// Tiến trình đã chết hoặc cổng không còn phản hồi -> dọn dẹp cache
		discCache.info = nil
		discCache.cachedAt = time.Time{}
	}
	discCache.mu.Unlock()

	// 2. Gom nhiều cuộc gọi đồng thời qua singleflight
	res, err, _ := discFlight.Do("discover_language_server", func() (any, error) {
		procs, err := QueryLanguageServerProcesses()
		if err != nil {
			return nil, fmt.Errorf("failed querying processes: %w", err)
		}
		if len(procs) == 0 {
			return nil, ErrLanguageServerNotFound
		}

		for _, p := range procs {
			if p.PID <= 0 || strings.TrimSpace(p.CommandLine) == "" {
				continue
			}
			csrfToken := ExtractCSRFToken(p.CommandLine)
			if csrfToken == "" {
				continue
			}
			port, err := DiscoverListeningPort(p.PID)
			if err != nil || port <= 0 {
				continue
			}

			var creationTime windows.Filetime
			if hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(p.PID)); err == nil {
				var et, kt, ut windows.Filetime
				_ = windows.GetProcessTimes(hProcess, &creationTime, &et, &kt, &ut)
				_ = windows.CloseHandle(hProcess)
			}

			info := &ServerInfo{
				PID:       p.PID,
				CSRFToken: csrfToken,
				Port:      port,
			}

			discCache.mu.Lock()
			discCache.info = info
			discCache.creationTime = creationTime
			discCache.cachedAt = time.Now()
			discCache.mu.Unlock()

			return info, nil
		}
		return nil, ErrLanguageServerNotFound
	})

	if err != nil {
		return nil, err
	}
	return res.(*ServerInfo), nil
}
