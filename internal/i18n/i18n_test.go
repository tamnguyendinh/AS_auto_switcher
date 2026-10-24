package i18n

import (
	"reflect"
	"sync"
	"testing"
)

func TestI18n_DictionaryCompleteness(t *testing.T) {
	locales := []Locale{LocaleVI, LocaleEN}

	for _, loc := range locales {
		dict := GetDictionary(loc)
		v := reflect.ValueOf(dict)
		typ := v.Type()

		for i := 0; i < v.NumField(); i++ {
			fieldName := typ.Field(i).Name
			val := v.Field(i).String()

			if val == "" {
				t.Errorf("Locale [%s]: field '%s' must not be empty", loc, fieldName)
			}
		}
	}
}

func TestI18n_LocaleSwitching(t *testing.T) {
	// Restore default at test end
	defer SetCurrentLocale(LocaleVI)

	SetCurrentLocale(LocaleEN)
	if GetCurrentLocale() != LocaleEN {
		t.Fatalf("expected current locale to be 'en', got '%s'", GetCurrentLocale())
	}
	dEN := Current()
	if dEN.BtnAddAccount != "Add" {
		t.Errorf("expected 'Add', got '%s'", dEN.BtnAddAccount)
	}

	SetCurrentLocale(LocaleVI)
	if GetCurrentLocale() != LocaleVI {
		t.Fatalf("expected current locale to be 'vi', got '%s'", GetCurrentLocale())
	}
	dVI := Current()
	if dVI.BtnAddAccount != "Thêm" {
		t.Errorf("expected 'Thêm', got '%s'", dVI.BtnAddAccount)
	}

	// Fallback test
	SetCurrentLocale(Locale("unknown"))
	if GetCurrentLocale() != LocaleVI {
		t.Fatalf("expected fallback to 'vi', got '%s'", GetCurrentLocale())
	}
}

func TestI18n_ThreadSafety(t *testing.T) {
	defer SetCurrentLocale(LocaleVI)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			if idx%2 == 0 {
				SetCurrentLocale(LocaleEN)
			} else {
				SetCurrentLocale(LocaleVI)
			}
		}(i)

		go func() {
			defer wg.Done()
			d := Current()
			if d.AppTitle == "" {
				t.Errorf("AppTitle should never be empty during concurrent read")
			}
		}()
	}
	wg.Wait()
}
