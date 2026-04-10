package requests

import (
	"monitor/internal/config"
	"monitor/internal/types"
)

// Semaphore limits the number of concurrent requests.
type Semaphore chan types.Void

// NewSemaphore returns a semaphore sized for the configured concurrency limit.
func NewSemaphore(cfg *config.Config) *Semaphore {
	s := Semaphore(make(chan types.Void, cfg.MaxConcurrentRequests))
	return &s
}

// Acquire reserves one semaphore slot.
func (s Semaphore) Acquire() {
	s <- types.Void{}
}

// Release frees one semaphore slot.
func (s Semaphore) Release() {
	<-s
}
