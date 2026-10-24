package i18n

// Dictionary defines the strongly-typed compile-time checked catalog of all localized UI strings.
type Dictionary struct {
	// Header & Window
	AppTitle          string
	LangDropdownLabel string

	// Toolbar Buttons & Controls
	BtnAddAccount   string
	BtnRefreshList  string
	BtnRefreshQuota string
	BtnAlwaysOnTop  string
	LblMaskAccounts string
	LblAutoSwitch   string

	// Account Card
	BadgeActive        string
	BadgeSwitch        string
	BadgeNeedOAuth     string
	BtnDeleteAccount   string
	DeleteConfirmTitle string
	DeleteConfirmMsg   string

	// Server & Connection Status
	StatusServerActive  string
	StatusServerStopped string
	StatusConnecting    string
	StatusNoCandidate   string
	StatusAutoSwitchOn  string
	StatusAutoSwitchOff string

	// Modals & Dialogs
	ModalAddTitle       string
	ModalClose          string
	ModalEmailLabel     string
	ModalSave           string
	ModalCancel         string
	ModalScanChrome     string
	ModalScanningChrome string
	ModalEmptyCandidate string

	// Tray Menu
	TrayOpen    string
	TrayRefresh string
	TrayExit    string
}

var viDictionary = Dictionary{
	AppTitle:            "Antigravity Multi-Account Switcher",
	LangDropdownLabel:   "VN",
	BtnAddAccount:       "Thêm",
	BtnRefreshList:      "Làm mới danh sách",
	BtnRefreshQuota:     "Làm mới Quota",
	BtnAlwaysOnTop:      "📌",
	LblMaskAccounts:     "Ẩn tài khoản:",
	LblAutoSwitch:       "Tự động:",
	BadgeActive:         "[✔ Dùng]",
	BadgeSwitch:         "[Chuyển]",
	BadgeNeedOAuth:      "[Cần OAuth]",
	BtnDeleteAccount:    "Xóa",
	DeleteConfirmTitle:  "Xác nhận xóa tài khoản",
	DeleteConfirmMsg:    "Bạn có chắc chắn muốn xóa tài khoản khỏi danh bạ đã quét không?",
	StatusServerActive:  "Antigravity: Hoạt động",
	StatusServerStopped: "Antigravity: Không hoạt động  | Dự phòng WinCred",
	StatusConnecting:    "Đang kết nối...",
	StatusNoCandidate:   "⚠️ Không có tài khoản để chuyển.",
	StatusAutoSwitchOn:  "Tự động chuyển tài khoản: BẬT",
	StatusAutoSwitchOff: "Tự động chuyển tài khoản: TẮT",
	ModalAddTitle:       "Thêm tài khoản",
	ModalClose:          "Đóng",
	ModalEmailLabel:     "Email tài khoản",
	ModalSave:           "Lưu",
	ModalCancel:         "Hủy",
	ModalScanChrome:     "Quét Chrome",
	ModalScanningChrome: "⏳ Đang quét...",
	ModalEmptyCandidate: "Không có tài khoản nào trong danh bạ.",
	TrayOpen:            "Hiển thị Switcher",
	TrayRefresh:         "Làm mới Chrome",
	TrayExit:            "Thoát",
}

var enDictionary = Dictionary{
	AppTitle:            "Antigravity Multi-Account Switcher",
	LangDropdownLabel:   "EN",
	BtnAddAccount:       "Add",
	BtnRefreshList:      "Refresh List",
	BtnRefreshQuota:     "Refresh Quota",
	BtnAlwaysOnTop:      "📌",
	LblMaskAccounts:     "Mask Accounts:",
	LblAutoSwitch:       "Auto Switch:",
	BadgeActive:         "[✔ Active]",
	BadgeSwitch:         "[Switch]",
	BadgeNeedOAuth:      "[Need OAuth]",
	BtnDeleteAccount:    "Delete",
	DeleteConfirmTitle:  "Confirm Account Deletion",
	DeleteConfirmMsg:    "Are you sure you want to remove this account from the discovered catalog?",
	StatusServerActive:  "Antigravity: Active",
	StatusServerStopped: "Antigravity: Inactive | WinCred Fallback",
	StatusConnecting:    "Connecting...",
	StatusNoCandidate:   "⚠️ No accounts available to switch.",
	StatusAutoSwitchOn:  "Auto-switch accounts: ON",
	StatusAutoSwitchOff: "Auto-switch accounts: OFF",
	ModalAddTitle:       "Add Account",
	ModalClose:          "Close",
	ModalEmailLabel:     "Account Email",
	ModalSave:           "Save",
	ModalCancel:         "Cancel",
	ModalScanChrome:     "Scan Chrome",
	ModalScanningChrome: "⏳ Scanning...",
	ModalEmptyCandidate: "No accounts found in catalog.",
	TrayOpen:            "Open Switcher",
	TrayRefresh:         "Refresh Chrome",
	TrayExit:            "Exit",
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
	return GetDictionary(GetCurrentLocale())
}
