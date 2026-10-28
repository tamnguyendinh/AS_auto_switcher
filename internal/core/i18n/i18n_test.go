package i18n

import (
	"encoding/json"
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

func TestI18n_JSONKeyParity(t *testing.T) {
	var viMap map[string]string
	if err := json.Unmarshal(viJSON, &viMap); err != nil {
		t.Fatalf("failed to unmarshal viJSON: %v", err)
	}

	var enMap map[string]string
	if err := json.Unmarshal(enJSON, &enMap); err != nil {
		t.Fatalf("failed to unmarshal enJSON: %v", err)
	}

	if len(viMap) != len(enMap) {
		t.Errorf("key count mismatch: vi has %d keys, en has %d keys", len(viMap), len(enMap))
	}

	for k, v := range viMap {
		if v == "" {
			t.Errorf("vi.json key %q has empty value", k)
		}
		enVal, ok := enMap[k]
		if !ok {
			t.Errorf("key %q present in vi.json but missing from en.json", k)
		} else if enVal == "" {
			t.Errorf("en.json key %q has empty value", k)
		}
	}

	for k := range enMap {
		if _, ok := viMap[k]; !ok {
			t.Errorf("key %q present in en.json but missing from vi.json", k)
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
	if dEN.BtnAdd != "Add" || dEN.BtnAddAccount != "Add" {
		t.Errorf("expected 'Add', got BtnAdd='%s', BtnAddAccount='%s'", dEN.BtnAdd, dEN.BtnAddAccount)
	}

	SetCurrentLocale(LocaleVI)
	if GetCurrentLocale() != LocaleVI {
		t.Fatalf("expected current locale to be 'vi', got '%s'", GetCurrentLocale())
	}
	dVI := Current()
	if dVI.BtnAdd != "Thêm" || dVI.BtnAddAccount != "Thêm" {
		t.Errorf("expected 'Thêm', got BtnAdd='%s', BtnAddAccount='%s'", dVI.BtnAdd, dVI.BtnAddAccount)
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
