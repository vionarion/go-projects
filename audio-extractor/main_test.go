package main

import (
	"testing"
)

func TestExtract(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		output    string
		track     int
		expectErr bool
	}{
		{
			name:      "valid inputs",
			input:     "video.mkv",
			output:    "audio.mp3",
			track:     0,
			expectErr: false,
		},
		{
			name:      "missing input",
			input:     "",
			output:    "audio.mp3",
			track:     0,
			expectErr: true,
		},
		{
			name:      "missing output",
			input:     "video.mkv",
			output:    "",
			track:     0,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := extract(tt.input, tt.output, tt.track)
			if (err != nil) != tt.expectErr {
				t.Errorf("extract() error = %v, expectErr = %v", err, tt.expectErr)
			}
		})
	}
}

func TestCut(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		output    string
		start     string
		end       string
		expectErr bool
	}{
		{
			name:      "valid inputs",
			input:     "audio.mp3",
			output:    "cut.mp3",
			start:     "00:01:05",
			end:       "00:46:00",
			expectErr: false,
		},
		{
			name:      "missing input",
			input:     "",
			output:    "cut.mp3",
			start:     "00:01:05",
			end:       "00:46:00",
			expectErr: true,
		},
		{
			name:      "missing output",
			input:     "audio.mp3",
			output:    "",
			start:     "00:01:05",
			end:       "00:46:00",
			expectErr: true,
		},
		{
			name:      "missing end time",
			input:     "audio.mp3",
			output:    "cut.mp3",
			start:     "00:01:05",
			end:       "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cut(tt.input, tt.output, tt.start, tt.end)
			if (err != nil) != tt.expectErr {
				t.Errorf("cut() error = %v, expectErr = %v", err, tt.expectErr)
			}
		})
	}
}
