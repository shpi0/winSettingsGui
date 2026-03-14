package scheduler

import (
	"sync"
	"time"

	"winSettingsGui/internal/config"
	"winSettingsGui/internal/power"
)

type Scheduler struct {
	mu             sync.Mutex
	jobs           []config.ScheduledJob
	lastFired      string
	stopCh         chan struct{}
	OnJobExecuted  func()
}

func New() *Scheduler {
	return &Scheduler{
		stopCh: make(chan struct{}),
	}
}

func (s *Scheduler) UpdateJobs(jobs []config.ScheduledJob) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = make([]config.ScheduledJob, len(jobs))
	copy(s.jobs, jobs)
}

func (s *Scheduler) Start() {
	go s.loop()
}

func (s *Scheduler) Stop() {
	close(s.stopCh)
}

func (s *Scheduler) loop() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case now := <-ticker.C:
			s.tick(now)
		}
	}
}

func (s *Scheduler) tick(now time.Time) {
	key := now.Format("2006-01-02 15:04")

	s.mu.Lock()
	if s.lastFired == key {
		s.mu.Unlock()
		return
	}
	s.lastFired = key

	weekday := convertWeekday(now.Weekday())
	hour, minute := now.Hour(), now.Minute()

	var toExec []config.ScheduledJob
	for _, j := range s.jobs {
		if j.Active && j.Weekdays[weekday] && j.Hour == hour && j.Minute == minute {
			toExec = append(toExec, j)
		}
	}
	s.mu.Unlock()

	for _, j := range toExec {
		executeJob(j)
	}

	if len(toExec) > 0 && s.OnJobExecuted != nil {
		s.OnJobExecuted()
	}
}

func convertWeekday(wd time.Weekday) int {
	if wd == time.Sunday {
		return 6
	}
	return int(wd) - 1
}

func executeJob(j config.ScheduledJob) {
	for _, a := range j.Actions {
		src := power.AC
		if a.Source == config.SourceDC {
			src = power.DC
		}
		switch a.Type {
		case config.ActionDisplay:
			_ = power.SetDisplayTimeout(a.Minutes, src)
		case config.ActionSleep:
			_ = power.SetSleepTimeout(a.Minutes, src)
		case config.ActionHibernate:
			_ = power.SetHibernateTimeout(a.Minutes, src)
		}
	}
}
