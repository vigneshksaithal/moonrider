package mqttclient

import (
	"context"
	"encoding/json"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	broker "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"moonrider/backend/internal/domain"
	"moonrider/backend/internal/engine"
	"moonrider/backend/internal/store"
	"testing"
	"time"
)

func TestMQTTPipelineRejectsInvalidAndDeduplicates(t *testing.T) {
	b := broker.New(nil)
	if err := b.AddHook(new(auth.AllowHook), nil); err != nil {
		t.Fatal(err)
	}
	tcp := listeners.NewTCP(listeners.Config{ID: "test", Address: "127.0.0.1:0"})
	if err := b.AddListener(tcp); err != nil {
		t.Fatal(err)
	}
	if err := b.Serve(); err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	t.Setenv("MQTT_BROKER", "tcp://"+tcp.Address())
	t.Setenv("MQTT_CLIENT_ID", "test-backend")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	q := make(chan engine.Delivery, 64)
	e, _ := engine.New(ctx, store.NewMemory(), domain.DefaultRules(), false)
	workerDone := make(chan struct{})
	go func() { defer close(workerDone); e.Run(ctx, q) }()
	defer func() { cancel(); <-workerDone }()
	sub, err := Subscribe(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Close()
	deadline := time.Now().Add(3 * time.Second)
	for !sub.Ready() {
		if time.Now().After(deadline) {
			t.Fatal("subscription never ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
	opts, err := Options("test-simulator")
	if err != nil {
		t.Fatal(err)
	}
	c := mqtt.NewClient(opts)
	if err = Wait(ctx, c.Connect()); err != nil {
		t.Fatal(err)
	}
	defer c.Disconnect(100)
	publish := func(payload []byte) {
		t.Helper()
		if err := Wait(ctx, c.Publish(Topic, 1, false, payload)); err != nil {
			t.Fatal(err)
		}
	}
	publish([]byte(`{"tractor_id":"wrong"}`))
	now := time.Now().UTC()
	r := domain.Reading{EventID: "same-event", TractorID: domain.TractorID, Sequence: 1, Timestamp: now, Battery: 12, Temperature: 50, Speed: 6}
	payload, _ := json.Marshal(r)
	publish(payload)
	publish(payload)
	deadline = time.Now().Add(3 * time.Second)
	for e.Snapshot().Version < 1 {
		if time.Now().After(deadline) {
			t.Fatal("MQTT reading never reached processor")
		}
		time.Sleep(10 * time.Millisecond)
	}
	// A subsequent unique event acts as a barrier behind the duplicate callback.
	r.EventID = "next-event"
	r.Sequence = 2
	r.Timestamp = now.Add(time.Millisecond)
	payload, _ = json.Marshal(r)
	publish(payload)
	deadline = time.Now().Add(3 * time.Second)
	for e.Snapshot().Version < 2 {
		if time.Now().After(deadline) {
			t.Fatal("subsequent event not processed")
		}
		time.Sleep(10 * time.Millisecond)
	}
	s := e.Snapshot()
	if s.Version != 2 || len(s.Alerts) != 1 {
		t.Fatalf("duplicate MQTT delivery created duplicate state: %+v", s)
	}
}
