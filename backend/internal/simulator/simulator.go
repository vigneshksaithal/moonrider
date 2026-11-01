package simulator

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math"
	"time"

	"moonrider/backend/internal/domain"
)

type Generator struct {
	session  string
	sequence uint64
}

func New() *Generator {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return &Generator{session: hex.EncodeToString(b)}
}
func (g *Generator) Next(scenario string, now time.Time) domain.Reading {
	g.sequence++
	r := domain.Reading{
		TractorID:   domain.TractorID,
		EventID:     g.session + "-" + now.Format("150405.000000000"),
		Sequence:    g.sequence,
		Timestamp:   now.UTC(),
		Battery:     76 + math.Sin(float64(g.sequence)/20),
		Temperature: 51 + 3*math.Sin(float64(g.sequence)/5),
		Speed:       6 + math.Sin(float64(g.sequence)/3),
	}
	if scenario == "overheat" {
		r.Temperature = 88 + math.Sin(float64(g.sequence))
	}
	if scenario == "low-battery" {
		r.Battery = 17
	}
	return r
}

// Cycle is also used by --demo. Silence models a lost device uplink; replayed
// events retain their original IDs and timestamps, just as redeliveries do.
func Cycle(second int) (scenario string, send bool, replay bool) {
	t := second % 80
	switch {
	case t < 10:
		return "normal", true, false
	case t < 23:
		return "overheat", true, false
	case t < 34:
		return "normal", true, false
	case t < 44:
		return "low-battery", true, false
	case t < 50:
		return "normal", true, false
	case t < 65:
		return "normal", false, false
	case t == 65:
		return "normal", true, true
	default:
		return "normal", true, false
	}
}

// Run is shared by the MQTT simulator and the in-memory demo. Only publish
// changes: the real path sends MQTT; the demo enqueues the same reading locally.
func Run(ctx context.Context, scenario string, duplicate bool, publish func(domain.Reading) error) error {
	switch scenario {
	case "normal", "overheat", "low-battery", "cycle":
	default:
		return fmt.Errorf("unknown scenario %q", scenario)
	}
	generator := New()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var last *domain.Reading
	emit := func(r domain.Reading) {
		if err := publish(r); err != nil && ctx.Err() == nil {
			slog.Warn("telemetry publish failed", "error", err)
		}
	}
	for second := 0; ; second++ {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			mode, send, replay := scenario, true, false
			if scenario == "cycle" {
				mode, send, replay = Cycle(second)
			}
			if !send {
				continue
			}
			if replay && last != nil {
				emit(*last)
			}
			reading := generator.Next(mode, now)
			last = &reading
			emit(reading)
			if duplicate {
				emit(reading)
			}
		}
	}
}
