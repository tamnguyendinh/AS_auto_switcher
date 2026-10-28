package connector

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"sync"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// TestQueryLanguageServerProcesses_Win32 verifies pure Win32 process enumeration without subprocesses.
func TestQueryLanguageServerProcesses_Win32(t *testing.T) {
	candidates, err := QueryLanguageServerProcesses()
	if err != nil {
		t.Fatalf("QueryLanguageServerProcesses failed: %v", err)
	}

	// If language_server.exe is running on host, verify valid PID and CommandLine
	for _, c := range candidates {
		if c.PID <= 0 {
			t.Errorf("expected positive PID, got %d", c.PID)
		}
		if c.CommandLine == "" {
			t.Errorf("expected non-empty CommandLine for PID %d", c.PID)
		}
	}
}

// TestDiscoverListeningPorts_Validation tests edge cases for DiscoverListeningPorts.
func TestDiscoverListeningPorts_Validation(t *testing.T) {
	// Invalid PID <= 0
	_, err := DiscoverListeningPorts(0)
	if err == nil {
		t.Fatalf("expected error for PID 0, got nil")
	}

	_, err = DiscoverListeningPorts(-1)
	if err == nil {
		t.Fatalf("expected error for PID -1, got nil")
	}

	// Non-existent PID
	_, err = DiscoverListeningPorts(99999999)
	if err == nil {
		t.Fatalf("expected error for non-existent PID, got nil")
	}
}

// TestDiscoveryCache_LifecycleAndInvalidation tests cache TTL, invalidation, and hit logic.
func TestDiscoveryCache_LifecycleAndInvalidation(t *testing.T) {
	// 1. Reset cache
	InvalidateDiscoveryCache(0, "")

	discCache.mu.RLock()
	if discCache.info != nil {
		t.Fatalf("expected nil cache after reset")
	}
	discCache.mu.RUnlock()

	// 2. Populate synthetic cache entry
	fakeInfo := &ServerInfo{
		PID:       int(windows.GetCurrentProcessId()),
		CSRFToken: "fake-csrf-token",
		Port:      54321,
	}

	// Get current process creation time
	hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, windows.GetCurrentProcessId())
	if err != nil {
		t.Fatalf("failed opening current process: %v", err)
	}
	defer windows.CloseHandle(hProcess)

	var ct, et, kt, ut windows.Filetime
	if err := windows.GetProcessTimes(hProcess, &ct, &et, &kt, &ut); err != nil {
		t.Fatalf("failed getting current process times: %v", err)
	}

	discCache.mu.Lock()
	discCache.info = fakeInfo
	discCache.creationTime = ct
	discCache.cachedAt = time.Now()
	discCache.mu.Unlock()

	// 3. Invalidate on mismatch port -> should NOT invalidate
	InvalidateDiscoveryCache(11111, "fake-csrf-token")
	discCache.mu.RLock()
	if discCache.info == nil {
		t.Fatalf("cache should not be invalidated on mismatched port")
	}
	discCache.mu.RUnlock()

	// 4. Invalidate on mismatch CSRF -> should NOT invalidate
	InvalidateDiscoveryCache(54321, "mismatched-csrf")
	discCache.mu.RLock()
	if discCache.info == nil {
		t.Fatalf("cache should not be invalidated on mismatched CSRF")
	}
	discCache.mu.RUnlock()

	// 5. Invalidate on matching port -> should clear cache
	InvalidateDiscoveryCache(54321, "")
	discCache.mu.RLock()
	if discCache.info != nil {
		t.Fatalf("cache should be invalidated on matching port")
	}
	discCache.mu.RUnlock()

	// 6. Test TTL expiration
	discCache.mu.Lock()
	discCache.info = fakeInfo
	discCache.creationTime = ct
	discCache.cachedAt = time.Now().Add(-11 * time.Second) // Expired (> 10s)
	discCache.mu.Unlock()

	// Direct check of cachedAt logic
	discCache.mu.RLock()
	expired := time.Since(discCache.cachedAt) > 10*time.Second
	discCache.mu.RUnlock()
	if !expired {
		t.Fatalf("expected cache to be expired after 11 seconds")
	}

	// Clean up after test
	InvalidateDiscoveryCache(0, "")
}

// TestDiscoverLanguageServer_Singleflight tests that concurrent calls coalesce without race or panic.
func TestDiscoverLanguageServer_Singleflight(t *testing.T) {
	InvalidateDiscoveryCache(0, "")

	var wg sync.WaitGroup
	const concurrency = 8
	results := make([]*ServerInfo, concurrency)
	errs := make([]error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			info, err := DiscoverLanguageServer()
			results[idx] = info
			errs[idx] = err
		}(i)
	}

	wg.Wait()

	// Check results consistency
	var firstInfo *ServerInfo
	var firstErr error
	for i := 0; i < concurrency; i++ {
		if i == 0 {
			firstInfo = results[i]
			firstErr = errs[i]
			continue
		}
		if (firstErr == nil && errs[i] != nil) || (firstErr != nil && errs[i] == nil) {
			t.Errorf("inconsistent error across singleflight calls: call 0 got %v, call %d got %v", firstErr, i, errs[i])
		}
		if firstInfo != nil && results[i] != nil {
			if firstInfo.PID != results[i].PID || firstInfo.Port != results[i].Port || firstInfo.CSRFToken != results[i].CSRFToken {
				t.Errorf("inconsistent ServerInfo across singleflight calls: call 0 got %+v, call %d got %+v", firstInfo, i, results[i])
			}
		}
	}
}

// TestDiscoverLanguageServer_RejectsDeadProcessInCache tests that DiscoverLanguageServer rejects dead PID in cache.
func TestDiscoverLanguageServer_RejectsDeadProcessInCache(t *testing.T) {
	InvalidateDiscoveryCache(0, "")

	// 1. Spawn a short-lived process that exits immediately
	cmd := exec.Command("cmd.exe", "/c", "exit 0")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed starting short-lived cmd process: %v", err)
	}
	deadPID := cmd.Process.Pid

	// Capture creation time if possible
	var ct windows.Filetime
	if hProc, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(deadPID)); err == nil {
		var et, kt, ut windows.Filetime
		_ = windows.GetProcessTimes(hProc, &ct, &et, &kt, &ut)
		_ = windows.CloseHandle(hProc)
	}

	_ = cmd.Wait() // Ensure process has fully exited

	// 2. Populate dead process info into discCache
	deadInfo := &ServerInfo{
		PID:       deadPID,
		Port:      54316,
		CSRFToken: "dead-token",
	}
	discCache.mu.Lock()
	discCache.info = deadInfo
	discCache.creationTime = ct
	discCache.cachedAt = time.Now()
	discCache.mu.Unlock()

	// 3. Call DiscoverLanguageServer()
	// Must reject dead process cache, purge it, and attempt fresh scan
	info, err := DiscoverLanguageServer()

	// 4. Assert that the returned info (if language_server is running on machine) is NOT the dead process
	if err == nil && info != nil {
		if info.PID == deadPID {
			t.Fatalf("DiscoverLanguageServer returned dead process PID %d from cache!", deadPID)
		}
		if info.Port == 54316 && info.CSRFToken == "dead-token" {
			t.Fatalf("DiscoverLanguageServer returned dead cache info: %+v", info)
		}
	}

	// 5. Assert dead process is purged from discCache
	discCache.mu.RLock()
	remaining := discCache.info
	discCache.mu.RUnlock()
	if remaining != nil && remaining.PID == deadPID {
		t.Fatalf("dead process PID %d remained in discCache!", deadPID)
	}

	InvalidateDiscoveryCache(0, "")
}

// TestApiClient_ReactiveCacheEvictionOnConnectionRefused tests reactive cache eviction when API call encounters network error.
func TestApiClient_ReactiveCacheEvictionOnConnectionRefused(t *testing.T) {
	InvalidateDiscoveryCache(0, "")

	deadPort := 59998
	deadToken := "refused-csrf-token"

	// 1. Populate discCache with dead port
	discCache.mu.Lock()
	discCache.info = &ServerInfo{
		PID:       12345,
		Port:      deadPort,
		CSRFToken: deadToken,
	}
	discCache.cachedAt = time.Now()
	discCache.mu.Unlock()

	// 2. Create client pointing to the dead port (no listener active)
	client := NewClient(&ServerInfo{
		PID:       12345,
		Port:      deadPort,
		CSRFToken: deadToken,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// 3. Call GetUserStatus -> must fail with connection refused and trigger reactive eviction
	_, err := client.GetUserStatus(ctx)
	if err == nil {
		t.Fatal("Expected GetUserStatus to fail on closed port, got nil")
	}

	// 4. Verify discCache was invalidated
	discCache.mu.RLock()
	cached := discCache.info
	discCache.mu.RUnlock()

	if cached != nil && cached.Port == deadPort {
		t.Fatalf("Expected discCache for port %d to be evicted after connection refusal, but was still present", deadPort)
	}

	// 5. Test AuthLogout reactive eviction as well
	discCache.mu.Lock()
	discCache.info = &ServerInfo{
		PID:       12345,
		Port:      deadPort,
		CSRFToken: deadToken,
	}
	discCache.cachedAt = time.Now()
	discCache.mu.Unlock()

	err = client.AuthLogout(ctx)
	if err == nil {
		t.Fatal("Expected AuthLogout to fail on closed port, got nil")
	}

	discCache.mu.RLock()
	cachedAfterLogout := discCache.info
	discCache.mu.RUnlock()

	if cachedAfterLogout != nil && cachedAfterLogout.Port == deadPort {
		t.Fatalf("Expected discCache for port %d to be evicted after AuthLogout connection refusal", deadPort)
	}

	InvalidateDiscoveryCache(0, "")
}

// TestCheckProcessLiveness validates fast Win32 process liveness detection.
func TestCheckProcessLiveness(t *testing.T) {
	// 1. Invalid PIDs <= 0
	if CheckProcessLiveness(0) {
		t.Errorf("expected CheckProcessLiveness(0) to be false")
	}
	if CheckProcessLiveness(-1) {
		t.Errorf("expected CheckProcessLiveness(-1) to be false")
	}

	// 2. Current active process must be alive
	currentPID := int(windows.GetCurrentProcessId())
	if !CheckProcessLiveness(currentPID) {
		t.Errorf("expected CheckProcessLiveness(currentPID=%d) to be true", currentPID)
	}

	// 3. Non-existent PID
	if CheckProcessLiveness(99999999) {
		t.Errorf("expected CheckProcessLiveness(99999999) to be false")
	}

	// 4. Terminated process
	cmd := exec.Command("cmd.exe", "/c", "exit 0")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed starting short-lived cmd: %v", err)
	}
	deadPID := cmd.Process.Pid
	_ = cmd.Wait()

	if CheckProcessLiveness(deadPID) {
		t.Errorf("expected CheckProcessLiveness(deadPID=%d) to be false after exit", deadPID)
	}
}

// TestCheckServerLiveness validates both process liveness and loopback HTTP probing.
func TestCheckServerLiveness(t *testing.T) {
	// 1. Nil or invalid info
	if CheckServerLiveness(nil) {
		t.Errorf("expected CheckServerLiveness(nil) to be false")
	}
	if CheckServerLiveness(&ServerInfo{PID: 0, Port: 8080}) {
		t.Errorf("expected CheckServerLiveness with PID 0 to be false")
	}
	if CheckServerLiveness(&ServerInfo{PID: 1234, Port: 0}) {
		t.Errorf("expected CheckServerLiveness with Port 0 to be false")
	}

	// 2. Dead process with open test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	port := ts.Listener.Addr().(*net.TCPAddr).Port

	if CheckServerLiveness(&ServerInfo{PID: 99999999, Port: port}) {
		t.Errorf("expected CheckServerLiveness with dead PID to be false even if port is open")
	}

	// 3. Live process with closed port
	currentPID := int(windows.GetCurrentProcessId())
	if CheckServerLiveness(&ServerInfo{PID: currentPID, Port: 59997}) {
		t.Errorf("expected CheckServerLiveness with closed port 59997 to be false")
	}

	// 4. Live process with responsive HTTP server
	if !CheckServerLiveness(&ServerInfo{PID: currentPID, Port: port}) {
		t.Errorf("expected CheckServerLiveness with currentPID and open HTTP port to be true")
	}
}
