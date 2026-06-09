package engine

import (
	"testing"
	"time"
)

func TestBrowserEngine_NewMethodsReturnErrorWhenNotLaunched(t *testing.T) {
	be := NewBrowserEngine("/tmp/test-screenshots")

	tests := []struct {
		name     string
		fn       func() error
		wantErr  bool
	}{
		{"TabList", func() error { be.TabList(); return nil }, false},
		{"Broadcast", func() error { _, err := be.Broadcast("1+1"); return err }, false},
		{"FillField", func() error { return be.FillField("#input", "val") }, true},
		{"SelectOption", func() error { return be.SelectOption("#sel", "opt") }, true},
		{"UploadFile", func() error { return be.UploadFile("#file", "/tmp/f") }, true},
		{"Submit", func() error { return be.Submit("#form") }, true},
		{"PrintToPDF", func() error { _, err := be.PrintToPDF("test"); return err }, true},
		{"ScreenshotFullPage", func() error { _, err := be.ScreenshotFullPage("test"); return err }, true},
		{"ScreenshotElement", func() error { _, err := be.ScreenshotElement("test", "#el"); return err }, true},
		{"SwitchToFrame", func() error { return be.SwitchToFrame("#iframe") }, true},
		{"SwitchToMain", func() error { return be.SwitchToMain() }, true},
		{"DownloadFile", func() error { _, err := be.DownloadFile("http://example.com/f"); return err }, true},
		{"WaitLoadStable", func() error { return be.WaitLoadStable(1) }, true},
		{"NetworkBlock", func() error { return be.NetworkBlock("*.css") }, false},
		{"NetworkMock", func() error { return be.NetworkMock("*.js", 200, "ok") }, false},
		{"SetLocalStorage", func() error { return be.SetLocalStorage(map[string]string{"k": "v"}) }, true},
		{"SetSessionStorage", func() error { return be.SetSessionStorage(map[string]string{"k": "v"}) }, true},
		{"ExportSessionData", func() error { _, err := be.ExportSessionData(); return err }, true},
		{"ImportSessionData", func() error { return be.ImportSessionData(map[string]interface{}{}) }, false},
		{"GetResourceTiming", func() error { _, err := be.GetResourceTiming(); return err }, true},
		{"GetPerformanceMetrics", func() error { _, err := be.GetPerformanceMetrics(); return err }, true},
		{"WaitForSelector", func() error { return be.WaitForSelector("#el", 1*time.Second) }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if tt.wantErr && err == nil {
				t.Errorf("expected error when browser not launched, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestTabGroup(t *testing.T) {
	tg := &TabGroup{}
	if tg.Count() != 0 {
		t.Errorf("expected 0 tabs, got %d", tg.Count())
	}
	if tg.Current() != nil {
		t.Errorf("expected nil current, got %v", tg.Current())
	}
	_, err := tg.Switch(0)
	if err == nil {
		t.Errorf("expected error switching to empty tab")
	}
	err = tg.Close(0)
	if err == nil {
		t.Errorf("expected error closing empty tab")
	}
}

func TestNetworkRule(t *testing.T) {
	be := NewBrowserEngine("/tmp/t")
	if err := be.NetworkBlock("*.css"); err != nil {
		t.Errorf("NetworkBlock: %v", err)
	}
	if err := be.NetworkMock("/api/*", 200, `{"ok":true}`); err != nil {
		t.Errorf("NetworkMock: %v", err)
	}
	if len(be.networkRules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(be.networkRules))
	}
}

func TestSetNetworkRules(t *testing.T) {
	be := NewBrowserEngine("/tmp/t")
	rules := []NetworkRule{
		{URLPattern: "*.png", Action: "block"},
		{URLPattern: "/api/*", Action: "mock", StatusCode: 200, Body: "ok"},
	}
	be.SetNetworkRules(rules)
	if len(be.networkRules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(be.networkRules))
	}
}

func TestSessionExportImport(t *testing.T) {
	s := &BrowserSession{Page: nil}
	_, err := s.ExportSessionData()
	if err == nil {
		t.Errorf("expected error with nil page")
	}
	err = s.ImportSessionData(map[string]interface{}{})
	if err != nil {
		t.Errorf("import with nil page: %v", err)
	}
}

func TestSessionMethodsReturnErrorWhenNoPage(t *testing.T) {
	s := &BrowserSession{}

	tests := []struct {
		name string
		fn   func() error
	}{
		{"ScreenshotFullPage", func() error { _, err := s.ScreenshotFullPage(""); return err }},
		{"ScreenshotElement", func() error { _, err := s.ScreenshotElement("#el"); return err }},
		{"FillField", func() error { return s.FillField("#i", "v") }},
		{"SelectOption", func() error { return s.SelectOption("#s", "o") }},
		{"UploadFile", func() error { return s.UploadFile("#f", "/p") }},
		{"Submit", func() error { return s.Submit("#f") }},
		{"PrintToPDF", func() error { _, err := s.PrintToPDF(); return err }},
		{"SetLocalStorage", func() error { return s.SetLocalStorage(nil) }},
		{"SetSessionStorage", func() error { return s.SetSessionStorage(nil) }},
		{"GetResourceTiming", func() error { _, err := s.GetResourceTiming(); return err }},
		{"GetPerformanceMetrics", func() error { _, err := s.GetPerformanceMetrics(); return err }},
		{"WaitLoadStable", func() error { return s.WaitLoadStable(1) }},
		{"DownloadFile", func() error { _, err := s.DownloadFile("http://x.com", "/tmp"); return err }},
		{"GetLocalStorage", func() error { _, err := s.GetLocalStorage(); return err }},
		{"GetSessionStorage", func() error { _, err := s.GetSessionStorage(); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Errorf("expected error with nil page, got nil")
			}
		})
	}
}
