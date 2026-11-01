package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"moonrider/backend/internal/domain"
	"net/url"
	"os"
	"testing"
	"time"
)

// CI provides a disposable PostgreSQL service. Each test uses an isolated schema;
// it never truncates or drops user tables. Local runs skip when no DB is supplied.
func TestPostgresAtomicDedupAndRestore(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
	if _, err = admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	repo, err := Open(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	now := time.Now().UTC()
	r := domain.Reading{EventID: "test-event", TractorID: domain.TractorID, Sequence: 1, Timestamp: now, Battery: 10, Temperature: 50, Speed: 6}
	s := domain.Empty()
	s.Version = 1
	s.Online = true
	s.Reading = &r
	s.LastSeen = &now
	s.Alerts = []domain.Alert{{ID: "low-1", Kind: "low_battery", Title: "Low battery", Message: "Charge", OpenedAt: now}}
	if ok, err := repo.Save(ctx, &r, s); err != nil || !ok {
		t.Fatalf("first commit: %v %v", ok, err)
	}
	s.Version = 2
	s.Alerts = nil
	if ok, err := repo.Save(ctx, &r, s); err != nil || ok {
		t.Fatalf("dedup commit: %v %v", ok, err)
	}
	restored, err := repo.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Version != 1 || len(restored.Alerts) != 1 {
		t.Fatal("duplicate overwrote persisted state")
	}
	// Force a state serialization failure after telemetry insertion. Both writes
	// must roll back, leaving the event retryable.
	s.Version = 3
	s.Reading = &domain.Reading{Battery: 0}
	invalid := r
	invalid.EventID = "retryable"
	// A deferred DB constraint makes the second write fail in the same transaction.
	if _, err = admin.ExecContext(ctx, "ALTER TABLE "+schema+".tractor_state ADD CONSTRAINT reject_three CHECK ((payload->>'version')::int != 3)"); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Save(ctx, &invalid, s); err == nil {
		t.Fatal("expected transaction failure")
	}
	s.Version = 2
	s.Reading = &invalid
	if ok, err := repo.Save(ctx, &invalid, s); err != nil || !ok {
		t.Fatalf("failed transaction consumed event ID: %v %v", ok, err)
	}
	var n int
	if err = repo.db.QueryRowContext(ctx, "SELECT count(*) FROM telemetry").Scan(&n); err != nil || n != 2 {
		t.Fatalf("unexpected persisted events: %d %v", n, err)
	}
	b, _ := json.Marshal(restored)
	if len(b) == 0 {
		t.Fatal("missing persisted snapshot")
	}
}
