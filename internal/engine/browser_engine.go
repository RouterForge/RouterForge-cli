package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type BrowserEngine struct {
	browser       *rod.Browser
	page          *rod.Page
	pages         []*rod.Page
	screenshotDir string
}

func NewBrowserEngine(screenshotDir string) *BrowserEngine {
	return &BrowserEngine{
		screenshotDir: screenshotDir,
	}
}

func (be *BrowserEngine) Launch() error {
	path, found := launcher.LookPath()
	if !found {
		l := launcher.New()
		u, err := l.Launch()
		if err != nil {
			return fmt.Errorf("failed to launch browser: %w", err)
		}
		browser := rod.New().ControlURL(u)
		if err := browser.Connect(); err != nil {
			return fmt.Errorf("failed to connect to browser: %w", err)
		}
		be.browser = browser
	} else {
		u := launcher.New().Bin(path).MustLaunch()
		browser := rod.New().ControlURL(u)
		if err := browser.Connect(); err != nil {
			return fmt.Errorf("failed to connect to browser: %w", err)
		}
		be.browser = browser
	}

	page, err := be.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}
	be.page = page
	return nil
}

func (be *BrowserEngine) Navigate(url string) error {
	if be.page == nil {
		return fmt.Errorf("browser not launched")
	}
	return be.page.Navigate(url)
}

func (be *BrowserEngine) WaitLoad() error {
	if be.page == nil {
		return fmt.Errorf("browser not launched")
	}
	return be.page.WaitLoad()
}

func (be *BrowserEngine) Screenshot(name string) (string, error) {
	if be.page == nil {
		return "", fmt.Errorf("browser not launched")
	}

	if err := os.MkdirAll(be.screenshotDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create screenshot dir: %w", err)
	}

	filename := filepath.Join(be.screenshotDir, fmt.Sprintf("%s_%d.png", name, time.Now().Unix()))
	data, err := be.page.Screenshot(false, &proto.PageCaptureScreenshot{
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
	if be.page == nil {
		return fmt.Errorf("browser not launched")
	}
	el, err := be.page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found %s: %w", selector, err)
	}
	return el.Click(proto.InputMouseButtonLeft, 1)
}

func (be *BrowserEngine) Type(selector, text string) error {
	if be.page == nil {
		return fmt.Errorf("browser not launched")
	}
	el, err := be.page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found %s: %w", selector, err)
	}
	return el.Input(text)
}

func (be *BrowserEngine) HTML() (string, error) {
	if be.page == nil {
		return "", fmt.Errorf("browser not launched")
	}
	return be.page.HTML()
}

func (be *BrowserEngine) Evaluate(js string) (interface{}, error) {
	if be.page == nil {
		return nil, fmt.Errorf("browser not launched")
	}
	return be.page.Eval(js)
}

func (be *BrowserEngine) Close() {
	if be.browser != nil {
		be.browser.Close()
	}
}

func (be *BrowserEngine) Page() *rod.Page {
	return be.page
}

func (be *BrowserEngine) GetCookies() ([]*proto.NetworkCookie, error) {
	if be.page == nil {
		return nil, fmt.Errorf("browser not launched")
	}
	cookies, err := be.page.Cookies(nil)
	if err != nil {
		return nil, fmt.Errorf("get cookies: %w", err)
	}
	return cookies, nil
}

func (be *BrowserEngine) SetCookie(name, value, domain, path string) error {
	if be.page == nil {
		return fmt.Errorf("browser not launched")
	}
	cookie := &proto.NetworkCookieParam{
		Name:   name,
		Value:  value,
		Domain: domain,
		Path:   path,
	}
	return be.page.SetCookies([]*proto.NetworkCookieParam{cookie})
}

func (be *BrowserEngine) ClearCookies() error {
	if be.page == nil {
		return fmt.Errorf("browser not launched")
	}
	return be.page.SetCookies([]*proto.NetworkCookieParam{})
}

func (be *BrowserEngine) ConsoleLogs() ([]string, error) {
	if be.page == nil {
		return nil, fmt.Errorf("browser not launched")
	}
	var logs []string
	be.page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) {
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
	be.pages = append(be.pages, page)
	be.page = page
	return page, nil
}

func (be *BrowserEngine) SwitchTab(index int) error {
	if index < 0 || index >= len(be.pages) {
		return fmt.Errorf("tab index %d out of range (0-%d)", index, len(be.pages)-1)
	}
	be.page = be.pages[index]
	return nil
}

func (be *BrowserEngine) CloseTab(index int) error {
	if index < 0 || index >= len(be.pages) {
		return fmt.Errorf("tab index %d out of range", index)
	}
	page := be.pages[index]
	if err := page.Close(); err != nil {
		return err
	}
	be.pages = append(be.pages[:index], be.pages[index+1:]...)
	if len(be.pages) > 0 && index < len(be.pages) {
		be.page = be.pages[index]
	} else if len(be.pages) > 0 {
		be.page = be.pages[0]
	} else {
		be.page = nil
	}
	return nil
}

func (be *BrowserEngine) TabCount() int {
	return len(be.pages)
}

func (be *BrowserEngine) WaitForSelector(selector string, timeout time.Duration) error {
	if be.page == nil {
		return fmt.Errorf("browser not launched")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, err := be.page.Context(ctx).Element(selector)
	return err
}
