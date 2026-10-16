package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/model"
	"antigravity-switch/internal/reloader"
	"antigravity-switch/internal/store"
	"antigravity-switch/internal/wincred"
)

func main() {
	var (
		flagCLI     = flag.Bool("cli", false, "Run in command-line interface mode")
		flagGUI     = flag.Bool("gui", false, "Force graphical interface mode")
		flagSwitch  = flag.String("switch", "", "Switch Antigravity session to specified Google account email")
		flagStatus  = flag.Bool("status", false, "Display current Antigravity identity and Language Server status")
		flagList    = flag.Bool("list", false, "List all discovered accounts across Chrome and accounts_store/")
		flagCapture = flag.Bool("capture", false, "Capture active Windows Credential token and save to accounts_store/")
	)

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
	isCLIAction := *flagCLI || *flagStatus || *flagList || *flagCapture || *flagSwitch != ""
	if !isCLIAction || *flagGUI {
		if err := launchGUI(mgr, *flagStatus, *flagList, *flagCapture, *flagSwitch); err != nil {
			fmt.Fprintf(os.Stderr, "Error in GUI launch: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Run CLI actions
	runCLI(mgr, *flagStatus, *flagList, *flagCapture, *flagSwitch)
}

func runCLI(mgr *manager.AccountManager, showStatus, showList, doCapture bool, switchTarget string) {
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
		res, err := mgr.SwitchAccount(ctx, switchTarget, false)
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
