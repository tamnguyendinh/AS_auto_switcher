package connector

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	ErrLanguageServerNotFound = errors.New("language_server.exe process not found or not ready")
	ErrNoListeningPort        = errors.New("no loopback listening port found for language_server.exe")
	ErrInvalidCSRFToken       = errors.New("invalid or missing CSRF token in language_server command line")
)

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

// QueryLanguageServerProcesses finds running instances of language_server.exe on Windows.
func QueryLanguageServerProcesses() ([]ProcessCandidate, error) {
	psScript := `Get-CimInstance Win32_Process -Filter "Name = 'language_server.exe'" | Select-Object ProcessId, CommandLine | ConvertTo-Json -Compress`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", psScript)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("querying Win32_Process via PowerShell failed: %w", err)
	}

	trimmed := bytes.TrimSpace(out)
	if len(trimmed) == 0 {
		return nil, nil
	}

	// Case 1: Single process returned as JSON object
	var single ProcessCandidate
	if err := json.Unmarshal(trimmed, &single); err == nil && single.PID > 0 {
		return []ProcessCandidate{single}, nil
	}

	// Case 2: Multiple processes returned as JSON array
	var list []ProcessCandidate
	if err := json.Unmarshal(trimmed, &list); err == nil {
		return list, nil
	}

	return nil, fmt.Errorf("unexpected json output from Win32_Process: %s", string(trimmed))
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

// discoverPortsViaNetstat executes netstat -ano -p tcp to locate listening ports.
func discoverPortsViaNetstat(pid int) ([]int, error) {
	if pid <= 0 {
		return nil, nil
	}

	cmd := exec.Command("netstat", "-ano", "-p", "tcp")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("netstat failed: %w", err)
	}

	targetPIDStr := strconv.Itoa(pid)
	portsMap := make(map[int]struct{})
	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 5 {
			proto := strings.ToUpper(parts[0])
			localAddr := parts[1]
			state := strings.ToUpper(parts[3])
			ownerPID := parts[4]

			if proto == "TCP" && state == "LISTENING" && ownerPID == targetPIDStr {
				idx := strings.LastIndex(localAddr, ":")
				if idx != -1 && idx+1 < len(localAddr) {
					portStr := localAddr[idx+1:]
					if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
						host := strings.Trim(localAddr[:idx], "[]")
						if host == "" || host == "127.0.0.1" || host == "0.0.0.0" || host == "::1" || host == "::" || host == "*" || host == "localhost" {
							portsMap[p] = struct{}{}
						}
					}
				}
			}
		}
	}

	ports := make([]int, 0, len(portsMap))
	for p := range portsMap {
		ports = append(ports, p)
	}
	sort.Ints(ports)
	return ports, nil
}

// DiscoverListeningPorts discovers all loopback TCP listening ports owned by the given PID.
// Uses fast Win32 GetExtendedTcpTable first, falling back to netstat.
func DiscoverListeningPorts(pid int) ([]int, error) {
	if pid <= 0 {
		return nil, errors.New("invalid PID")
	}

	// Fast path: Win32 GetExtendedTcpTable
	if ports, err := discoverPortsViaWin32(pid); err == nil && len(ports) > 0 {
		return ports, nil
	}

	// Fallback path: netstat
	if ports, err := discoverPortsViaNetstat(pid); err == nil && len(ports) > 0 {
		return ports, nil
	}

	return nil, fmt.Errorf("no listening ports found for PID %d", pid)
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

	// Probe each port to find HTTP 200 listener
	for _, p := range ports {
		if ProbeHTTPPort(p, 250*time.Millisecond) {
			return p, nil
		}
	}

	// Default to lowest port if probing does not match
	return ports[0], nil
}

// DiscoverLanguageServer scans running processes for language_server.exe,
// extracts the CSRF token and loopback listening port.
func DiscoverLanguageServer() (*ServerInfo, error) {
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

		return &ServerInfo{
			PID:       p.PID,
			CSRFToken: csrfToken,
			Port:      port,
		}, nil
	}

	return nil, ErrLanguageServerNotFound
}
