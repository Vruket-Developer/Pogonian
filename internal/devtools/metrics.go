package devtools

import (
	"runtime"
	"sync"
	"time"
)

// RequestMetric captures performance telemetry for a single HTTP request.
type RequestMetric struct {
	ID        int64         `json:"id"`
	Method    string        `json:"method"`
	Path      string        `json:"path"`
	Status    int           `json:"status"`
	Duration  time.Duration `json:"duration"`
	DurationStr string      `json:"duration_str"`
	AllocKB   uint64        `json:"alloc_kb"`
	Timestamp time.Time     `json:"timestamp"`
}

// TelemetryRecorder stores recent request metrics in memory.
type TelemetryRecorder struct {
	mu      sync.Mutex
	metrics []*RequestMetric
	maxSize int
	counter int64
}

var GlobalRecorder = NewTelemetryRecorder(50)

// NewTelemetryRecorder creates a new recorder with fixed capacity.
func NewTelemetryRecorder(maxSize int) *TelemetryRecorder {
	return &TelemetryRecorder{
		metrics: make([]*RequestMetric, 0, maxSize),
		maxSize: maxSize,
	}
}

// Record captures a completed request's metrics.
func (tr *TelemetryRecorder) Record(method, path string, status int, duration time.Duration) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	tr.counter++
	metric := &RequestMetric{
		ID:          tr.counter,
		Method:      method,
		Path:        path,
		Status:      status,
		Duration:    duration,
		DurationStr: duration.Round(time.Microsecond).String(),
		AllocKB:     m.Alloc / 1024,
		Timestamp:   time.Now().UTC(),
	}

	if len(tr.metrics) >= tr.maxSize {
		tr.metrics = tr.metrics[1:]
	}
	tr.metrics = append(tr.metrics, metric)
}

// GetMetrics returns a copy of all recorded request metrics.
func (tr *TelemetryRecorder) GetMetrics() []*RequestMetric {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	result := make([]*RequestMetric, len(tr.metrics))
	copy(result, tr.metrics)
	return result
}
