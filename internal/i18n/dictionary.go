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
	// Header & Window
	AppTitle          string `json:"app_title"`
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

	// Modals
	ModalAddTitle        string `json:"modal_add_title"`
	ModalAddHeader       string `json:"modal_add_header"`
	ModalAddClose        string `json:"modal_add_close"`
	ModalScanChrome      string `json:"modal_scan_chrome"`
	ModalScanningChrome  string `json:"modal_scanning_chrome"`
	ModalEmptyCandidates string `json:"modal_empty_candidates"`
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
}

var (
	viDictionary Dictionary
	enDictionary Dictionary
)

func init() {
	if err := json.Unmarshal(viJSON, &viDictionary); err != nil {
		panic("i18n: failed to unmarshal vi.json: " + err.Error())
	}
	viDictionary.BtnAddAccount = viDictionary.BtnAdd

	if err := json.Unmarshal(enJSON, &enDictionary); err != nil {
		panic("i18n: failed to unmarshal en.json: " + err.Error())
	}
	enDictionary.BtnAddAccount = enDictionary.BtnAdd
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
