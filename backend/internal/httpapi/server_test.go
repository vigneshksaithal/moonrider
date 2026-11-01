package httpapi

import (
	"context"
	"github.com/gorilla/websocket"
	"moonrider/backend/internal/domain"
	"moonrider/backend/internal/engine"
	"moonrider/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRESTAndLiveShareCommittedState(t *testing.T) {
	repo := store.NewMemory()
	e, _ := engine.New(context.Background(), repo, domain.DefaultRules(), true)
	api := &Server{Engine: e, Repo: repo, MQTTReady: func() bool { return true }, Origins: []string{"http://localhost:5173"}}
	server := httptest.NewServer(api.Handler())
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/api/live", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var first domain.Snapshot
	if err = conn.ReadJSON(&first); err != nil {
		t.Fatal(err)
	}
	if first.Version != 0 {
		t.Fatal("missing initial snapshot")
	}
	now := time.Now().UTC()
	r := domain.Reading{EventID: "a", TractorID: domain.TractorID, Sequence: 1, Timestamp: now, Battery: 12, Temperature: 50, Speed: 6}
	if err = e.Process(context.Background(), r, now); err != nil {
		t.Fatal(err)
	}
	var live domain.Snapshot
	if err = conn.ReadJSON(&live); err != nil {
		t.Fatal(err)
	}
	if live.Version != 1 || len(live.Alerts) != 1 {
		t.Fatal("live update was not committed state")
	}
	for _, path := range []string{"/healthz", "/readyz", "/api/tractors/MR-001", "/api/alerts"} {
		response, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != 200 {
			t.Fatalf("%s: %d", path, response.StatusCode)
		}
	}
	req, _ := http.NewRequest("GET", server.URL+"/api/alerts", nil)
	req.Header.Set("Origin", "https://evil.example")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 403 {
		t.Fatal("unapproved origin accepted")
	}
}
func TestReadinessReportsUnavailableMQTT(t *testing.T) {
	repo := store.NewMemory()
	e, _ := engine.New(context.Background(), repo, domain.DefaultRules(), false)
	api := &Server{Engine: e, Repo: repo, MQTTReady: func() bool { return false }}
	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/readyz", nil))
	if w.Code != 503 {
		t.Fatal("readiness hid missing subscription")
	}
}
