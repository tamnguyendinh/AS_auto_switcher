package i18n

import "sync"

// Locale defines the supported language identifier.
type Locale string

const (
	LocaleVI Locale = "vi"
	LocaleEN Locale = "en"
)

var (
	currentLocaleMu sync.RWMutex
	currentLocale   Locale = LocaleVI
)

// SetCurrentLocale sets the active locale thread-safely.
func SetCurrentLocale(l Locale) {
	currentLocaleMu.Lock()
	defer currentLocaleMu.Unlock()
	if l != LocaleEN && l != LocaleVI {
		l = LocaleVI
	}
	currentLocale = l
}

// GetCurrentLocale returns the active locale thread-safely.
func GetCurrentLocale() Locale {
	currentLocaleMu.RLock()
	defer currentLocaleMu.RUnlock()
	return currentLocale
}
