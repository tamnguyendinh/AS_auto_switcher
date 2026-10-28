//go:build ignore

// Read-only F4 probe. Run explicitly from repository root with go run.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"antigravity-switch/internal/engines/antigravity/telemetry"
	"antigravity-switch/internal/ui"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
)

func main() {
	tmp, err := os.MkdirTemp(".tmp", "supervisor-r2-legacy-")
	if err != nil {
		panic(err)
	}
	tmp, err = filepath.Abs(tmp)
	if err != nil {
		panic(err)
	}
	root, err := filepath.Abs(".tmp")
	if err != nil {
		panic(err)
	}
	if filepath.Dir(tmp) != root {
		panic("cleanup outside .tmp")
	}
	defer os.RemoveAll(tmp)
	fyneApp := test.NewApp()
	defer fyneApp.Quit()
	a := ui.NewAppWithFyne(fyneApp, nil)
	a.SetPreferencesPath(filepath.Join(tmp, "prefs.json"))
	a.SetQuotaCache(telemetry.NewQuotaCache(filepath.Join(tmp, "cache.json")))
	a.Init()
	defer a.Close()
	attached := map[uintptr]bool{}
	var walk func(fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		v := reflect.ValueOf(o)
		if v.Kind() == reflect.Pointer {
			attached[v.Pointer()] = true
		}
		if c, ok := o.(*fyne.Container); ok {
			for _, child := range c.Objects {
				walk(child)
			}
		}
		if s, ok := o.(*container.Scroll); ok {
			walk(s.Content)
		}
	}
	walk(a.Window().Content())
	var records []map[string]any
	for _, name := range []string{"headerBg", "statusBarLabel", "activeStatusLabel", "activeQuotaLabel"} {
		v := reflect.ValueOf(a).Elem().FieldByName(name)
		if !v.IsValid() {
			panic(fmt.Sprintf("field %s not found", name))
		}
		records = append(records, map[string]any{"field": name, "initialized": !v.IsNil(), "attached_to_render_tree": attached[v.Pointer()]})
	}
	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "  ")
	if err := e.Encode(map[string]any{"driver": "Fyne test.NewApp", "legacy_objects": records}); err != nil {
		panic(err)
	}
}
