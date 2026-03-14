package shared

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

var spinnerFrames = []string{"|", "/", "-", "\\"}

type Spinner struct {
	out      io.Writer
	enabled  bool
	interval time.Duration

	mu      sync.Mutex
	once    sync.Once
	message string
	done    chan struct{}
}

func NewSpinner(out io.Writer, message string) *Spinner {
	return newSpinner(out, message, isTTYWriter(out), 100*time.Millisecond)
}

func newSpinner(out io.Writer, message string, enabled bool, interval time.Duration) *Spinner {
	if out == nil {
		out = os.Stderr
	}
	if interval <= 0 {
		interval = 100 * time.Millisecond
	}

	s := &Spinner{
		out:      out,
		enabled:  enabled,
		interval: interval,
		message:  message,
		done:     make(chan struct{}),
	}
	if enabled {
		go s.run()
	}
	return s
}

func (s *Spinner) Update(message string) {
	if s == nil || !s.enabled {
		return
	}
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

func (s *Spinner) Success(message string) {
	s.stop(message)
}

func (s *Spinner) Fail(message string) {
	s.stop(message)
}

func (s *Spinner) Stop() {
	s.stop("")
}

func (s *Spinner) stop(final string) {
	if s == nil {
		return
	}
	s.once.Do(func() {
		close(s.done)
		if !s.enabled {
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if final == "" {
			_, _ = fmt.Fprint(s.out, "\r\033[K")
			return
		}
		_, _ = fmt.Fprintf(s.out, "\r\033[K%s\n", final)
	})
}

func (s *Spinner) run() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	frame := 0
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			message := s.message
			_, _ = fmt.Fprintf(s.out, "\r\033[K%s %s", spinnerFrames[frame], message)
			s.mu.Unlock()
			frame = (frame + 1) % len(spinnerFrames)
		}
	}
}

func isTTYWriter(out io.Writer) bool {
	file, ok := out.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
