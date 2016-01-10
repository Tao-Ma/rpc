package rpc

import ()

type BackgroudService struct {
	running bool
	err     error

	quit  chan struct{}
	ready chan bool

	s Service
}

func NewBackgroundService(s Service) (*BackgroudService, error) {
	if s == nil {
		return nil, ErrServiceInvalidArg
	}

	bg := new(BackgroudService)

	bg.s = s

	return bg, nil
}

func (bg *BackgroudService) run() {
	bg.ready <- true
	bg.s.Loop(bg.quit)
	close(bg.ready)
}

func (bg *BackgroudService) Run() {
	if bg.running {
		return
	}

	bg.quit = make(chan struct{}, 1)
	bg.ready = make(chan bool, 1)

	go bg.run()

	select {
	case <-bg.ready:
		bg.running = true
	}
}

func (bg *BackgroudService) Stop() {
	if !bg.running {
		return
	}

	bg.running = false

	close(bg.quit)
	bg.s.StopLoop(false)

	select {
	case <-bg.ready:
		bg.s.Cleanup()
	}
}
