// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package progress

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ProgressBar represents a simple progress bar
type ProgressBar struct {
	total       int
	current     int
	width       int
	description string
	startTime   time.Time
	lastUpdate  time.Time
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int, description string) *ProgressBar {
	return &ProgressBar{
		total:       total,
		current:     0,
		width:       50,
		description: description,
		startTime:   time.Now(),
		lastUpdate:  time.Now(),
	}
}

// Update increments the progress bar
func (pb *ProgressBar) Update(increment int) {
	pb.current += increment

	// Only update display every 100ms to avoid flickering
	if time.Since(pb.lastUpdate) > 100*time.Millisecond || pb.current >= pb.total {
		pb.render()
		pb.lastUpdate = time.Now()
	}
}

// SetCurrent sets the current progress value
func (pb *ProgressBar) SetCurrent(current int) {
	pb.current = current
	if time.Since(pb.lastUpdate) > 100*time.Millisecond || pb.current >= pb.total {
		pb.render()
		pb.lastUpdate = time.Now()
	}
}

// Finish completes the progress bar
func (pb *ProgressBar) Finish() {
	pb.current = pb.total
	pb.render()
	fmt.Println() // New line after completion
}

// render draws the progress bar
func (pb *ProgressBar) render() {
	percentage := float64(pb.current) / float64(pb.total) * 100
	if percentage > 100 {
		percentage = 100
	}

	filled := int(float64(pb.width) * percentage / 100)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", pb.width-filled)

	elapsed := time.Since(pb.startTime)

	// Estimate time remaining
	var eta string
	if pb.current > 0 && pb.current < pb.total {
		rate := float64(pb.current) / elapsed.Seconds()
		remaining := float64(pb.total-pb.current) / rate
		eta = fmt.Sprintf(" ETA: %s", formatDuration(time.Duration(remaining)*time.Second))
	} else if pb.current >= pb.total {
		eta = fmt.Sprintf(" Done in %s", formatDuration(elapsed))
	} else {
		eta = ""
	}

	// Format: Description [██████████░░░░░░░░] 65% (650/1000) ETA: 2s
	fmt.Printf("\r%s [%s] %.1f%% (%d/%d)%s",
		pb.description, bar, percentage, pb.current, pb.total, eta)
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// Spinner represents a simple spinner for indeterminate progress
type Spinner struct {
	message string
	frames  []string
	delay   time.Duration

	done chan struct{}
	wg   sync.WaitGroup
	once sync.Once
}

// NewSpinner creates a new spinner
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		delay:   100 * time.Millisecond,
		done:    make(chan struct{}),
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		i := 0
		for {
			select {
			case <-s.done:
				fmt.Print("\r\033[K")
				return
			default:
				frame := s.frames[i%len(s.frames)]
				fmt.Printf("\r%s %s", frame, s.message)
				time.Sleep(s.delay)
				i++
			}
		}
	}()
}

// Stop ends the spinner
func (s *Spinner) Stop() {
	s.once.Do(func() {
		close(s.done)
		s.wg.Wait()
	})
}

// UpdateMessage changes the spinner message
func (s *Spinner) UpdateMessage(message string) {
	s.message = message
}
