# ⚡ Antigravity Multi-Account Switcher - **Version 1.0.3.0**

[🇻🇳 Tiếng Việt](README_VN.md) | [🇬🇧 English](README_EN.md)

If you have a personal account, a work account, and a freelance/side-project account, and you waste valuable time signing in and out whenever switching accounts—this tool is built for you.

### **When accounts switch automatically, all active work continues smoothly without interruption or manual intervention.**

### **Safe and compliant with Google-approved account switching mechanisms. No modification to Antigravity source code, no API wrapping.**

A native Windows application written in **Go** that lets you manage multiple Google accounts, monitor quotas in real time, and switch active sessions in Antigravity 2.x smoothly and securely. Built with a modern, intuitive **Fyne** desktop GUI, it provides a seamless Zero-Console experience packaged into a single standalone executable `antigravity-switch.exe`.

The application features a built-in **Autonomous Multi-Account Switcher & Failover Daemon**. You can easily choose between manual switching (via the **Switch** button on the GUI) and fully autonomous switching (AutoSwitch). When AutoSwitch is enabled, the daemon monitors quota availability, detects exhaustion, performs in-memory backoff retries, and seamlessly fails over to the most optimal backup account using the **Burn-down Sorter** algorithm—keeping your coding flow uninterrupted.

## ✨ Key Highlights & User Experience

- **Intuitive Account Grid:** View your accounts at a glance with clear, distinctive status badges:
  - `[✔ Active]`: Currently active account inside Antigravity IDE.
  - `[Switch]`: Switch to this account instantly with a single click.
  - `[Needs OAuth]`: Reminds you to log into the IDE to grant authorization.
- **Safe Delete Button `[ 🗑 ]`:** Clean up inactive accounts easily. Includes a danger confirmation dialog to protect against accidentally deleting the active account.
- **Autonomous AutoSwitch Toggle:** Seamlessly toggle automated account failover when quota is exhausted, ensuring non-stop developer productivity.
- **Real-Time Status Bar:** Live updates of the system state (`IDLE`, `EVALUATING`, `SWITCHING`, `SWITCHED`...).
- **Comprehensive Quota Dashboard:** Visually track remaining quota percentages for Gemini and Claude across both 5-hour and Weekly reset windows. Progress bars dynamically shift color based on availability (green: plentiful, yellow: moderate, red: near exhaustion).
- **Instant Quota & Account Refresh:** Refresh the entire list or individual accounts in a split second.
- **3 Modern Themes:** Switch between **Dark**, **Light**, and **Warm** themes to suit your viewing preference.
- **Always-on-Top Pinning:** Keep the switcher floating on top while working to keep an eye on quotas.
- **System Tray Minimization:** Minimize to the Windows Taskbar/System Tray when not in focus, with quick access via context menu.

## 🚀 Installation & Getting Started

### 1. Portable Executable (No Installation Required)

The application is distributed as a **single standalone executable `antigravity-switch.exe`** with a Zero-Console experience (no background command prompt popup):
- Download `antigravity-switch.exe` (or grab it from `bin\antigravity-switch.exe`).
- Double-click the executable to launch the GUI immediately.

### 2. Quick Start Guide

1. **Save an Account:** Sign in to your Google account within **Antigravity IDE**, open the Switcher, and click **Account List** to capture the active session. Repeat for any additional accounts.
2. **Switch Manually:** Click the **Switch** button on any saved account card to switch your active working account.
3. **Switch Automatically:** Turn on the **AutoSwitch** toggle. The application will monitor your quota and automatically rotate to the best available backup account when limits are reached—without manual input.

### 3. Building from Source

For developers who want to compile the binary from source:

**Prerequisites:**
- **Go 1.22+** (latest stable release recommended).
- **GCC (MinGW-w64 / MSYS2 UCRT64)** installed and configured in your system `PATH` for CGO support.

**Build Command:**
Open PowerShell in the project root directory and run the build automation script:
```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1
```

Once built, the compiled binary will be placed at `bin\antigravity-switch.exe`.

## 🔒 Safety & Security

- **100% Local & Offline:** Never contacts any third-party servers. All account session data is stored securely on your local machine.
- **Standard-Compliant:** The switching mechanism is fully compliant with Antigravity IDE conventions. It does not alter original IDE binaries and does not use API proxy wrappers.
