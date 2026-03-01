package util

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Shell represents a shell command executor
type Shell struct {
	mu sync.Mutex
}

// NewShell creates a new shell executor
func NewShell() *Shell {
	return &Shell{}
}

// CommandResult represents the result of a shell command execution
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Error    error
}

// Run executes a shell command and returns the result
func (s *Shell) Run(command string, args ...string) *CommandResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	cmd := exec.Command(command, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &CommandResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		result.Error = err
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
	}

	return result
}

// RunWithOutput executes a command and streams output to stdout/stderr
func (s *Shell) RunWithOutput(command string, args ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// RunWithPrefix executes a command and prefixes each line of output
func (s *Shell) RunWithPrefix(prefix, command string, args ...string) error {
	return s.RunWithFilteredPrefix(prefix, command, nil, args...)
}

// RunWithFilteredPrefix executes a command and prefixes each line of output,
// optionally filtering lines based on a filter function.
// If filterFunc is nil, all lines are printed (same as RunWithPrefix).
// If filterFunc returns true, the line is printed; otherwise it's discarded.
func (s *Shell) RunWithFilteredPrefix(prefix, command string, filterFunc func(string) bool, args ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cmd := exec.Command(command, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		StreamWithPrefixFiltered(stdout, prefix, false, filterFunc)
	}()

	go func() {
		defer wg.Done()
		StreamWithPrefixFiltered(stderr, prefix, true, filterFunc)
	}()

	wg.Wait()

	return cmd.Wait()
}

// StreamWithPrefix reads from a reader and writes to stdout/stderr with a prefix.
// This is a shared utility function used by both Shell and SSH.
// The function blocks until the reader is exhausted (EOF).
// When isError is true, output is written to stderr; otherwise to stdout.
// Lines are buffered and printed with the prefix, handling incomplete lines correctly.
func StreamWithPrefix(reader io.Reader, prefix string, isError bool) {
	StreamWithPrefixFiltered(reader, prefix, isError, nil)
}

// StreamWithPrefixFiltered reads from a reader and writes to stdout/stderr with a prefix,
// optionally filtering lines based on a filter function.
// If filterFunc is nil, all lines are printed (same as StreamWithPrefix).
// If filterFunc returns true, the line is printed; otherwise it's discarded.
func StreamWithPrefixFiltered(reader io.Reader, prefix string, isError bool, filterFunc func(string) bool) {
	buf := make([]byte, 4096)
	leftover := ""

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			text := leftover + string(buf[:n])
			lines := strings.Split(text, "\n")

			// Process all complete lines
			for i := 0; i < len(lines)-1; i++ {
				line := strings.TrimRight(lines[i], "\r")
				if line != "" {
					// Apply filter if provided
					if filterFunc == nil || filterFunc(line) {
						if isError {
							fmt.Fprintf(os.Stderr, "[%s] %s\n", prefix, line)
						} else {
							fmt.Printf("[%s] %s\n", prefix, line)
						}
					}
				}
			}

			// Keep the last incomplete line
			leftover = lines[len(lines)-1]
		}

		if err != nil {
			if err != io.EOF && leftover != "" {
				leftover = strings.TrimRight(leftover, "\r")
				if leftover != "" {
					// Apply filter if provided
					if filterFunc == nil || filterFunc(leftover) {
						if isError {
							fmt.Fprintf(os.Stderr, "[%s] %s\n", prefix, leftover)
						} else {
							fmt.Printf("[%s] %s\n", prefix, leftover)
						}
					}
				}
			}
			break
		}
	}
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
)

// clearCurrentLine clears the current terminal line to prevent overlap with spinners
func clearCurrentLine() {
	fmt.Print("\r\033[K")
}

// LogLine prints a log line with optional prefix
func LogLine(message string, prefix ...string) {
	if len(prefix) > 0 && prefix[0] != "" {
		fmt.Printf("[%s] %s\n", prefix[0], message)
	} else {
		fmt.Println(message)
	}
}

// LogSuccess prints a success message in green
func LogSuccess(message string, scope string) {
	clearCurrentLine()
	if scope != "" {
		fmt.Printf("%s[%s]%s %s\n", ColorGreen, scope, ColorReset, message)
	} else {
		fmt.Printf("%s%s%s\n", ColorGreen, message, ColorReset)
	}
}

// LogError prints an error message in red
func LogError(message string, scope string) {
	clearCurrentLine()
	if scope != "" {
		fmt.Printf("%s[%s]%s %s\n", ColorRed, scope, ColorReset, message)
	} else {
		fmt.Printf("%s%s%s\n", ColorRed, message, ColorReset)
	}
}

// LogWarning prints a warning message in yellow
func LogWarning(message string, scope string) {
	clearCurrentLine()
	if scope != "" {
		fmt.Printf("%s[%s]%s %s\n", ColorYellow, scope, ColorReset, message)
	} else {
		fmt.Printf("%s%s%s\n", ColorYellow, message, ColorReset)
	}
}

// LogInfo prints an info message in cyan
func LogInfo(message string, scope string) {
	clearCurrentLine()
	if scope != "" {
		fmt.Printf("%s[%s]%s %s\n", ColorCyan, scope, ColorReset, message)
	} else {
		fmt.Printf("%s%s%s\n", ColorCyan, message, ColorReset)
	}
}

// LogLineWithTimestamp prints a log line with timestamp
func LogLineWithTimestamp(message string, prefix ...string) {
	timestamp := time.Now().Format("15:04:05")
	if len(prefix) > 0 && prefix[0] != "" {
		fmt.Printf("[%s][%s] %s\n", timestamp, prefix[0], message)
	} else {
		fmt.Printf("[%s] %s\n", timestamp, message)
	}
}

// Retry executes a function with retry logic
func Retry(maxAttempts int, delay time.Duration, fn func() error) error {
	var err error
	for i := 0; i < maxAttempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}

		if i < maxAttempts-1 {
			time.Sleep(delay)
		}
	}
	return fmt.Errorf("max attempts (%d) reached: %w", maxAttempts, err)
}

// EscapeShellArg escapes special characters in a string to prevent shell injection
// This is a basic escaping function that removes potentially dangerous characters
// Used for sanitizing user input that will be passed to shell commands
func EscapeShellArg(s string) string {
	// Remove or replace characters that could be dangerous in shell context
	// We're extra cautious here since these values go into shell commands
	replacer := strings.NewReplacer(
		" ", "",
		"\t", "",
		"\n", "",
		"\r", "",
		"\"", "",
		"'", "",
		"`", "",
		"$", "",
		"\\", "",
		";", "",
		"&", "",
		"|", "",
		"<", "",
		">", "",
		"(", "",
		")", "",
	)
	return replacer.Replace(s)
}
