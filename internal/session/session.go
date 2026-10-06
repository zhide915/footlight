package session

import (
	"encoding/json"
	"io"
)

// Session mirrors the JSON that Claude Code feeds footlight on stdin. Pointer
// fields are optional: a nil pointer means the field was absent from the
// payload, and the widget that reads it renders nothing.
type Session struct {
	Model         Model          `json:"model"`
	Workspace     Workspace      `json:"workspace"`
	ContextWindow *ContextWindow `json:"context_window"`
	Cost          *Cost          `json:"cost"`
	Effort        *Effort        `json:"effort"`
	Thinking      *Thinking      `json:"thinking"`
	RateLimits    *RateLimits    `json:"rate_limits"`
	SessionName   *string        `json:"session_name"`
	Version       string         `json:"version"`
}

type Model struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type Workspace struct {
	CurrentDir string `json:"current_dir"`
	ProjectDir string `json:"project_dir"`
}

type ContextWindow struct {
	UsedPercentage    *float64 `json:"used_percentage"`
	ContextWindowSize int      `json:"context_window_size"`
}

type Cost struct {
	TotalCostUSD      float64 `json:"total_cost_usd"`
	TotalDurationMS   int64   `json:"total_duration_ms"`
	TotalLinesAdded   int     `json:"total_lines_added"`
	TotalLinesRemoved int     `json:"total_lines_removed"`
}

type Effort struct {
	Level string `json:"level"`
}

type Thinking struct {
	Enabled bool `json:"enabled"`
}

type RateLimits struct {
	FiveHour RateWindow `json:"five_hour"`
	SevenDay RateWindow `json:"seven_day"`
}

type RateWindow struct {
	UsedPercentage float64 `json:"used_percentage"`
	ResetsAt       int64   `json:"resets_at"`
}

func Parse(r io.Reader) (*Session, error) {
	var s Session
	if err := json.NewDecoder(r).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}
