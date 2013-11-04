package social

import (
	"fmt"
	"gnd.la/log"
	"gnd.la/mux"
	"gnd.la/tasks"
	"time"
)

const (
	pollInterval = 5 * time.Minute
)

type Sharer struct {
	service  Service
	interval time.Duration
	provider ShareProvider
	config   interface{}
	task     *tasks.Task
}

func (s *Sharer) share(ctx *mux.Context) {
	last, err := s.provider.LastShare(ctx, s.service)
	if err != nil {
		log.Errorf("error finding last share time on %s: %s", s.service, err)
		return
	}
	if last.Before(time.Now().Add(-s.interval)) {
		item, err := s.provider.Item(ctx, s.service)
		if err != nil {
			log.Errorf("error finding next time for sharing on %s: %s", s.service, err)
			return
		}
		if item != nil {
			result, err := Share(s.service, item, s.config)
			if err != nil {
				log.Errorf("error sharing on %s: %s", s.service, err)
			}
			s.provider.Shared(ctx, s.service, item, result, err)
		}
	}
}

func (s *Sharer) Schedule(m *mux.Mux, interval time.Duration) {
	if s.task != nil {
		s.task.Stop()
	}
	s.interval = interval
	s.task = tasks.Schedule(m, pollInterval, nil, s.share)
}

func (s *Sharer) Stop() {
	if s.task != nil {
		s.task.Stop()
		s.task = nil
	}
}

func NewSharer(s Service, provider ShareProvider, config interface{}) *Sharer {
	if provider == nil {
		panic(fmt.Errorf("provider can't be nil"))
	}
	if err := validateConfig(s, config); err != nil {
		panic(err)
	}
	return &Sharer{
		service:  s,
		provider: provider,
		config:   config,
	}
}
