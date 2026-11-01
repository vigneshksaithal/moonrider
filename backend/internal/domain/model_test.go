package domain

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func TestValidation(t *testing.T) {
	now := time.Now()
	valid := Reading{EventID: "a", TractorID: TractorID, Sequence: 1, Timestamp: now, Battery: 70, Temperature: 50, Speed: 6}
	cases := []struct {
		name   string
		modify func(*Reading)
	}{
		{"wrong tractor", func(r *Reading) { r.TractorID = "unknown" }},
		{"missing id", func(r *Reading) { r.EventID = "" }},
		{"future sample", func(r *Reading) { r.Timestamp = now.Add(time.Minute) }},
		{"invalid battery", func(r *Reading) { r.Battery = 101 }},
		{"non finite", func(r *Reading) { r.Temperature = math.NaN() }},
		{"negative speed", func(r *Reading) { r.Speed = -1 }},
	}
	if err := valid.Validate(now); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := valid
			tc.modify(&r)
			if r.Validate(now) == nil {
				t.Fatal("accepted invalid reading")
			}
		})
	}
}

func TestInvalidRulesRejected(t *testing.T) {
	for _, value := range []float64{math.NaN(), math.Inf(1), -1000, 181} {
		r := DefaultRules()
		r.Hot = value
		if r.Validate() == nil {
			t.Fatalf("accepted hot threshold %v", value)
		}
	}
	r := DefaultRules()
	r.Low = 26
	if r.Validate() == nil {
		t.Fatal("accepted inverted hysteresis")
	}
}

func TestMissingOrNullSensorsRejected(t *testing.T) {
	for _, payload := range []string{
		`{"event_id":"a","tractor_id":"MR-001","sequence":1,"timestamp":"2026-11-01T00:00:00Z"}`,
		`{"event_id":"a","tractor_id":"MR-001","sequence":1,"timestamp":"2026-11-01T00:00:00Z","battery_pct":null,"motor_temp_c":50,"speed_kmh":6}`,
	} {
		var r Reading
		if err := json.Unmarshal([]byte(payload), &r); err == nil {
			t.Fatal("missing sensor accepted as a zero value")
		}
	}
}
