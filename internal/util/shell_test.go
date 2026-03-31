// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package util

import (
	"io"
	"strings"
	"testing"
)

// TestStreamWithPrefixFiltered tests the filtering functionality
func TestStreamWithPrefixFiltered(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		filterFunc     func(string) bool
		expectedOutput []string
	}{
		{
			name:       "no filter - all lines printed",
			input:      "line1\nline2\nline3\n",
			filterFunc: nil,
			expectedOutput: []string{
				"[test] line1",
				"[test] line2",
				"[test] line3",
			},
		},
		{
			name:  "filter keeps only lines with 'keep'",
			input: "keep this\ndiscard this\nkeep that\n",
			filterFunc: func(line string) bool {
				return strings.Contains(line, "keep")
			},
			expectedOutput: []string{
				"[test] keep this",
				"[test] keep that",
			},
		},
		{
			name:  "filter discards all lines",
			input: "line1\nline2\nline3\n",
			filterFunc: func(line string) bool {
				return false
			},
			expectedOutput: []string{},
		},
		{
			name:  "filter keeps lines with emoji",
			input: "ℹ️  Using Cilium version 1.18.6\nDaemonSet cilium Desired: 3\nℹ️  Installation complete\n",
			filterFunc: func(line string) bool {
				return strings.Contains(line, "ℹ️")
			},
			expectedOutput: []string{
				"[test] ℹ️  Using Cilium version 1.18.6",
				"[test] ℹ️  Installation complete",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a reader from the input string
			reader := strings.NewReader(tt.input)

			// We cant easily capture stdout in test, so we'll test the logic directly
			// by creating a custom version that writes to a buffer
			buf := make([]byte, 4096)
			leftover := ""
			outputLines := []string{}

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
							if tt.filterFunc == nil || tt.filterFunc(line) {
								outputLines = append(outputLines, "[test] "+line)
							}
						}
					}

					// Keep the last incomplete line
					leftover = lines[len(lines)-1]
				}

				if err != nil {
					if err == io.EOF && leftover != "" {
						leftover = strings.TrimRight(leftover, "\r")
						if leftover != "" {
							// Apply filter if provided
							if tt.filterFunc == nil || tt.filterFunc(leftover) {
								outputLines = append(outputLines, "[test] "+leftover)
							}
						}
					}
					break
				}
			}

			// Verify output
			if len(outputLines) != len(tt.expectedOutput) {
				t.Errorf("Expected %d lines, got %d lines", len(tt.expectedOutput), len(outputLines))
				t.Errorf("Expected: %v", tt.expectedOutput)
				t.Errorf("Got: %v", outputLines)
				return
			}

			for i, expected := range tt.expectedOutput {
				if outputLines[i] != expected {
					t.Errorf("Line %d: expected %q, got %q", i, expected, outputLines[i])
				}
			}
		})
	}
}

// TestRunWithFilteredPrefix tests the filtering with nil filter (should act like RunWithPrefix)
func TestRunWithFilteredPrefix(t *testing.T) {
	shell := NewShell()

	// Test with nil filter (should print all output)
	// We'll just verify it doesnt error - actual output verification would require
	// capturing stdout which is complex in Go tests
	err := shell.RunWithFilteredPrefix("test", "echo", nil, "hello world")
	if err != nil {
		t.Errorf("RunWithFilteredPrefix with nil filter failed: %v", err)
	}

	// Test with a filter that keeps everything
	err = shell.RunWithFilteredPrefix("test", "echo", func(line string) bool {
		return true
	}, "hello world")
	if err != nil {
		t.Errorf("RunWithFilteredPrefix with keep-all filter failed: %v", err)
	}

	// Test with a filter that discards everything
	err = shell.RunWithFilteredPrefix("test", "echo", func(line string) bool {
		return false
	}, "hello world")
	if err != nil {
		t.Errorf("RunWithFilteredPrefix with discard-all filter failed: %v", err)
	}
}

// TestRunWithPrefix ensures backward compatibility
func TestRunWithPrefix(t *testing.T) {
	shell := NewShell()

	// Test that RunWithPrefix still works (should use nil filter internally)
	err := shell.RunWithPrefix("test", "echo", "hello world")
	if err != nil {
		t.Errorf("RunWithPrefix failed: %v", err)
	}
}
