package clipboard

import (
	"context"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Monitor watches clipboard for fetch code.
type Monitor struct {
	ctx       context.Context
	lastText  string
	interval  time.Duration
	stopChan  chan struct{}
	isRunning bool
}

// NewMonitor creates a clipboard monitor.
func NewMonitor() *Monitor {
	return &Monitor{
		interval: 1 * time.Second,
		stopChan: make(chan struct{}),
	}
}

// SetContext sets the runtime context.
func (m *Monitor) SetContext(ctx context.Context) {
	m.ctx = ctx
}

// Start begins monitoring the clipboard.
func (m *Monitor) Start() {
	if m.isRunning {
		return
	}
	m.isRunning = true
	go m.loop()
}

// Stop halts monitoring.
func (m *Monitor) Stop() {
	if !m.isRunning {
		return
	}
	m.isRunning = false
	close(m.stopChan)
}

func (m *Monitor) loop() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.check()
		case <-m.stopChan:
			return
		}
	}
}

func (m *Monitor) check() {
	text, err := clipboard.ReadAll()
	if err != nil {
		return
	}
	if text == m.lastText {
		return
	}
	m.lastText = text
	if m.ctx == nil {
		return
	}
	if strings.Contains(text, "fetch(") {
		runtime.EventsEmit(m.ctx, "clipboard:fetch", text)
	}
}
