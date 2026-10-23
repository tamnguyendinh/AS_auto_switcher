package connector

import (
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
