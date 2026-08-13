package loadtest

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ApiClient/internal/models"
)

// Executor executes a single HTTP request.
type Executor interface {
	Execute(ctx context.Context, req models.HttpRequest) (models.HttpResponse, error)
}

// State tracks a running load test.
type State struct {
	ID        string
	Config    models.LoadTestConfig
	BaseReq   models.HttpRequest
	StartTime time.Time
	Ctx       context.Context
	Cancel    context.CancelFunc

	completed   int64
	success     int64
	fail        int64
	totalBytes  int64
	totalTimeMs int64
	maxTimeMs   int64
	minTimeMs   int64
	statMu      sync.Mutex

	lastCompleted int64
	lastBytes     int64
	lastTime      time.Time

	logFile *os.File
	logMu   sync.Mutex
}

// Engine manages load tests.
type Engine struct {
	mu    sync.Mutex
	tests map[string]*State
	exec  Executor
	emit  func(event string, data ...any)
}

// NewEngine creates a new load test engine.
func NewEngine(exec Executor, emit func(event string, data ...any)) *Engine {
	return &Engine{
		tests: make(map[string]*State),
		exec:  exec,
		emit:  emit,
	}
}

// Start starts a new load test and returns its ID.
func (e *Engine) Start(req models.HttpRequest, config models.LoadTestConfig, logDir string) (string, error) {
	if config.Concurrency <= 0 {
		config.Concurrency = 1
	}
	if config.Mode == string(models.LoadTestModeRounds) && config.Rounds <= 0 {
		return "", fmt.Errorf("压测轮次必须大于 0")
	}
	if config.Mode == string(models.LoadTestModeDuration) && config.DurationSeconds <= 0 {
		return "", fmt.Errorf("压测时长必须大于 0")
	}

	id := generateID()
	var ctx context.Context
	var cancel context.CancelFunc
	if config.Mode == string(models.LoadTestModeDuration) {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(config.DurationSeconds)*time.Second)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	s := &State{
		ID:        id,
		Config:    config,
		BaseReq:   req,
		StartTime: time.Now(),
		Ctx:       ctx,
		Cancel:    cancel,
		minTimeMs: -1,
		lastTime:  time.Now(),
	}

	if config.LogEnabled && logDir != "" {
		if err := s.openLog(logDir); err != nil {
			cancel()
			return "", fmt.Errorf("无法创建日志文件: %w", err)
		}
	}

	e.mu.Lock()
	e.tests[id] = s
	e.mu.Unlock()

	go e.run(s)
	return id, nil
}

// Stop terminates the load test with the given ID.
func (e *Engine) Stop(id string) error {
	e.mu.Lock()
	s, ok := e.tests[id]
	e.mu.Unlock()
	if !ok {
		return fmt.Errorf("压测任务不存在或已结束")
	}
	s.Cancel()
	return nil
}

func (e *Engine) run(s *State) {
	defer e.cleanup(s)

	var wg sync.WaitGroup
	for i := 0; i < s.Config.Concurrency; i++ {
		wg.Add(1)
		go e.worker(s, &wg)
	}

	// 实时进度推送协程
	ticker := time.NewTicker(200 * time.Millisecond)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				e.emitProgress(s)
			case <-done:
				return
			}
		}
	}()

	wg.Wait()
	ticker.Stop()
	close(done)

	// 推送最终结果
	e.emitComplete(s)
}

func (e *Engine) worker(s *State, wg *sync.WaitGroup) {
	defer wg.Done()
	round := 0
	for {
		select {
		case <-s.Ctx.Done():
			return
		default:
		}

		if s.Config.Mode == string(models.LoadTestModeRounds) {
			if round >= s.Config.Rounds {
				return
			}
		}

		e.executeOnce(s)
		round++
	}
}

func (e *Engine) executeOnce(s *State) {
	ctx, cancel := context.WithTimeout(s.Ctx, 30*time.Second)
	defer cancel()

	start := time.Now()
	resp, err := e.exec.Execute(ctx, s.BaseReq)
	duration := time.Since(start).Milliseconds()

	atomic.AddInt64(&s.completed, 1)
	atomic.AddInt64(&s.totalTimeMs, duration)
	atomic.AddInt64(&s.totalBytes, resp.SizeBytes)

	success := err == nil && resp.StatusCode < http.StatusBadRequest
	if success {
		atomic.AddInt64(&s.success, 1)
	} else {
		atomic.AddInt64(&s.fail, 1)
	}

	s.statMu.Lock()
	if s.minTimeMs == -1 || duration < s.minTimeMs {
		s.minTimeMs = duration
	}
	if duration > s.maxTimeMs {
		s.maxTimeMs = duration
	}
	s.statMu.Unlock()

	if s.logFile != nil {
		s.logRequest(start, duration, resp, err)
	}
}

func (e *Engine) emitProgress(s *State) {
	completed := atomic.LoadInt64(&s.completed)
	totalBytes := atomic.LoadInt64(&s.totalBytes)

	now := time.Now()
	elapsedSec := now.Sub(s.lastTime).Seconds()
	var rps, speed float64
	if elapsedSec > 0 {
		rps = float64(completed-s.lastCompleted) / elapsedSec
		speed = float64(totalBytes-s.lastBytes) / elapsedSec
	}
	s.lastCompleted = completed
	s.lastBytes = totalBytes
	s.lastTime = now

	total := e.totalRequests(s)
	var percent float64
	if total > 0 {
		percent = float64(completed) / float64(total) * 100
		if percent > 100 {
			percent = 100
		}
	} else if s.Config.Mode == string(models.LoadTestModeDuration) {
		elapsedTotal := now.Sub(s.StartTime).Seconds()
		percent = elapsedTotal / float64(s.Config.DurationSeconds) * 100
		if percent > 100 {
			percent = 100
		}
	}

	e.emit("loadtest:progress", models.LoadTestProgress{
		ID:            s.ID,
		Completed:     completed,
		Total:         total,
		Percent:       percent,
		RPS:           rps,
		DownloadSpeed: speed,
		Running:       true,
	})
}

func (e *Engine) emitComplete(s *State) {
	completed := atomic.LoadInt64(&s.completed)
	success := atomic.LoadInt64(&s.success)
	fail := atomic.LoadInt64(&s.fail)
	totalBytes := atomic.LoadInt64(&s.totalBytes)
	totalTimeMs := atomic.LoadInt64(&s.totalTimeMs)

	durationSec := time.Since(s.StartTime).Seconds()
	var avgTimeMs float64
	if completed > 0 {
		avgTimeMs = float64(totalTimeMs) / float64(completed)
	}

	s.statMu.Lock()
	minTimeMs := s.minTimeMs
	maxTimeMs := s.maxTimeMs
	s.statMu.Unlock()
	if minTimeMs < 0 {
		minTimeMs = 0
	}

	result := models.LoadTestResult{
		TotalRequests: completed,
		DurationSec:   durationSec,
		SuccessCount:  success,
		FailCount:     fail,
		ErrorRate:     0,
		TotalBytesMB:  float64(totalBytes) / 1024 / 1024,
		RPS:           0,
		SuccessRPS:    0,
		BytesPerSecMB: 0,
		MaxTimeMs:     maxTimeMs,
		MinTimeMs:     minTimeMs,
		AvgTimeMs:     avgTimeMs,
	}
	if completed > 0 {
		result.ErrorRate = float64(fail) / float64(completed) * 100
		result.RPS = float64(completed) / durationSec
	}
	if success > 0 {
		result.SuccessRPS = float64(success) / durationSec
	}
	if durationSec > 0 {
		result.BytesPerSecMB = float64(totalBytes) / 1024 / 1024 / durationSec
	}

	e.emit("loadtest:progress", models.LoadTestProgress{
		ID:      s.ID,
		Percent: 100,
		Running: false,
	})
	e.emit("loadtest:complete", result)
}

func (e *Engine) totalRequests(s *State) int64 {
	if s.Config.Mode == string(models.LoadTestModeRounds) {
		return int64(s.Config.Concurrency * s.Config.Rounds)
	}
	return 0
}

func (e *Engine) cleanup(s *State) {
	s.Cancel()
	if s.logFile != nil {
		s.logFile.Close()
	}
	e.mu.Lock()
	delete(e.tests, s.ID)
	e.mu.Unlock()
}

func (s *State) openLog(logDir string) error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}
	fileName := s.StartTime.Format("2006-01-02_15-04-05") + ".log"
	path := filepath.Join(logDir, fileName)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	s.logFile = f
	return nil
}

func (s *State) logRequest(start time.Time, duration int64, resp models.HttpResponse, err error) {
	s.logMu.Lock()
	defer s.logMu.Unlock()
	if s.logFile == nil {
		return
	}
	status := resp.Status
	if status == "" && err != nil {
		status = err.Error()
	}
	line := fmt.Sprintf("%s | %s | %s | status=%s | time=%dms | size=%d | err=%v\n",
		start.Format("2006-01-02 15:04:05.000"),
		s.BaseReq.Method,
		s.BaseReq.ResolvedURL,
		status,
		duration,
		resp.SizeBytes,
		err,
	)
	_, _ = s.logFile.WriteString(line)
}

func generateID() string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 12)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// SanitizeName removes invalid filename characters.
func SanitizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "loadtest"
	}
	re := regexp.MustCompile(`[\\/:*?"<>|]`)
	return re.ReplaceAllString(name, "_")
}
