package monitor

import (
	"sync"
	"time"

	runtime "github.com/testground/sdk-go/runtime"
)

// Monitor represents a monitoring entity.
type Monitor struct {
	isCancel   bool          // Indicates whether the monitoring operation should be canceled.
	MaxRuntime time.Duration // Maximum runtime in milliseconds.
	startTime  time.Time     // Start time of execution.
	mutex      sync.Mutex    // Mutex for safe access to fields.
	isDebug    bool
}

// NewMonitor creates a new Monitor instance with default values.
func NewMonitor(maxRuntime time.Duration, debug bool) *Monitor {
	return &Monitor{
		MaxRuntime: maxRuntime,
		isCancel:   false,
		mutex:      sync.Mutex{},
		isDebug:    debug,
	}
}

// Start sets the start time of the execution and waits until the maximum runtime has been reached to set IsCancel to true.
func (m *Monitor) Start() {
	m.mutex.Lock()
	m.startTime = time.Now()
	m.mutex.Unlock()

	// Wait until max runtime is reached
	<-time.After(m.MaxRuntime)

	m.mutex.Lock()
	m.isCancel = true
	m.mutex.Unlock()
}

func (m *Monitor) SetCancel(cancel bool) {
	m.mutex.Lock()
	m.isCancel = cancel
	m.mutex.Unlock()
}

func (m *Monitor) IsCancel() bool {
	var res bool
	m.mutex.Lock()
	res = m.isCancel
	m.mutex.Unlock()
	return res
}

// GetElapsedTime returns the elapsed time since the start of the monitoring operation.
func (m *Monitor) GetElapsedTime() time.Duration {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return time.Since(m.startTime)
}

func (m *Monitor) Log(msg string, a ...interface{}) {
	runtime.CurrentRunEnv().RecordMessage(msg, a...)
}

func (m *Monitor) Debug(msg string, a ...interface{}) {
	if m.isDebug {
		runtime.CurrentRunEnv().RecordMessage(msg, a...)
	}
}

func (m *Monitor) IsDebug() bool {
	return m.isDebug
}

func (m *Monitor) FailWithError(err error) {
	runtime.CurrentRunEnv().RecordFailure(err)
}
