package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/model"
	"antigravity-switch/internal/reloader"
	"antigravity-switch/internal/store"
	"antigravity-switch/internal/telemetry"
	"antigravity-switch/internal/wincred"
)

var (
	flagCLI     = flag.Bool("cli", false, "Run in command-line interface mode")
	flagGUI     = flag.Bool("gui", false, "Force graphical interface mode")
	flagSwitch  = flag.String("switch", "", "Switch Antigravity session to specified Google account email")
	flagStatus  = flag.Bool("status", false, "Display current Antigravity identity and Language Server status")
	flagList    = flag.Bool("list", false, "List all discovered accounts across Chrome and accounts_store/")
	flagCapture = flag.Bool("capture", false, "Capture active Windows Credential token and save to accounts_store/")
	flagQuota   = flag.Bool("quota", false, "Display detailed multi-account quota telemetry table")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "⚡ Antigravity Multi-Account Switcher (Go Native Engine)\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  antigravity-switch [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  antigravity-switch                      # Launch Fyne native GUI (default)\n")
		fmt.Fprintf(os.Stderr, "  antigravity-switch --list               # List all accounts\n")
		fmt.Fprintf(os.Stderr, "  antigravity-switch --status             # Show active session and server status\n")
		fmt.Fprintf(os.Stderr, "  antigravity-switch --quota              # Display detailed quota telemetry table\n")
		fmt.Fprintf(os.Stderr, "  antigravity-switch --switch dev@work.com # Hot-swap to dev@work.com\n")
		fmt.Fprintf(os.Stderr, "  antigravity-switch --capture            # Capture active credential to store\n")
	}

	flag.Parse()

	// Initialize core orchestrator dependencies
	accStore, err := store.NewAccountStore("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing account store: %v\n", err)
		os.Exit(1)
	}

	sessionReloader := reloader.NewSessionReloader()
	mgr, err := manager.NewAccountManager(accStore, sessionReloader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing account manager: %v\n", err)
		os.Exit(1)
	}

	// Determine execution mode: CLI vs GUI
	isCLIAction := *flagCLI || *flagStatus || *flagList || *flagCapture || *flagQuota || *flagSwitch != ""
	if !isCLIAction || *flagGUI {
		if err := launchGUI(mgr, *flagStatus, *flagList, *flagCapture, *flagSwitch, *flagQuota); err != nil {
			fmt.Fprintf(os.Stderr, "Error in GUI launch: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Run CLI actions
	runCLI(mgr, *flagStatus, *flagList, *flagCapture, *flagSwitch, *flagQuota)
}

func runCLI(mgr *manager.AccountManager, showStatus, showList, doCapture bool, switchTarget string, showQuota bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if doCapture {
		fmt.Println("[INFO] Capturing active Antigravity session from Windows Credential Manager...")
		email, err := mgr.CaptureCurrentSession()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Capture failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[SUCCESS] Saved active session for '%s' into accounts_store/%s.json\n", email, email)
		return
	}

	if switchTarget != "" {
		fmt.Printf("[INFO] Initiating hot-swap to target account: %s...\n", switchTarget)
		res, err := mgr.SwitchAccount(ctx, switchTarget, false) // forceLogout = false: smooth zero-chrome hot-swap
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Hot-swap failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[SUCCESS] %s (Duration: %s)\n", res.Message, res.Duration.Round(time.Millisecond))
		return
	}

	if showStatus {
		fmt.Println("================================================================================")
		fmt.Println("⚡ ANTIGRAVITY SESSION STATUS")
		fmt.Println("================================================================================")

		activeEmail, err := mgr.GetCurrentActiveEmail(ctx)
		if err != nil || activeEmail == "" {
			fmt.Println("Active Email:    (No active session detected)")
		} else {
			fmt.Printf("Active Email:    %s\n", activeEmail)
		}

		proc, err := connector.DiscoverLanguageServer()
		if err != nil {
			fmt.Println("Language Server: Not Running (Using Credential Manager fallback)")
		} else {
			fmt.Printf("Language Server: Running (PID: %d | Port: %d | CSRF: Present)\n", proc.PID, proc.Port)
		}

		token, err := wincred.ReadToken()
		if err != nil || token == nil {
			fmt.Println("Win32 Credential: None")
		} else {
			fmt.Printf("Win32 Credential: Present (%d bytes access token)\n", len(token.AccessToken))
		}
		fmt.Println("================================================================================")
		return
	}

	if showQuota {
		if err := runQuotaCLI(os.Stdout, mgr, ctx); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed retrieving quota telemetry: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Default CLI action or --list: print full accounts overview table
	if showList || true {
		overview, err := mgr.GetAccountStates(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed resolving account states: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("================================================================================")
		fmt.Println("⚡ ANTIGRAVITY ACCOUNTS OVERVIEW")
		fmt.Println("================================================================================")
		if overview.ActiveEmail != "" {
			fmt.Printf("ACTIVE IDENTITY:  %s\n", overview.ActiveEmail)
		} else {
			fmt.Println("ACTIVE IDENTITY:  (None)")
		}

		if overview.LanguageServerRunning && overview.ServerInfo != nil {
			fmt.Printf("LANGUAGE SERVER:  PID %d | Port %d (Running)\n", overview.ServerInfo.PID, overview.ServerInfo.Port)
		} else {
			fmt.Println("LANGUAGE SERVER:  Not detected (Credential fallback mode)")
		}
		fmt.Println("--------------------------------------------------------------------------------")

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "STATUS\tEMAIL\tNAME\tSTATE\tTOKEN STORED\tPROFILE")
		fmt.Fprintln(w, "------\t-----\t----\t-----\t------------\t-------")

		storedCount := 0
		for _, acc := range overview.Accounts {
			badge := acc.DisplayBadge
			name := acc.Name
			if name == "" {
				name = "-"
			}
			profile := acc.ProfileDir
			if profile == "" {
				profile = "-"
			}

			tokenStr := "NO"
			if acc.HasToken {
				tokenStr = "YES"
				storedCount++
			}

			statusTag := ""
			switch acc.State {
			case model.StateActive:
				statusTag = "[ACTIVE]"
			case model.StateReady:
				statusTag = "[READY]"
			default:
				statusTag = "[UNAUTH]"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", statusTag, acc.Email, name, strings.ToUpper(string(acc.State)), tokenStr, profile)
			_ = badge
		}
		w.Flush()

		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("Total: %d accounts (%d stored in JSON, %d on Chrome)\n", len(overview.Accounts), storedCount, len(overview.Accounts))
		fmt.Println("================================================================================")
	}
}

func runQuotaCLI(w io.Writer, mgr *manager.AccountManager, ctx context.Context) error {
	cachePath := telemetry.DefaultQuotaCachePath
	if mgr != nil && mgr.Store() != nil && mgr.Store().GetStoreDir() != "" {
		cachePath = filepath.Join(mgr.Store().GetStoreDir(), ".quota_cache.json")
	}

	cache := telemetry.NewQuotaCache(cachePath)
	_ = cache.LoadFromDisk()

	// Initialize TelemetryPool backed by store
	var pool *telemetry.TelemetryPool
	if mgr != nil && mgr.Store() != nil {
		refresher := connector.NewGoogleOAuthRefresher(nil)
		client := telemetry.NewQuotaClient(refresher, nil)
		p, err := telemetry.NewTelemetryPool(client, cache, mgr.Store())
		if err == nil {
			pool = p
		}
	}

	// Determine if cache refresh is needed
	var storedAccounts []string
	if mgr != nil && mgr.Store() != nil {
		storedAccounts, _ = mgr.Store().ListAccounts()
	}

	needsRefresh := false
	if len(storedAccounts) > 0 {
		for _, email := range storedAccounts {
			data, exists := cache.Get(email)
			if !exists || data == nil || time.Since(data.LastUpdated) > 5*time.Minute {
				needsRefresh = true
				break
			}
		}
	} else if len(cache.GetAll()) == 0 {
		needsRefresh = true
	}

	if needsRefresh && pool != nil {
		fmt.Fprintln(w, "[INFO] Fetching latest quota telemetry from Google Cloud Code API...")
		pollCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
		_, _ = pool.PollAll(pollCtx)
		cancel()
	}

	return renderQuotaTable(w, mgr, cache, ctx)
}

func renderQuotaTable(w io.Writer, mgr *manager.AccountManager, cache *telemetry.QuotaCache, ctx context.Context) error {
	overview, err := mgr.GetAccountStates(ctx)
	if err != nil {
		return fmt.Errorf("failed resolving account states: %w", err)
	}

	fmt.Fprintln(w, "================================================================================")
	fmt.Fprintln(w, "⚡ ANTIGRAVITY MULTI-ACCOUNT QUOTA TELEMETRY")
	fmt.Fprintln(w, "================================================================================")
	if overview.ActiveEmail != "" {
		fmt.Fprintf(w, "ACTIVE IDENTITY:  %s\n", overview.ActiveEmail)
	} else {
		fmt.Fprintln(w, "ACTIVE IDENTITY:  (None)")
	}

	if overview.LanguageServerRunning && overview.ServerInfo != nil {
		fmt.Fprintf(w, "LANGUAGE SERVER:  PID %d | Port %d (Running)\n", overview.ServerInfo.PID, overview.ServerInfo.Port)
	} else {
		fmt.Fprintln(w, "LANGUAGE SERVER:  Not detected (Credential fallback mode)")
	}
	fmt.Fprintln(w, "--------------------------------------------------------------------------------")

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "EMAIL\tCLAUDE 4.6\tGEMINI 2.5\tGPT-OSS\tRESET TIME (UTC)\tHEALTH STATUS")
	fmt.Fprintln(tw, "-----\t----------\t----------\t-------\t----------------\t-------------")

	activeCount := 0
	readyCount := 0
	lowQuotaCount := 0

	for _, acc := range overview.Accounts {
		var (
			claudeStr = "N/A"
			geminiStr = "N/A"
			gptStr    = "N/A"
			resetStr  = "-"
			statusStr = "Ready"
		)

		isLowQuota := false
		if cache != nil {
			if q, ok := cache.Get(acc.Email); ok && q != nil {
				if q.Claude.RemainingFraction >= 0 {
					claudeStr = fmt.Sprintf("%.1f%%", q.Claude.RemainingFraction*100)
				}
				if q.Gemini.RemainingFraction >= 0 {
					geminiStr = fmt.Sprintf("%.1f%%", q.Gemini.RemainingFraction*100)
				}
				if q.GPT.RemainingFraction >= 0 {
					gptStr = fmt.Sprintf("%.1f%%", q.GPT.RemainingFraction*100)
				}

				if !q.ResetTime.IsZero() {
					resetStr = q.ResetTime.UTC().Format("15:04:05 UTC")
				}

				if q.Status == telemetry.StatusDepleted || strings.HasPrefix(q.Status, "Low") ||
					q.Claude.RemainingFraction < 0.15 || q.Gemini.RemainingFraction < 0.15 || q.GPT.RemainingFraction < 0.15 {
					isLowQuota = true
				}
			}
		}

		if acc.State == model.StateActive {
			activeCount++
			if isLowQuota {
				statusStr = "Active (Low Quota)"
				lowQuotaCount++
			} else {
				statusStr = "Active"
			}
		} else if isLowQuota {
			statusStr = "Low Quota"
			lowQuotaCount++
		} else if acc.HasToken {
			statusStr = "Ready"
			readyCount++
		} else {
			statusStr = "Unauthenticated"
		}

		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", acc.Email, claudeStr, geminiStr, gptStr, resetStr, statusStr)
	}
	tw.Flush()

	fmt.Fprintln(w, "--------------------------------------------------------------------------------")
	fmt.Fprintf(w, "Total: %d accounts (%d Active, %d Ready, %d Low Quota)\n", len(overview.Accounts), activeCount, readyCount, lowQuotaCount)
	fmt.Fprintln(w, "================================================================================")
	return nil
}
