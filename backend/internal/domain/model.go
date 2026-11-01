package domain

import (
	"encoding/json"
	"fmt"
	"math"
	"time"
)

const TractorID = "MR-001"

type Reading struct {
	EventID     string    `json:"event_id"`
	TractorID   string    `json:"tractor_id"`
	Sequence    uint64    `json:"sequence"`
	Timestamp   time.Time `json:"timestamp"`
	Battery     float64   `json:"battery_pct"`
	Temperature float64   `json:"motor_temp_c"`
	Speed       float64   `json:"speed_kmh"`
}

func (r *Reading) UnmarshalJSON(data []byte) error {
	// Missing sensors must not silently turn into valid-looking zero readings.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for _, key := range []string{"event_id", "tractor_id", "sequence", "timestamp", "battery_pct", "motor_temp_c", "speed_kmh"} {
		value, ok := fields[key]
		if !ok || string(value) == "null" {
			return fmt.Errorf("missing telemetry field %s", key)
		}
	}
	type plain Reading
	return json.Unmarshal(data, (*plain)(r))
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
	ID         string     `json:"id"`
	Kind       string     `json:"kind"`
	Title      string     `json:"title"`
	Message    string     `json:"message"`
	OpenedAt   time.Time  `json:"opened_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

type Rules struct {
	Hot       float64       `json:"hot_c"`
	Cool      float64       `json:"cool_c"`
	Low       float64       `json:"low_battery_pct"`
	Recovered float64       `json:"recovered_battery_pct"`
	Sustain   time.Duration `json:"-"`
	Offline   time.Duration `json:"-"`
}

// DefaultRules is the single place to tune the demo thresholds and timing.
func DefaultRules() Rules {
	return Rules{Hot: 80, Cool: 75, Low: 20, Recovered: 25, Sustain: 5 * time.Second, Offline: 10 * time.Second}
}

func (r Rules) Validate() error {
	if !finite(r.Hot) || !finite(r.Cool) || !finite(r.Low) || !finite(r.Recovered) ||
		r.Cool >= r.Hot || r.Cool < -40 || r.Hot > 180 || r.Low < 0 || r.Low >= r.Recovered || r.Recovered > 100 ||
		r.Sustain <= 0 || r.Offline <= 0 {
		return fmt.Errorf("invalid fault rules: use finite sensor thresholds, recovery hysteresis, and positive durations")
	}
	return nil
}

type Snapshot struct {
	Version   uint64     `json:"version"`
	TractorID string     `json:"tractor_id"`
	Reading   *Reading   `json:"reading"`
	Online    bool       `json:"online"`
	LastSeen  *time.Time `json:"last_seen"`
	Alerts    []Alert    `json:"alerts"`
	Demo      bool       `json:"demo"`
	Rules     Rules      `json:"rules"`
	// Persist the window state so restarting does not lose an active episode.
	HotSince  *time.Time `json:"hot_since,omitempty"`
	CoolSince *time.Time `json:"cool_since,omitempty"`
}

func Empty() Snapshot { return Snapshot{TractorID: TractorID, Alerts: []Alert{}} }
func Clone(s Snapshot) Snapshot {
	out := s
	if s.Reading != nil {
		r := *s.Reading
		out.Reading = &r
	}
	out.LastSeen = copyTime(s.LastSeen)
	out.HotSince = copyTime(s.HotSince)
	out.CoolSince = copyTime(s.CoolSince)
	out.Alerts = append([]Alert{}, s.Alerts...)
	for i := range out.Alerts {
		out.Alerts[i].ResolvedAt = copyTime(s.Alerts[i].ResolvedAt)
	}
	return out
}

func copyTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	value := *t
	return &value
}
