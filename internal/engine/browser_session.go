package engine

import (
	"context"
	"fmt"
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
	return s.Page.Navigate(url)
}

func (s *BrowserSession) Screenshot(name string) ([]byte, error) {
	if s.Page == nil {
		return nil, fmt.Errorf("no page in session")
	}
	return s.Page.Screenshot(false, &proto.PageCaptureScreenshot{
		Format: proto.PageCaptureScreenshotFormatPng,
	})
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
