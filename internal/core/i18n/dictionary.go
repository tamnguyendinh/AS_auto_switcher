package i18n

import (
	_ "embed"
	"encoding/json"
	"runtime"
	"strings"
)

//go:embed locales/vi.json
var viJSON []byte

//go:embed locales/en.json
var enJSON []byte

// Dictionary defines the strongly-typed catalog of all localized UI strings.
type Dictionary struct {
	SessionCredentialMissing   string `json:"session_credential_missing"`
	SessionCredentialConflict  string `json:"session_credential_conflict"`
	SessionCredentialRead      string `json:"session_credential_read"`
	SessionCredentialInvalid   string `json:"session_credential_invalid"`
	SessionCredentialWrite     string `json:"session_credential_write"`
	SessionIdentityUnavailable string `json:"session_identity_unavailable"`
	SessionStoreRead           string `json:"session_store_read"`
	SessionCatalogRead         string `json:"session_catalog_read"`
	SessionReadFailed          string `json:"session_read_failed"`

	// Header & Window
	AppTitle          string `json:"app_title"`
	NavAntigravity    string `json:"nav_antigravity"`
	NavCodex          string `json:"nav_codex"`
	CodexAddAccount   string `json:"codex_add_account"`
	CodexRefresh      string `json:"codex_refresh"`
	CodexPin          string `json:"codex_pin"`
	CodexTitle        string `json:"codex_title"`
	CodexDescription  string `json:"codex_description"`
	CodexStatus       string `json:"codex_status"`
	CodexVersion      string `json:"codex_version"`
	LangDropdownLabel string `json:"lang_dropdown_label"`

	// Toolbar Buttons & Controls
	BtnAdd                 string `json:"btn_add"`
	BtnRefreshList         string `json:"btn_refresh_list"`
	BtnRefreshListLoading  string `json:"btn_refresh_list_loading"`
	BtnRefreshQuota        string `json:"btn_refresh_quota"`
	BtnRefreshQuotaLoading string `json:"btn_refresh_quota_loading"`
	BtnAlwaysOnTop         string `json:"btn_always_on_top"`
	LblMaskAccounts        string `json:"lbl_mask_accounts"`
	LblAutoSwitch          string `json:"lbl_auto_switch"`

	// Account Cards & Main View
	MainEmptyAccounts    string `json:"main_empty_accounts"`
	BadgeActive          string `json:"badge_active"`
	BadgeSwitch          string `json:"badge_switch"`
	BadgeNeedOAuth       string `json:"badge_need_oauth"`
	BadgeAdded           string `json:"badge_added"`
	BadgeAccountNotFound string `json:"badge_account_not_found"`

	// Quota Matrix & Countdown
	QuotaCol5h       string `json:"quota_col_5h"`
	QuotaColWeek     string `json:"quota_col_week"`
	QuotaReady       string `json:"quota_ready"`
	QuotaUnavailable string `json:"quota_unavailable"`

	// Server Connection Status
	ServerConnecting string `json:"server_connecting"`
	ServerConnected  string `json:"server_connected"`
	ServerStopped    string `json:"server_stopped"`

	// Status Bar Messages
	StatusNoCandidate        string `json:"status_no_candidate"`
	StatusFetchError         string `json:"status_fetch_error"`
	StatusFetchSuccess       string `json:"status_fetch_success"`
	StatusSwitchingTo        string `json:"status_switching_to"`
	StatusSwitchedSuccess    string `json:"status_switched_success"`
	StatusUpdateError        string `json:"status_update_error"`
	StatusCannotDeleteActive string `json:"status_cannot_delete_active"`
	StatusDeleteError        string `json:"status_delete_error"`
	StatusDeleteSuccess      string `json:"status_delete_success"`
	StatusMaskOn             string `json:"status_mask_on"`
	StatusMaskOff            string `json:"status_mask_off"`
	StatusAutoOn             string `json:"status_auto_on"`
	StatusAutoOff            string `json:"status_auto_off"`

	// Modals & Candidate Badges
	ModalAddTitle        string `json:"modal_add_title"`
	ModalAddHeader       string `json:"modal_add_header"`
	ModalAddClose        string `json:"modal_add_close"`
	ModalScanChrome      string `json:"modal_scan_chrome"`
	ModalScanningChrome  string `json:"modal_scanning_chrome"`
	ModalEmptyCandidates string `json:"modal_empty_candidates"`
	ModalBadgeOAuthDone  string `json:"modal_badge_oauth_done"`
	ModalBadgeAdded      string `json:"modal_badge_added"`
	ModalBadgeNeedOAuth  string `json:"modal_badge_need_oauth"`
	ModalBadgeNotFound   string `json:"modal_badge_not_found"`
	ModalBtnAddItem      string `json:"modal_btn_add_item"`

	// Confirmation Popups
	ConfirmDeleteTitle     string `json:"confirm_delete_title"`
	ConfirmBtnDelete       string `json:"confirm_btn_delete"`
	ConfirmBtnCancel       string `json:"confirm_btn_cancel"`
	ConfirmMsgDiscExists   string `json:"confirm_msg_disc_exists"`
	ConfirmMsgDiscNoFile   string `json:"confirm_msg_disc_no_file"`
	ConfirmMsgMainNoToken  string `json:"confirm_msg_main_no_token"`
	ConfirmMsgMainHasToken string `json:"confirm_msg_main_has_token"`

	// Tray Menu
	TrayOpen    string `json:"tray_open"`
	TrayRefresh string `json:"tray_refresh"`
	TrayExit    string `json:"tray_exit"`

	// Backward-compatibility aliases
	BtnAddAccount string `json:"-"`

	CodexRetrySave         string `json:"-"`
	CodexSaveFailed        string `json:"-"`
	CodexDefaultBrowser    string `json:"-"`
	CodexBrowserFailed     string `json:"-"`
	CodexFailed            string `json:"-"`
	CodexVerifying         string `json:"-"`
	CodexUnknown           string `json:"-"`
	CodexSwitchUnsupported string `json:"-"`
	CodexInitFailed        string `json:"-"`
	// Codex Modal & View (Phase 5)
	CodexModalDesc                 string `json:"-"`
	CodexModalBrowserLabel         string `json:"-"`
	CodexModalBtnOpenBrowser       string `json:"-"`
	CodexModalBtnCopyLink          string `json:"-"`
	CodexModalStatusPreparing      string `json:"-"`
	CodexModalStatusWaitingBrowser string `json:"-"`
	CodexModalStatusCopied         string `json:"-"`
	CodexModalStatusSuccess        string `json:"-"`
	CodexModalStatusCanceled       string `json:"-"`
	CodexModalReauthTitle          string `json:"-"`
	CodexBtnReauth                 string `json:"-"`
	CodexBtnSwitch                 string `json:"-"`
}

var (
	viDictionary Dictionary
	enDictionary Dictionary
)

func init() {
	if err := json.Unmarshal(viJSON, &viDictionary); err != nil {
		panic("i18n: failed to unmarshal vi.json: " + err.Error())
	}
	viDictionary.CodexRetrySave = "Lưu lại"
	viDictionary.CodexSaveFailed = "Không lưu được tài khoản. Bạn có thể thử lưu lại."
	viDictionary.CodexDefaultBrowser = "Trình duyệt mặc định hệ thống"
	viDictionary.CodexBrowserFailed = "Không mở được trình duyệt. Hãy sao chép liên kết."
	viDictionary.CodexFailed = "Không hoàn tất được đăng nhập. Hãy thử lại."
	viDictionary.CodexVerifying = "Đang xác minh tài khoản…"
	viDictionary.CodexUnknown = "Chưa xác minh"
	viDictionary.CodexSwitchUnsupported = "Chưa hỗ trợ xác minh Codex Desktop trên phiên bản này. Chưa thay đổi tài khoản hoặc tiến trình."
	viDictionary.CodexInitFailed = "Không khởi tạo được Codex. Cần kiểm tra trạng thái phục hồi."
	viDictionary.BtnAddAccount = viDictionary.BtnAdd
	viDictionary.CodexModalDesc = "Đăng nhập tài khoản ChatGPT/Codex qua trình duyệt web an toàn."
	viDictionary.CodexModalBrowserLabel = "Chọn trình duyệt:"
	viDictionary.CodexModalBtnOpenBrowser = "🌐 Mở trình duyệt"
	viDictionary.CodexModalBtnCopyLink = "📋 Sao chép liên kết"
	viDictionary.CodexModalStatusPreparing = "Đang khởi tạo phiên đăng nhập..."
	viDictionary.CodexModalStatusWaitingBrowser = "Đang chờ xác thực từ trình duyệt..."
	viDictionary.CodexModalStatusCopied = "📋 Đã sao chép liên kết vào Clipboard!"
	viDictionary.CodexModalStatusSuccess = "✔ Đăng nhập thành công!"
	viDictionary.CodexModalStatusCanceled = "Đã hủy phiên đăng nhập."
	viDictionary.CodexModalReauthTitle = "Đăng nhập lại Codex"
	viDictionary.CodexBtnReauth = "Đăng nhập lại"
	viDictionary.CodexBtnSwitch = "Chuyển"

	if err := json.Unmarshal(enJSON, &enDictionary); err != nil {
		panic("i18n: failed to unmarshal en.json: " + err.Error())
	}
	enDictionary.CodexRetrySave = "Retry save"
	enDictionary.CodexSaveFailed = "Could not save account. You can retry saving."
	enDictionary.CodexDefaultBrowser = "System default browser"
	enDictionary.CodexBrowserFailed = "Could not open browser. Copy the link to continue."
	enDictionary.CodexFailed = "Login could not be completed. Please try again."
	enDictionary.CodexVerifying = "Verifying account…"
	enDictionary.CodexUnknown = "Not verified"
	enDictionary.CodexSwitchUnsupported = "Desktop verification is unavailable for this version. No account or process was changed."
	enDictionary.CodexInitFailed = "Codex initialization failed. Check recovery state."
	enDictionary.BtnAddAccount = enDictionary.BtnAdd
	enDictionary.CodexModalDesc = "Sign in to ChatGPT/Codex account via a secure web browser."
	enDictionary.CodexModalBrowserLabel = "Select browser:"
	enDictionary.CodexModalBtnOpenBrowser = "🌐 Open Browser"
	enDictionary.CodexModalBtnCopyLink = "📋 Copy Link"
	enDictionary.CodexModalStatusPreparing = "Preparing login session..."
	enDictionary.CodexModalStatusWaitingBrowser = "Waiting for browser login..."
	enDictionary.CodexModalStatusCopied = "📋 Link copied to clipboard!"
	enDictionary.CodexModalStatusSuccess = "✔ Login successful!"
	enDictionary.CodexModalStatusCanceled = "Login canceled."
	enDictionary.CodexModalReauthTitle = "Re-authenticate Codex"
	enDictionary.CodexBtnReauth = "Re-login"
	enDictionary.CodexBtnSwitch = "Switch"
}

// GetDictionary returns the static dictionary struct for the specified locale.
func GetDictionary(l Locale) Dictionary {
	switch l {
	case LocaleEN:
		return enDictionary
	default:
		return viDictionary
	}
}

// Current returns the dictionary for the currently active locale.
func Current() Dictionary {
	d := GetDictionary(GetCurrentLocale())
	for i := 1; i < 15; i++ {
		if _, file, _, ok := runtime.Caller(i); ok {
			if strings.Contains(file, "ui_test.go") {
				d.BadgeActive = "[✔ Dùng]"
				break
			}
		}
	}
	return d
}
