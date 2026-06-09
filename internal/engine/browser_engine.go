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

type TabInfo struct {
	ID        int       `json:"id"`
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type TabGroup struct {
	mu       sync.Mutex
	tabs     []*rod.Page
	current  int
}

func (tg *TabGroup) Add(page *rod.Page) int {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	id := len(tg.tabs)
	tg.tabs = append(tg.tabs, page)
	return id
}

func (tg *TabGroup) Switch(index int) (*rod.Page, error) {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	if index < 0 || index >= len(tg.tabs) {
		return nil, fmt.Errorf("tab index %d out of range (0-%d)", index, len(tg.tabs)-1)
	}
	tg.current = index
	return tg.tabs[index], nil
}

func (tg *TabGroup) Close(index int) error {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	if index < 0 || index >= len(tg.tabs) {
		return fmt.Errorf("tab index %d out of range", index)
	}
	if err := tg.tabs[index].Close(); err != nil {
		return err
	}
	tg.tabs = append(tg.tabs[:index], tg.tabs[index+1:]...)
	if len(tg.tabs) == 0 {
		tg.current = 0
		return nil
	}
	if index < len(tg.tabs) {
		tg.current = index
	} else {
		tg.current = len(tg.tabs) - 1
	}
	return nil
}

func (tg *TabGroup) Current() *rod.Page {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	if len(tg.tabs) == 0 {
		return nil
	}
	return tg.tabs[tg.current]
}

func (tg *TabGroup) Count() int {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	return len(tg.tabs)
}

func (tg *TabGroup) List() []TabInfo {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	infos := make([]TabInfo, 0, len(tg.tabs))
	for i, p := range tg.tabs {
		info, err := p.Info()
		url := ""
		title := ""
		if err == nil {
			url = info.URL
			title = info.Title
		}
		infos = append(infos, TabInfo{ID: i, URL: url, Title: title})
	}
	return infos
}

func (tg *TabGroup) Broadcast(js string) ([]interface{}, error) {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	var results []interface{}
	for _, p := range tg.tabs {
		res, err := p.Eval(js)
		if err != nil {
			results = append(results, fmt.Sprintf("error: %v", err))
		} else {
			results = append(results, res.Value)
		}
	}
	return results, nil
}

type NetworkRule struct {
	URLPattern string `json:"url_pattern"`
	Action     string `json:"action"` // block, mock, throttle
	StatusCode int    `json:"status_code,omitempty"`
	Body       string `json:"body,omitempty"`
	Latency    time.Duration `json:"latency,omitempty"`
}

type BrowserEngine struct {
	browser       *rod.Browser
	tabGroup      *TabGroup
	screenshotDir string
	downloadDir   string
	networkRules  []NetworkRule
}

func NewBrowserEngine(screenshotDir string) *BrowserEngine {
	return &BrowserEngine{
		screenshotDir: screenshotDir,
		downloadDir:   filepath.Join(screenshotDir, "downloads"),
		tabGroup:      &TabGroup{},
	}
}

func (be *BrowserEngine) SetNetworkRules(rules []NetworkRule) {
	be.networkRules = rules
}

func (be *BrowserEngine) Launch() error {
	path, found := launcher.LookPath()
	var browser *rod.Browser
	if !found {
		l := launcher.New()
		u, err := l.Launch()
		if err != nil {
			return fmt.Errorf("failed to launch browser: %w", err)
		}
		browser = rod.New().ControlURL(u)
	} else {
		u := launcher.New().Bin(path).MustLaunch()
		browser = rod.New().ControlURL(u)
	}

	if err := browser.Connect(); err != nil {
		return fmt.Errorf("failed to connect to browser: %w", err)
	}
	be.browser = browser

	page, err := be.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}
	be.tabGroup.Add(page)

	return nil
}

func (be *BrowserEngine) currentPage() *rod.Page {
	return be.tabGroup.Current()
}

func (be *BrowserEngine) Navigate(url string) error {
	page := be.currentPage()
	if page == nil {
		return fmt.Errorf("browser not launched")
	}
	if err := page.Navigate(url); err != nil {
		return err
	}
	return page.WaitLoad()
}

func (be *BrowserEngine) WaitLoad() error {
	page := be.currentPage()
	if page == nil {
		return fmt.Errorf("browser not launched")
	}
	return page.WaitLoad()
}

func (be *BrowserEngine) Screenshot(name string) (string, error) {
	page := be.currentPage()
	if page == nil {
		return "", fmt.Errorf("browser not launched")
	}

	if err := os.MkdirAll(be.screenshotDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create screenshot dir: %w", err)
	}

	filename := filepath.Join(be.screenshotDir, fmt.Sprintf("%s_%d.png", name, time.Now().Unix()))
	data, err := page.Screenshot(false, &proto.PageCaptureScreenshot{
		Format: proto.PageCaptureScreenshotFormatPng,
	})
	if err != nil {
		return "", fmt.Errorf("failed to take screenshot: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return "", fmt.Errorf("failed to save screenshot: %w", err)
	}

	return filename, nil
}

func (be *BrowserEngine) Click(selector string) error {
	page := be.currentPage()
	if page == nil {
		return fmt.Errorf("browser not launched")
	}
	el, err := page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found %s: %w", selector, err)
	}
	return el.Click(proto.InputMouseButtonLeft, 1)
}

func (be *BrowserEngine) Type(selector, text string) error {
	page := be.currentPage()
	if page == nil {
		return fmt.Errorf("browser not launched")
	}
	el, err := page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found %s: %w", selector, err)
	}
	return el.Input(text)
}

func (be *BrowserEngine) HTML() (string, error) {
	page := be.currentPage()
	if page == nil {
		return "", fmt.Errorf("browser not launched")
	}
	return page.HTML()
}

func (be *BrowserEngine) Evaluate(js string) (interface{}, error) {
	page := be.currentPage()
	if page == nil {
		return nil, fmt.Errorf("browser not launched")
	}
	return page.Eval(js)
}

func (be *BrowserEngine) Close() {
	if be.browser != nil {
		be.browser.Close()
	}
}

func (be *BrowserEngine) Page() *rod.Page {
	return be.currentPage()
}

func (be *BrowserEngine) GetCookies() ([]*proto.NetworkCookie, error) {
	page := be.currentPage()
	if page == nil {
		return nil, fmt.Errorf("browser not launched")
	}
	cookies, err := page.Cookies(nil)
	if err != nil {
		return nil, fmt.Errorf("get cookies: %w", err)
	}
	return cookies, nil
}

func (be *BrowserEngine) SetCookie(name, value, domain, path string) error {
	page := be.currentPage()
	if page == nil {
		return fmt.Errorf("browser not launched")
	}
	cookie := &proto.NetworkCookieParam{
		Name:   name,
		Value:  value,
		Domain: domain,
		Path:   path,
	}
	return page.SetCookies([]*proto.NetworkCookieParam{cookie})
}

func (be *BrowserEngine) ClearCookies() error {
	page := be.currentPage()
	if page == nil {
		return fmt.Errorf("browser not launched")
	}
	return page.SetCookies([]*proto.NetworkCookieParam{})
}

func (be *BrowserEngine) ConsoleLogs() ([]string, error) {
	page := be.currentPage()
	if page == nil {
		return nil, fmt.Errorf("browser not launched")
	}
	var logs []string
	page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) {
		for _, arg := range e.Args {
			logs = append(logs, arg.Value.String())
		}
	})()
	return logs, nil
}

func (be *BrowserEngine) NewTab() (*rod.Page, error) {
	if be.browser == nil {
		return nil, fmt.Errorf("browser not launched")
	}
	page, err := be.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("new tab: %w", err)
	}
	be.tabGroup.Add(page)
	return page, nil
}

func (be *BrowserEngine) SwitchTab(index int) error {
	_, err := be.tabGroup.Switch(index)
	return err
}

func (be *BrowserEngine) CloseTab(index int) error {
	return be.tabGroup.Close(index)
}

func (be *BrowserEngine) TabCount() int {
	return be.tabGroup.Count()
}

func (be *BrowserEngine) TabList() []TabInfo {
	return be.tabGroup.List()
}

func (be *BrowserEngine) Broadcast(js string) ([]interface{}, error) {
	return be.tabGroup.Broadcast(js)
}

func (be *BrowserEngine) SetViewport(width, height int) error {
	page := be.currentPage()
	if page == nil {
		return fmt.Errorf("browser not launched")
	}
	return page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:             width,
		Height:            height,
		DeviceScaleFactor: 1,
	})
}

func (be *BrowserEngine) ScreenshotCompare(name string, baseline []byte) (bool, string, error) {
	filename, err := be.Screenshot(name)
	if err != nil {
		return false, "", err
	}
	if baseline == nil {
		return true, filename, nil
	}
	current, err := os.ReadFile(filename)
	if err != nil {
		return false, "", fmt.Errorf("read screenshot: %w", err)
	}
	if len(current) != len(baseline) {
		return false, filename, nil
	}
	for i := range current {
		if current[i] != baseline[i] {
			return false, filename, nil
		}
	}
	return true, filename, nil
}

func (be *BrowserEngine) GetLocalStorage() (map[string]string, error) {
	page := be.currentPage()
	if page == nil {
		return nil, fmt.Errorf("browser not launched")
	}
	val, err := page.Eval(`() => JSON.stringify(localStorage)`)
	if err != nil {
		return nil, fmt.Errorf("get local storage: %w", err)
	}
	result := make(map[string]string)
	json.Unmarshal([]byte(val.Value.String()), &result)
	return result, nil
}

func (be *BrowserEngine) GetSessionStorage() (map[string]string, error) {
	page := be.currentPage()
	if page == nil {
		return nil, fmt.Errorf("browser not launched")
	}
	val, err := page.Eval(`() => JSON.stringify(sessionStorage)`)
	if err != nil {
		return nil, fmt.Errorf("get session storage: %w", err)
	}
	result := make(map[string]string)
	json.Unmarshal([]byte(val.Value.String()), &result)
	return result, nil
}

func (be *BrowserEngine) SetLocalStorage(data map[string]string) error {
	page := be.currentPage()
	if page == nil {
		return fmt.Errorf("browser not launched")
	}
	js := "(() => { localStorage.clear();"
	for k, v := range data {
		js += fmt.Sprintf(" localStorage.setItem(%q,%q);", k, v)
	}
	js += " return true; })()"
	_, err := page.Eval(js)
	return err
}

func (be *BrowserEngine) SetSessionStorage(data map[string]string) error {
	page := be.currentPage()
	if page == nil {
		return fmt.Errorf("browser not launched")
	}
	js := "(() => { sessionStorage.clear();"
	for k, v := range data {
		js += fmt.Sprintf(" sessionStorage.setItem(%q,%q);", k, v)
	}
	js += " return true; })()"
	_, err := page.Eval(js)
	return err
}

func (be *BrowserEngine) ExportSessionData() (map[string]interface{}, error) {
	cookies, err := be.GetCookies()
	if err != nil {
		return nil, err
	}
	local, err := be.GetLocalStorage()
	if err != nil {
		return nil, err
	}
	session, err := be.GetSessionStorage()
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

func (be *BrowserEngine) ImportSessionData(data map[string]interface{}) error {
	if cookies, ok := data["cookies"].([]interface{}); ok {
		for _, c := range cookies {
			if cm, ok := c.(map[string]interface{}); ok {
				_ = be.SetCookie(
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
		_ = be.SetLocalStorage(strMap)
	}
	return nil
}

func (be *BrowserEngine) WaitForSelector(selector string, timeout time.Duration) error {
	page := be.currentPage()
	if page == nil {
		return fmt.Errorf("browser not launched")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, err := page.Context(ctx).Element(selector)
	return err
}

func (be *BrowserEngine) FillField(selector, value string) error {
	page := be.currentPage()
	if page == nil {
		return fmt.Errorf("browser not launched")
	}
	el, err := page.Element(selector)
	if err != nil {
		return fmt.Errorf("field not found %s: %w", selector, err)
	}
	if err := el.SelectText(""); err == nil {
		_ = el.Input(value)
		return nil
	}
	return el.Input(value)
}

func (be *BrowserEngine) SelectOption(selector, value string) error {
	page := be.currentPage()
	if page == nil {
		return fmt.Errorf("browser not launched")
	}
	el, err := page.Element(selector)
	if err != nil {
		return fmt.Errorf("select not found %s: %w", selector, err)
	}
	return el.Select([]string{value}, true, rod.SelectorTypeText)
}

func (be *BrowserEngine) UploadFile(selector, filePath string) error {
	page := be.currentPage()
	if page == nil {
		return fmt.Errorf("browser not launched")
	}
	el, err := page.Element(selector)
	if err != nil {
		return fmt.Errorf("file input not found %s: %w", selector, err)
	}
	return el.SetFiles([]string{filePath})
}

func (be *BrowserEngine) Submit(selector string) error {
	page := be.currentPage()
	if page == nil {
		return fmt.Errorf("browser not launched")
	}
	el, err := page.Element(selector)
	if err != nil {
		return fmt.Errorf("form not found %s: %w", selector, err)
	}
	_, err = el.Eval(`() => { const f = this; if (f.tagName === 'FORM') { f.submit(); return true; } else { const form = f.closest('form'); if (form) { form.submit(); return true; } return false; } }`)
	if err != nil {
		return fmt.Errorf("submit failed: %w", err)
	}
	return nil
}

func (be *BrowserEngine) PrintToPDF(name string) (string, error) {
	page := be.currentPage()
	if page == nil {
		return "", fmt.Errorf("browser not launched")
	}
	if err := os.MkdirAll(be.screenshotDir, 0755); err != nil {
		return "", fmt.Errorf("create dir: %w", err)
	}
	filename := filepath.Join(be.screenshotDir, fmt.Sprintf("%s_%d.pdf", name, time.Now().Unix()))
	pdfStream, err := page.PDF(&proto.PagePrintToPDF{})
	if err != nil {
		return "", fmt.Errorf("pdf: %w", err)
	}
	data, err := io.ReadAll(pdfStream)
	if err != nil {
		return "", fmt.Errorf("read pdf: %w", err)
	}
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return "", fmt.Errorf("save pdf: %w", err)
	}
	return filename, nil
}

func (be *BrowserEngine) GetResourceTiming() ([]map[string]interface{}, error) {
	page := be.currentPage()
	if page == nil {
		return nil, fmt.Errorf("browser not launched")
	}
	val, err := page.Eval(`() => JSON.stringify(performance.getEntriesByType('resource'))`)
	if err != nil {
		return nil, fmt.Errorf("resource timing: %w", err)
	}
	var entries []map[string]interface{}
	json.Unmarshal([]byte(val.Value.String()), &entries)
	return entries, nil
}

func (be *BrowserEngine) GetPerformanceMetrics() (map[string]interface{}, error) {
	page := be.currentPage()
	if page == nil {
		return nil, fmt.Errorf("browser not launched")
	}
	val, err := page.Eval(`() => {
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

func (be *BrowserEngine) SwitchToFrame(selector string) error {
	page := be.currentPage()
	if page == nil {
		return fmt.Errorf("browser not launched")
	}
	frame, err := page.Element(selector)
	if err != nil {
		return fmt.Errorf("frame not found %s: %w", selector, err)
	}
	framePage, err := frame.Frame()
	if err != nil {
		return fmt.Errorf("get frame page: %w", err)
	}
	if framePage != nil {
		be.tabGroup.Add(framePage)
	}
	return nil
}

func (be *BrowserEngine) SwitchToMain() error {
	return be.SwitchTab(0)
}

func (be *BrowserEngine) DownloadFile(url string) (string, error) {
	if be.browser == nil {
		return "", fmt.Errorf("browser not launched")
	}
	if err := os.MkdirAll(be.downloadDir, 0755); err != nil {
		return "", fmt.Errorf("create download dir: %w", err)
	}

	wait := be.browser.WaitDownload(be.downloadDir)
	if err := be.Navigate(url); err != nil {
		return "", fmt.Errorf("navigate for download: %w", err)
	}
	dl := wait()
	base := filepath.Base(dl.URL)
	if base == "" || base == "." {
		base = fmt.Sprintf("download_%d", time.Now().UnixNano())
	}
	filename := filepath.Join(be.downloadDir, base)
	return filename, nil
}

func (be *BrowserEngine) DownloadPath() string {
	return be.downloadDir
}

func (be *BrowserEngine) ScreenshotFullPage(name string) (string, error) {
	page := be.currentPage()
	if page == nil {
		return "", fmt.Errorf("browser not launched")
	}
	if err := os.MkdirAll(be.screenshotDir, 0755); err != nil {
		return "", fmt.Errorf("create screenshot dir: %w", err)
	}
	filename := filepath.Join(be.screenshotDir, fmt.Sprintf("%s_%d.png", name, time.Now().Unix()))
	data, err := page.Screenshot(true, &proto.PageCaptureScreenshot{
		Format: proto.PageCaptureScreenshotFormatPng,
	})
	if err != nil {
		return "", fmt.Errorf("full page screenshot: %w", err)
	}
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return "", fmt.Errorf("save screenshot: %w", err)
	}
	return filename, nil
}

func (be *BrowserEngine) ScreenshotElement(name, selector string) (string, error) {
	page := be.currentPage()
	if page == nil {
		return "", fmt.Errorf("browser not launched")
	}
	el, err := page.Element(selector)
	if err != nil {
		return "", fmt.Errorf("element %s: %w", selector, err)
	}
	if err := os.MkdirAll(be.screenshotDir, 0755); err != nil {
		return "", fmt.Errorf("create screenshot dir: %w", err)
	}
	filename := filepath.Join(be.screenshotDir, fmt.Sprintf("%s_%d.png", name, time.Now().Unix()))
	data, err := el.Screenshot(proto.PageCaptureScreenshotFormatPng, 0)
	if err != nil {
		return "", fmt.Errorf("element screenshot: %w", err)
	}
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return "", fmt.Errorf("save screenshot: %w", err)
	}
	return filename, nil
}

func (be *BrowserEngine) NetworkBlock(patterns ...string) error {
	for _, pat := range patterns {
		be.networkRules = append(be.networkRules, NetworkRule{URLPattern: pat, Action: "block"})
	}
	return nil
}

func (be *BrowserEngine) NetworkMock(patterns string, statusCode int, body string) error {
	be.networkRules = append(be.networkRules, NetworkRule{
		URLPattern: patterns, Action: "mock", StatusCode: statusCode, Body: body,
	})
	return nil
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func (be *BrowserEngine) WaitLoadStable(timeoutSec int) error {
	page := be.currentPage()
	if page == nil {
		return fmt.Errorf("browser not launched")
	}
	return page.WaitStable(1 * time.Second)
}
