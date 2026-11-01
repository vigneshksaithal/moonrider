package engine

import (
	"context"
	"errors"
	"fmt"
	"moonrider/backend/internal/domain"
	"moonrider/backend/internal/store"
	"testing"
	"time"
)

var base = time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

func reading(second int, temp, battery float64) domain.Reading {
	return domain.Reading{EventID: fmt.Sprintf("event-%d", second), TractorID: domain.TractorID, Sequence: uint64(second + 1), Timestamp: base.Add(time.Duration(second) * time.Second), Temperature: temp, Battery: battery, Speed: 6}
}
func setup(t *testing.T) *Engine {
	t.Helper()
	e, err := New(context.Background(), store.NewMemory(), domain.DefaultRules(), false)
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func send(t *testing.T, e *Engine, second int, temp, battery float64) {
	t.Helper()
	r := reading(second, temp, battery)
	if err := e.Process(context.Background(), r, r.Timestamp); err != nil {
		t.Fatal(err)
	}
}
func active(e *Engine, kind string) int {
	n := 0
	for _, a := range e.Snapshot().Alerts {
		if a.Kind == kind && a.ResolvedAt == nil {
			n++
		}
	}
	return n
}

func TestOverheatSustainHysteresisAndNewEpisode(t *testing.T) {
	e := setup(t)
	for i := 0; i < 5; i++ {
		send(t, e, i, 88, 76)
	}
	if active(e, "overheating") != 0 {
		t.Fatal("alert before sustained window")
	}
	for i := 5; i < 10; i++ {
		send(t, e, i, 88, 76)
	}
	if len(e.Snapshot().Alerts) != 1 {
		t.Fatal("repeated hot readings must create one alert")
	}
	send(t, e, 10, 77, 76)
	if active(e, "overheating") != 1 {
		t.Fatal("hysteresis band must preserve alert")
	}
	for i := 11; i <= 16; i++ {
		send(t, e, i, 50, 76)
	}
	if active(e, "overheating") != 0 {
		t.Fatal("sustained cooling must resolve")
	}
	for i := 17; i <= 22; i++ {
		send(t, e, i, 88, 76)
	}
	if len(e.Snapshot().Alerts) != 2 {
		t.Fatal("new fault episode must create a new alert")
	}
}
func TestLowBatteryRecovery(t *testing.T) {
	e := setup(t)
	send(t, e, 0, 50, 19)
	send(t, e, 1, 50, 18)
	send(t, e, 2, 50, 23)
	if active(e, "low_battery") != 1 || len(e.Snapshot().Alerts) != 1 {
		t.Fatal("low battery episode duplicated or resolved too soon")
	}
	send(t, e, 3, 50, 26)
	if active(e, "low_battery") != 0 {
		t.Fatal("battery did not recover")
	}
}
func TestDuplicateAndOldEventsCannotRestoreOnline(t *testing.T) {
	e := setup(t)
	send(t, e, 0, 50, 76)
	if err := e.CheckOffline(context.Background(), base.Add(11*time.Second)); err != nil {
		t.Fatal(err)
	}
	if e.Snapshot().Online || active(e, "offline") != 1 {
		t.Fatal("missing offline transition")
	}
	version := e.Snapshot().Version
	for i := 0; i < 3; i++ {
		if err := e.Process(context.Background(), reading(0, 50, 76), base.Add(12*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	if e.Snapshot().Online || e.Snapshot().Version != version {
		t.Fatal("replay changed health state")
	}
	send(t, e, 13, 50, 76)
	if !e.Snapshot().Online || active(e, "offline") != 0 || len(e.Snapshot().Alerts) != 1 {
		t.Fatal("fresh event must resolve the existing offline episode")
	}
}
func TestOutOfOrderAndReusedID(t *testing.T) {
	e := setup(t)
	send(t, e, 2, 50, 76)
	version := e.Snapshot().Version
	old := reading(1, 88, 10)
	if err := e.Process(context.Background(), old, base.Add(3*time.Second)); err != nil {
		t.Fatal(err)
	}
	reuse := reading(3, 88, 10)
	reuse.EventID = "event-2"
	if err := e.Process(context.Background(), reuse, reuse.Timestamp); err != nil {
		t.Fatal(err)
	}
	if e.Snapshot().Version != version || e.Snapshot().Reading.Battery != 76 {
		t.Fatal("old or duplicate event modified state")
	}
}
func TestOfflineGapResetsSustainedWindow(t *testing.T) {
	e := setup(t)
	send(t, e, 0, 88, 76)
	send(t, e, 1, 88, 76)
	if err := e.CheckOffline(context.Background(), base.Add(12*time.Second)); err != nil {
		t.Fatal(err)
	}
	send(t, e, 13, 88, 76)
	if active(e, "overheating") != 0 {
		t.Fatal("missing readings must not satisfy sustained heat")
	}
}
func TestSnapshotIsDetachedAndSlowSubscriberGetsLatest(t *testing.T) {
	e := setup(t)
	ch, done := e.Subscribe()
	defer done()
	for i := 0; i < 20; i++ {
		send(t, e, i, 50, 76)
	}
	latest := <-ch
	if latest.Version != 20 {
		t.Fatal("slow client did not receive latest snapshot")
	}
	latest.Reading.Battery = 0
	if e.Snapshot().Reading.Battery == 0 {
		t.Fatal("caller mutated engine state")
	}
}

type failingStore struct {
	*store.Memory
	fail bool
}

func (f *failingStore) Save(ctx context.Context, r *domain.Reading, s domain.Snapshot) (bool, error) {
	if f.fail {
		return false, errors.New("storage unavailable")
	}
	return f.Memory.Save(ctx, r, s)
}
func TestFailedPersistenceDoesNotPublishOrAdvanceState(t *testing.T) {
	repo := &failingStore{Memory: store.NewMemory(), fail: true}
	e, err := New(context.Background(), repo, domain.DefaultRules(), false)
	if err != nil {
		t.Fatal(err)
	}
	r := reading(0, 50, 10)
	if e.Process(context.Background(), r, base) == nil {
		t.Fatal("expected storage error")
	}
	if e.Snapshot().Version != 0 || len(e.Snapshot().Alerts) != 0 {
		t.Fatal("uncommitted state escaped")
	}
	repo.fail = false
	if err := e.Process(context.Background(), r, base); err != nil {
		t.Fatal(err)
	}
	if e.Snapshot().Version != 1 || active(e, "low_battery") != 1 {
		t.Fatal("retry did not commit")
	}
}
func TestRestartPreservesAlertAndDeduplication(t *testing.T) {
	repo := store.NewMemory()
	e, _ := New(context.Background(), repo, domain.DefaultRules(), false)
	send(t, e, 0, 50, 10)
	restored, err := New(context.Background(), repo, domain.DefaultRules(), false)
	if err != nil {
		t.Fatal(err)
	}
	send(t, restored, 0, 50, 10)
	send(t, restored, 1, 50, 10)
	if len(restored.Snapshot().Alerts) != 1 {
		t.Fatal("restart duplicated active alert")
	}
}
