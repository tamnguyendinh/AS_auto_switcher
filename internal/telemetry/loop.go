package telemetry

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// DefaultLoopInterval is the authoritative 60-second auto-telemetry period.
	DefaultLoopInterval = 60 * time.Second
)

var (
	// ErrLoopAlreadyRunning is returned when attempting to start an already running loop.
	ErrLoopAlreadyRunning = errors.New("auto telemetry loop is already running")

	// ErrNilPool is returned when the telemetry pool is nil.
	ErrNilPool = errors.New("telemetry pool cannot be nil")
)

// AutoTelemetryLoop manages periodic background quota telemetry polling.
type AutoTelemetryLoop struct {
	pool         *TelemetryPool
	interval     time.Duration
	isRefreshing atomic.Bool // Single-flight gate: drops overlapping executions
	running      atomic.Bool // True when the background loop is active
	triggerChan  chan struct{}
	stopChan     chan struct{}
	doneChan     chan struct{}
	stopOnce     sync.Once
	ctx          context.Context
	cancel       context.CancelFunc
	onUpdate     func([]*AccountQuotaData)
	onUpdateMu   sync.RWMutex
	mu           sync.Mutex
}

// NewAutoTelemetryLoop creates an AutoTelemetryLoop with the specified interval.
// If interval <= 0, DefaultLoopInterval (60s) is used.
func NewAutoTelemetryLoop(pool *TelemetryPool, interval time.Duration) (*AutoTelemetryLoop, error) {
	if pool == nil {
		return nil, ErrNilPool
	}
	if interval <= 0 {
		interval = DefaultLoopInterval
	}
	return &AutoTelemetryLoop{
		pool:     pool,
		interval: interval,
	}, nil
}

// Interval returns the configured polling interval.
func (l *AutoTelemetryLoop) Interval() time.Duration {
	return l.interval
}

// IsRunning returns whether the background telemetry loop is actively running.
func (l *AutoTelemetryLoop) IsRunning() bool {
	return l.running.Load()
}

// IsRefreshing returns whether a telemetry polling pass is currently in flight.
func (l *AutoTelemetryLoop) IsRefreshing() bool {
	return l.isRefreshing.Load()
}

// SetUpdateCallback sets a callback invoked after each successful polling cycle.
func (l *AutoTelemetryLoop) SetUpdateCallback(cb func([]*AccountQuotaData)) {
	l.onUpdateMu.Lock()
	defer l.onUpdateMu.Unlock()
	l.onUpdate = cb
}

// Start launches the background 60s ticker loop.
func (l *AutoTelemetryLoop) Start(parentCtx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.running.Load() {
		return ErrLoopAlreadyRunning
	}

	if parentCtx == nil {
		parentCtx = context.Background()
	}
	l.ctx, l.cancel = context.WithCancel(parentCtx)
	l.triggerChan = make(chan struct{}, 1)
	l.stopChan = make(chan struct{})
	l.doneChan = make(chan struct{})
	l.stopOnce = sync.Once{}
	l.running.Store(true)

	go l.run()
	return nil
}

func (l *AutoTelemetryLoop) run() {
	defer close(l.doneChan)
	defer l.running.Store(false)

	ticker := time.NewTicker(l.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.poll()
		case <-l.triggerChan:
			ticker.Reset(l.interval)
			l.poll()
		case <-l.stopChan:
			return
		case <-l.ctx.Done():
			return
		}
	}
}

func (l *AutoTelemetryLoop) poll() {
	// Single-flight gate: if a poll is already in progress, drop this tick immediately
	if !l.isRefreshing.CompareAndSwap(false, true) {
		return
	}
	defer l.isRefreshing.Store(false)

	baseCtx := l.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	pollCtx, cancel := context.WithTimeout(baseCtx, 10*time.Second)
	defer cancel()

	results, err := l.pool.PollAll(pollCtx)
	if err == nil {
		l.onUpdateMu.RLock()
		cb := l.onUpdate
		l.onUpdateMu.RUnlock()
		if cb != nil {
			cb(results)
		}
	}
}

// TriggerNow triggers an immediate polling pass and resets the 60s ticker.
// If another poll is already in flight, the single-flight gate drops the duplicate.
func (l *AutoTelemetryLoop) TriggerNow() {
	if !l.running.Load() {
		go l.poll()
		return
	}

	select {
	case l.triggerChan <- struct{}{}:
	default:
		// Trigger already queued
	}
}

// Stop safely stops the background telemetry loop and waits for exit.
func (l *AutoTelemetryLoop) Stop() {
	if !l.running.Load() {
		return
	}

	l.stopOnce.Do(func() {
		if l.cancel != nil {
			l.cancel()
		}
		close(l.stopChan)
	})

	<-l.doneChan
}
