package util

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ProgressBar represents a simple progress bar for CLI operations
type ProgressBar struct {
	total      int
	current    int
	width      int
	prefix     string
	suffix     string
	mu         sync.Mutex
	startTime  time.Time
	lastUpdate time.Time
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int, prefix string) *ProgressBar {
	return &ProgressBar{
		total:     total,
		current:   0,
		width:     50,
		prefix:    prefix,
		startTime: time.Now(),
	}
}

// Increment increments the progress by 1
func (p *ProgressBar) Increment() {
	p.Add(1)
}

// Add increments the progress by n
func (p *ProgressBar) Add(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current += n
	if p.current > p.total {
		p.current = p.total
	}

	// Rate limit updates to avoid excessive printing
	now := time.Now()
	if now.Sub(p.lastUpdate) < 100*time.Millisecond && p.current < p.total {
		return
	}
	p.lastUpdate = now

	p.render()
}

// SetCurrent sets the current progress value
func (p *ProgressBar) SetCurrent(current int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = current
	if p.current > p.total {
		p.current = p.total
	}
	p.render()
}

// Finish completes the progress bar
func (p *ProgressBar) Finish() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = p.total
	p.render()
	fmt.Println() // New line after completion
}

// render displays the progress bar
func (p *ProgressBar) render() {
	percent := float64(p.current) / float64(p.total)
	filled := int(percent * float64(p.width))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.width-filled)

	elapsed := time.Since(p.startTime)

	// Calculate ETA
	var eta string
	if p.current > 0 && p.current < p.total {
		rate := float64(p.current) / elapsed.Seconds()
		remaining := float64(p.total-p.current) / rate
		eta = fmt.Sprintf(" ETA: %s", time.Duration(remaining*float64(time.Second)).Round(time.Second))
	}

	fmt.Printf("\r%s [%s] %d/%d (%.0f%%)%s",
		p.prefix, bar, p.current, p.total, percent*100, eta)
}

// Spinner represents a simple spinner for indeterminate operations
type Spinner struct {
	message string
	scope   string
	frames  []string
	current int
	stop    chan bool
	mu      sync.Mutex
	active  bool
}

// NewSpinner creates a new spinner with a message and scope
func NewSpinner(message string, scope string) *Spinner {
	return &Spinner{
		message: message,
		scope:   scope,
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		stop:    make(chan bool),
	}
}

// Start starts the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.stop:
				return
			case <-ticker.C:
				s.mu.Lock()
				if s.scope != "" {
					// Cyan color for scope
					fmt.Printf("\r\033[36m[%s]\033[0m %s %s ", s.scope, s.frames[s.current], s.message)
				} else {
					fmt.Printf("\r%s %s ", s.frames[s.current], s.message)
				}
				s.current = (s.current + 1) % len(s.frames)
				s.mu.Unlock()
			}
		}
	}()
}

// Stop stops the spinner and optionally clears the line
func (s *Spinner) Stop(clearLine bool) {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	s.mu.Unlock()

	s.stop <- true
	if clearLine {
		// Clear the line by overwriting with spaces
		fmt.Print("\r\033[K")
	}
}

// UpdateMessage updates the spinner message
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}
