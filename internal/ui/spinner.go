package ui

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
)

type Spinner struct {
	label     string
	count     int64
	stopChan  chan struct{}
	doneChan  chan struct{}
	isRunning int32
}

func NewSpinner(label string) *Spinner {
	return &Spinner{
		label:    label,
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}
}

func (s *Spinner) Increment() {
	atomic.AddInt64(&s.count, 1)
}

func (s *Spinner) Add(n int64) {
	atomic.AddInt64(&s.count, n)
}

func (s *Spinner) Count() int64 {
	return atomic.LoadInt64(&s.count)
}

func (s *Spinner) Start() {
	if !atomic.CompareAndSwapInt32(&s.isRunning, 0, 1) {
		return
	}

	go func() {
		defer close(s.doneChan)
		spinChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-s.stopChan:
				fmt.Print("\r\033[2K\r")
				return
			case <-time.After(80 * time.Millisecond):
				c := atomic.LoadInt64(&s.count)
				fmt.Printf("\r\033[2K  %s %s (%d files scanned)", color.CyanString(spinChars[i%len(spinChars)]), s.label, c)
				i++
			}
		}
	}()
}

func (s *Spinner) Stop() {
	if !atomic.CompareAndSwapInt32(&s.isRunning, 1, 0) {
		return
	}
	close(s.stopChan)
	<-s.doneChan
	time.Sleep(20 * time.Millisecond)
	fmt.Print("\r\033[2K\r")
}

