package engine

import (
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
