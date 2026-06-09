package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type BrowserSession struct {
	ID        string
	Browser   *rod.Browser
	Page      *rod.Page
	CreatedAt time.Time
	LastUsed  time.Time
	InUse     bool
}

type BrowserSessionManager struct {
	mu       sync.Mutex
	sessions map[string]*BrowserSession
	maxSize  int
	timeout  time.Duration
}

func NewBrowserSessionManager(maxSize int, timeout time.Duration) *BrowserSessionManager {
	return &BrowserSessionManager{
		sessions: make(map[string]*BrowserSession),
		maxSize:  maxSize,
		timeout:  timeout,
	}
}

func (bsm *BrowserSessionManager) Acquire() (*BrowserSession, error) {
	bsm.mu.Lock()
	defer bsm.mu.Unlock()

	for id, s := range bsm.sessions {
		if !s.InUse && time.Since(s.LastUsed) < bsm.timeout {
			s.InUse = true
			s.LastUsed = time.Now()
			page, err := s.Browser.Page(proto.TargetCreateTarget{})
			if err == nil {
				s.Page = page
			}
			return s, nil
		}
		_ = id
	}

	if len(bsm.sessions) >= bsm.maxSize {
		oldest := bsm.evictOldest()
		if oldest != nil {
			oldest.Browser.Close()
			delete(bsm.sessions, oldest.ID)
		}
	}

	path, found := launcher.LookPath()
	var browser *rod.Browser
	if !found {
		l := launcher.New()
		u, err := l.Launch()
		if err != nil {
			return nil, fmt.Errorf("launch browser: %w", err)
		}
		browser = rod.New().ControlURL(u)
	} else {
		u := launcher.New().Bin(path).MustLaunch()
		browser = rod.New().ControlURL(u)
	}

	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("connect browser: %w", err)
	}

	page, err := browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		browser.Close()
		return nil, fmt.Errorf("create page: %w", err)
	}

	id := fmt.Sprintf("session-%d", time.Now().UnixNano())
	session := &BrowserSession{
		ID:        id,
		Browser:   browser,
		Page:      page,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		InUse:     true,
	}
	bsm.sessions[id] = session
	return session, nil
}

func (bsm *BrowserSessionManager) Release(id string) {
	bsm.mu.Lock()
	defer bsm.mu.Unlock()
	if s, ok := bsm.sessions[id]; ok {
		s.InUse = false
		s.LastUsed = time.Now()
	}
}

func (bsm *BrowserSessionManager) CloseAll() {
	bsm.mu.Lock()
	defer bsm.mu.Unlock()
	for _, s := range bsm.sessions {
		s.Browser.Close()
	}
	bsm.sessions = make(map[string]*BrowserSession)
}

func (bsm *BrowserSessionManager) Stats() (active, idle, total int) {
	bsm.mu.Lock()
	defer bsm.mu.Unlock()
	for _, s := range bsm.sessions {
		total++
		if s.InUse {
			active++
		} else {
			idle++
		}
	}
	return
}

func (bsm *BrowserSessionManager) evictOldest() *BrowserSession {
	var oldest *BrowserSession
	for _, s := range bsm.sessions {
		if oldest == nil || s.LastUsed.Before(oldest.LastUsed) {
			oldest = s
		}
	}
	return oldest
}

func (s *BrowserSession) Navigate(url string) error {
	if s.Page == nil {
		return fmt.Errorf("no page in session")
	}
	if err := s.Page.Navigate(url); err != nil {
		return err
	}
	return s.Page.WaitLoad()
}

func (s *BrowserSession) Screenshot(name string) ([]byte, error) {
	if s.Page == nil {
		return nil, fmt.Errorf("no page in session")
	}
	return s.Page.Screenshot(false, &proto.PageCaptureScreenshot{
		Format: proto.PageCaptureScreenshotFormatPng,
	})
}

func (s *BrowserSession) ScreenshotFullPage(name string) ([]byte, error) {
	if s.Page == nil {
		return nil, fmt.Errorf("no page in session")
	}
	return s.Page.Screenshot(true, &proto.PageCaptureScreenshot{
		Format: proto.PageCaptureScreenshotFormatPng,
	})
}

func (s *BrowserSession) ScreenshotElement(selector string) ([]byte, error) {
	if s.Page == nil {
		return nil, fmt.Errorf("no page in session")
	}
	el, err := s.Page.Element(selector)
	if err != nil {
		return nil, fmt.Errorf("element %s: %w", selector, err)
	}
	return el.Screenshot(proto.PageCaptureScreenshotFormatPng, 0)
}

func (s *BrowserSession) HTML() (string, error) {
	if s.Page == nil {
		return "", fmt.Errorf("no page in session")
	}
	return s.Page.HTML()
}

func (s *BrowserSession) Evaluate(js string) (interface{}, error) {
	if s.Page == nil {
		return nil, fmt.Errorf("no page in session")
	}
	return s.Page.Eval(js)
}

func (s *BrowserSession) Click(selector string) error {
	if s.Page == nil {
		return fmt.Errorf("no page in session")
	}
	el, err := s.Page.Element(selector)
	if err != nil {
		return err
	}
	return el.Click(proto.InputMouseButtonLeft, 1)
}

func (s *BrowserSession) Type(selector, text string) error {
	if s.Page == nil {
		return fmt.Errorf("no page in session")
	}
	el, err := s.Page.Element(selector)
	if err != nil {
		return err
	}
	return el.Input(text)
}

func (s *BrowserSession) FillField(selector, value string) error {
	if s.Page == nil {
		return fmt.Errorf("no page in session")
	}
	el, err := s.Page.Element(selector)
	if err != nil {
		return fmt.Errorf("field %s: %w", selector, err)
	}
	if err := el.SelectText(""); err == nil {
		_ = el.Input(value)
		return nil
	}
	return el.Input(value)
}

func (s *BrowserSession) SelectOption(selector, value string) error {
	if s.Page == nil {
		return fmt.Errorf("no page in session")
	}
	el, err := s.Page.Element(selector)
	if err != nil {
		return fmt.Errorf("select %s: %w", selector, err)
	}
	return el.Select([]string{value}, true, rod.SelectorTypeText)
}

func (s *BrowserSession) UploadFile(selector, filePath string) error {
	if s.Page == nil {
		return fmt.Errorf("no page in session")
	}
	el, err := s.Page.Element(selector)
	if err != nil {
		return fmt.Errorf("file input %s: %w", selector, err)
	}
	return el.SetFiles([]string{filePath})
}

func (s *BrowserSession) Submit(selector string) error {
	if s.Page == nil {
		return fmt.Errorf("no page in session")
	}
	el, err := s.Page.Element(selector)
	if err != nil {
		return fmt.Errorf("form %s: %w", selector, err)
	}
	_, err = el.Eval(`() => { const f = this; if (f.tagName === 'FORM') { f.submit(); return true; } else { const form = f.closest('form'); if (form) { form.submit(); return true; } return false; } }`)
	if err != nil {
		return fmt.Errorf("submit failed: %w", err)
	}
	return nil
}

func (s *BrowserSession) PrintToPDF() ([]byte, error) {
	if s.Page == nil {
		return nil, fmt.Errorf("no page in session")
	}
	stream, err := s.Page.PDF(&proto.PagePrintToPDF{})
	if err != nil {
		return nil, fmt.Errorf("pdf: %w", err)
	}
	return io.ReadAll(stream)
}

func (s *BrowserSession) GetCookies() ([]*proto.NetworkCookie, error) {
	if s.Page == nil {
		return nil, fmt.Errorf("no page in session")
	}
	return s.Page.Cookies(nil)
}

func (s *BrowserSession) SetCookie(name, value, domain, path string) error {
	if s.Page == nil {
		return fmt.Errorf("no page in session")
	}
	return s.Page.SetCookies([]*proto.NetworkCookieParam{{
		Name: name, Value: value, Domain: domain, Path: path,
	}})
}

func (s *BrowserSession) ClearCookies() error {
	if s.Page == nil {
		return fmt.Errorf("no page in session")
	}
	return s.Page.SetCookies([]*proto.NetworkCookieParam{})
}

func (s *BrowserSession) ConsoleLogs() ([]string, error) {
	if s.Page == nil {
		return nil, fmt.Errorf("no page in session")
	}
	var logs []string
	s.Page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) {
		for _, arg := range e.Args {
			logs = append(logs, arg.Value.String())
		}
	})()
	return logs, nil
}

func (s *BrowserSession) WaitForSelector(selector string, timeout time.Duration) error {
	if s.Page == nil {
		return fmt.Errorf("no page in session")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, err := s.Page.Context(ctx).Element(selector)
	return err
}

func (s *BrowserSession) GetLocalStorage() (map[string]string, error) {
	if s.Page == nil {
		return nil, fmt.Errorf("no page in session")
	}
	val, err := s.Page.Eval(`() => JSON.stringify(localStorage)`)
	if err != nil {
		return nil, fmt.Errorf("get local storage: %w", err)
	}
	result := make(map[string]string)
	json.Unmarshal([]byte(val.Value.String()), &result)
	return result, nil
}

func (s *BrowserSession) GetSessionStorage() (map[string]string, error) {
	if s.Page == nil {
		return nil, fmt.Errorf("no page in session")
	}
	val, err := s.Page.Eval(`() => JSON.stringify(sessionStorage)`)
	if err != nil {
		return nil, fmt.Errorf("get session storage: %w", err)
	}
	result := make(map[string]string)
	json.Unmarshal([]byte(val.Value.String()), &result)
	return result, nil
}

func (s *BrowserSession) SetLocalStorage(data map[string]string) error {
	if s.Page == nil {
		return fmt.Errorf("no page in session")
	}
	js := "(() => { localStorage.clear();"
	for k, v := range data {
		js += fmt.Sprintf(" localStorage.setItem(%q,%q);", k, v)
	}
	js += " return true; })()"
	_, err := s.Page.Eval(js)
	return err
}

func (s *BrowserSession) SetSessionStorage(data map[string]string) error {
	if s.Page == nil {
		return fmt.Errorf("no page in session")
	}
	js := "(() => { sessionStorage.clear();"
	for k, v := range data {
		js += fmt.Sprintf(" sessionStorage.setItem(%q,%q);", k, v)
	}
	js += " return true; })()"
	_, err := s.Page.Eval(js)
	return err
}

func (s *BrowserSession) ExportSessionData() (map[string]interface{}, error) {
	cookies, err := s.GetCookies()
	if err != nil {
		return nil, err
	}
	local, err := s.GetLocalStorage()
	if err != nil {
		return nil, err
	}
	session, err := s.GetSessionStorage()
	if err != nil {
		return nil, err
	}
	cookieMarshaled := make([]map[string]interface{}, 0, len(cookies))
	for _, c := range cookies {
		cookieMarshaled = append(cookieMarshaled, map[string]interface{}{
			"name": c.Name, "value": c.Value, "domain": c.Domain, "path": c.Path,
		})
	}
	return map[string]interface{}{
		"cookies":        cookieMarshaled,
		"localStorage":   local,
		"sessionStorage": session,
	}, nil
}

func (s *BrowserSession) ImportSessionData(data map[string]interface{}) error {
	if cookies, ok := data["cookies"].([]interface{}); ok {
		for _, c := range cookies {
			if cm, ok := c.(map[string]interface{}); ok {
				_ = s.SetCookie(
					toString(cm["name"]),
					toString(cm["value"]),
					toString(cm["domain"]),
					toString(cm["path"]),
				)
			}
		}
	}
	if local, ok := data["localStorage"].(map[string]interface{}); ok {
		strMap := make(map[string]string)
		for k, v := range local {
			strMap[k] = toString(v)
		}
		_ = s.SetLocalStorage(strMap)
	}
	return nil
}

func (s *BrowserSession) GetResourceTiming() ([]map[string]interface{}, error) {
	if s.Page == nil {
		return nil, fmt.Errorf("no page in session")
	}
	val, err := s.Page.Eval(`() => JSON.stringify(performance.getEntriesByType('resource'))`)
	if err != nil {
		return nil, fmt.Errorf("resource timing: %w", err)
	}
	var entries []map[string]interface{}
	json.Unmarshal([]byte(val.Value.String()), &entries)
	return entries, nil
}

func (s *BrowserSession) GetPerformanceMetrics() (map[string]interface{}, error) {
	if s.Page == nil {
		return nil, fmt.Errorf("no page in session")
	}
	val, err := s.Page.Eval(`() => {
		const n = performance.getEntriesByType('navigation')[0];
		return JSON.stringify({
			domContentLoaded: n ? n.domContentLoadedEventEnd - n.domContentLoadedEventStart : 0,
			loadTime: n ? n.loadEventEnd - n.loadEventStart : 0,
			domInteractive: n ? n.domInteractive : 0,
			totalTime: n ? n.loadEventEnd : 0,
			fetchTime: n ? n.responseEnd - n.fetchStart : 0,
			redirectCount: n ? n.redirectCount : 0,
			transferSize: n ? n.transferSize : 0,
			encodedBodySize: n ? n.encodedBodySize : 0,
			decodedBodySize: n ? n.decodedBodySize : 0,
		});
	}`)
	if err != nil {
		return nil, fmt.Errorf("performance metrics: %w", err)
	}
	var metrics map[string]interface{}
	json.Unmarshal([]byte(val.Value.String()), &metrics)
	return metrics, nil
}

func (s *BrowserSession) DownloadFile(url string, downloadDir string) (string, error) {
	if s.Page == nil || s.Browser == nil {
		return "", fmt.Errorf("no page or browser in session")
	}
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return "", fmt.Errorf("create download dir: %w", err)
	}

	wait := s.Browser.WaitDownload(downloadDir)
	if err := s.Page.Navigate(url); err != nil {
		return "", fmt.Errorf("navigate for download: %w", err)
	}
	dl := wait()
	base := filepath.Base(dl.URL)
	if base == "" || base == "." {
		base = fmt.Sprintf("download_%d", time.Now().UnixNano())
	}
	filename := filepath.Join(downloadDir, base)
	return filename, nil
}

func (s *BrowserSession) WaitLoadStable(timeoutSec int) error {
	if s.Page == nil {
		return fmt.Errorf("no page in session")
	}
	return s.Page.WaitStable(1 * time.Second)
}
