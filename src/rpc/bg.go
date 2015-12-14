package rpc

import (
	"time"
)

type ServiceLoop interface {
	ServiceLoop(chan struct{}, chan bool)
}

type BackgroudService struct {
	running bool
	err     error

	quit  chan struct{}
	ready chan bool

	l ServiceLoop

	start_timeout time.Duration
	stop_timeout  time.Duration
}

func NewBackgroundService(l ServiceLoop) (*BackgroudService, error) {
	bg := new(BackgroudService)

	bg.l = l
	bg.start_timeout = 3000 * time.Millisecond
	bg.stop_timeout = 3000 * time.Millisecond

	return bg, nil
}

func (bg *BackgroudService) Run() {
	if bg.running {
		return
	}

	bg.quit = make(chan struct{}, 1)
	bg.ready = make(chan bool, 1)

	go bg.l.ServiceLoop(bg.quit, bg.ready)

	select {
	case <-bg.ready:
		bg.running = true
	case <-time.Tick(bg.start_timeout):
		close(bg.quit)
		// TODO: bg.err
		break
	}

	close(bg.ready)
}

func (bg *BackgroudService) Stop() {
	if !bg.running {
		return
	}

	bg.running = false
	close(bg.quit)
}
