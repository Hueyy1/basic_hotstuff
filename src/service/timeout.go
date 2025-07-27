package service

import "time"

type TimeoutService interface {
	SoftStart()
	Reset()
	Stop()
	Timeout() <-chan time.Time
	Duration() time.Duration
}

type TimeoutServiceImpl struct {
	duration  time.Duration
	Timer     *time.Timer
	isStarted bool
}

func NewTimeoutService(duration time.Duration) TimeoutService {
	t := &TimeoutServiceImpl{duration: duration, Timer: time.NewTimer(duration)}
	t.Stop()
	return t
}

func (t *TimeoutServiceImpl) SoftStart() {
	if t.isStarted {
		return
	}
	t.Reset()
	t.isStarted = true
}

func (t *TimeoutServiceImpl) Reset() {
	t.Timer.Reset(t.duration)
}

func (t *TimeoutServiceImpl) Stop() {
	t.Timer.Stop()
	t.isStarted = false
}

func (t *TimeoutServiceImpl) Timeout() <-chan time.Time {
	return t.Timer.C
}

func (t *TimeoutServiceImpl) Duration() time.Duration {
	return t.duration
}
