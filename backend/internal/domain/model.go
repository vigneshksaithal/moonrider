package domain

import (
	"encoding/json"
	"fmt"
	"math"
	"time"
)

const TractorID = "MR-001"

type Reading struct {
	EventID string `json:"event_id"`
	TractorID string `json:"tractor_id"`
	Sequence uint64 `json:"sequence"`
	Timestamp time.Time `json:"timestamp"`
	Battery float64 `json:"battery_pct"`
	Temperature float64 `json:"motor_temp_c"`
	Speed float64 `json:"speed_kmh"`
}

func (r Reading) Validate(now time.Time) error {
	if r.TractorID != TractorID || len(r.EventID) == 0 || len(r.EventID) > 128 || r.Sequence == 0 {
		return fmt.Errorf("invalid tractor, event ID, or sequence")
	}
	if r.Timestamp.IsZero() || r.Timestamp.After(now.Add(5*time.Second)) {
		return fmt.Errorf("invalid or future timestamp")
	}
	if !finite(r.Battery) || !finite(r.Temperature) || !finite(r.Speed) || r.Battery < 0 || r.Battery > 100 || r.Temperature < -40 || r.Temperature > 180 || r.Speed < 0 || r.Speed > 80 {
		return fmt.Errorf("reading outside supported sensor ranges")
	}
	return nil
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

type Alert struct {
	ID string `json:"id"`
	Kind string `json:"kind"`
	Title string `json:"title"`
	Message string `json:"message"`
	OpenedAt time.Time `json:"opened_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

type Rules struct {
	Hot float64 `json:"hot_c"`
	Cool float64 `json:"cool_c"`
	Low float64 `json:"low_battery_pct"`
	Recovered float64 `json:"recovered_battery_pct"`
	Sustain time.Duration `json:"-"`
	Offline time.Duration `json:"-"`
}

func DefaultRules() Rules {
	return Rules{Hot:80, Cool:75, Low:20, Recovered:25, Sustain:5*time.Second, Offline:10*time.Second}
}

type Snapshot struct {
	Version uint64 `json:"version"`
	TractorID string `json:"tractor_id"`
	Reading *Reading `json:"reading"`
	Online bool `json:"online"`
	LastSeen *time.Time `json:"last_seen"`
	Alerts []Alert `json:"alerts"`
	Demo bool `json:"demo"`
	// Persist the window state so restarting does not lose an active episode.
	HotSince *time.Time `json:"hot_since,omitempty"`
	CoolSince *time.Time `json:"cool_since,omitempty"`
}

func Empty() Snapshot { return Snapshot{TractorID:TractorID, Alerts:[]Alert{}} }
func Clone(s Snapshot) Snapshot {
	b, _ := json.Marshal(s)
	var out Snapshot
	_ = json.Unmarshal(b, &out)
	return out
}
