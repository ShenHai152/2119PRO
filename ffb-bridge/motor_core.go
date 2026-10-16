package main

import (
	"errors"
	"time"
)

var errMotorDisconnected = errors.New("motor output is disconnected")

type motorOutput interface {
	set(left, right float64) error
	healthy() error
	close()
}

type motorOpener func() (motorOutput, string, error)

type reconnectingMotor struct {
	opener      motorOpener
	output      motorOutput
	retry       time.Duration
	nextAttempt time.Time
	nextHealth  time.Time
}

func newReconnectingMotor(opener motorOpener, retry time.Duration) *reconnectingMotor {
	if retry <= 0 {
		retry = time.Second
	}
	return &reconnectingMotor{opener: opener, retry: retry}
}

func (m *reconnectingMotor) connected() bool {
	return m != nil && m.output != nil
}

func (m *reconnectingMotor) tick(now time.Time) (bool, string, error) {
	if m == nil || m.opener == nil || m.output != nil || now.Before(m.nextAttempt) {
		return false, "", nil
	}
	output, label, err := m.opener()
	if err != nil {
		m.nextAttempt = now.Add(m.retry)
		return false, "", err
	}
	if output == nil {
		m.nextAttempt = now.Add(m.retry)
		return false, "", errors.New("motor opener returned nil output")
	}
	if err := output.set(0, 0); err != nil {
		output.close()
		m.nextAttempt = now.Add(m.retry)
		return false, "", err
	}
	m.output = output
	m.nextAttempt = time.Time{}
	m.nextHealth = now.Add(time.Second)
	return true, label, nil
}

func (m *reconnectingMotor) healthCheck(now time.Time) error {
	if m == nil || m.output == nil || now.Before(m.nextHealth) {
		return nil
	}
	m.nextHealth = now.Add(time.Second)
	if err := m.output.healthy(); err != nil {
		m.output.close()
		m.output = nil
		m.nextAttempt = now.Add(m.retry)
		return err
	}
	return nil
}

func (m *reconnectingMotor) apply(now time.Time, left, right float64) error {
	if m == nil || m.output == nil {
		return errMotorDisconnected
	}
	if err := m.output.set(left, right); err != nil {
		m.output.close()
		m.output = nil
		m.nextAttempt = now.Add(m.retry)
		m.nextHealth = time.Time{}
		return err
	}
	return nil
}

func (m *reconnectingMotor) close() {
	if m != nil && m.output != nil {
		m.output.close()
		m.output = nil
		m.nextHealth = time.Time{}
	}
}
