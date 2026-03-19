package requests

import (
	"monitor/internal/config"
	"monitor/internal/types"
)

type Semaphore chan types.Void

func NewSemaphore(cfg *config.Config) *Semaphore {
	s := Semaphore(make(chan types.Void, cfg.MaxConcurrentRequests))
	return &s
}

func (s Semaphore) Acquire() {
	s <- types.Void{}
}

func (s Semaphore) Release() {
	<-s
}
