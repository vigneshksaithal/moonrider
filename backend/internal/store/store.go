package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	_ "github.com/jackc/pgx/v5/stdlib"
	"moonrider/backend/internal/domain"
)

// Save commits event deduplication and its resulting state together.
// A false result means the event was already processed.
type Repository interface {
	Load(context.Context) (domain.Snapshot, error)
	Save(context.Context, *domain.Reading, domain.Snapshot) (bool, error)
	Ping(context.Context) error
	Close() error
}

type Memory struct {
	mu       sync.Mutex
	snapshot domain.Snapshot
	ids      map[string]bool
}

func NewMemory() *Memory { return &Memory{snapshot: domain.Empty(), ids: map[string]bool{}} }
func (m *Memory) Load(context.Context) (domain.Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return domain.Clone(m.snapshot), nil
}
func (m *Memory) Save(_ context.Context, r *domain.Reading, s domain.Snapshot) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r != nil {
		if m.ids[r.EventID] {
			return false, nil
		}
		m.ids[r.EventID] = true
	}
	m.snapshot = domain.Clone(s)
	return true, nil
}
func (*Memory) Ping(context.Context) error { return nil }
func (*Memory) Close() error               { return nil }

type Postgres struct{ db *sql.DB }

func Open(ctx context.Context, url string) (*Postgres, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	p := &Postgres{db: db}
	if err = db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	// A single persisted snapshot keeps this single-tractor MVP small; telemetry
	// remains independently queryable. Use migrations for a larger deployment.
	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS telemetry (
		event_id text PRIMARY KEY, recorded_at timestamptz NOT NULL, payload jsonb NOT NULL);
		CREATE INDEX IF NOT EXISTS telemetry_recorded_at ON telemetry(recorded_at);
		CREATE TABLE IF NOT EXISTS tractor_state (id integer PRIMARY KEY CHECK(id=1), payload jsonb NOT NULL);`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("schema: %w", err)
	}
	return p, nil
}
func (p *Postgres) Load(ctx context.Context) (domain.Snapshot, error) {
	s := domain.Empty()
	var b []byte
	err := p.db.QueryRowContext(ctx, "SELECT payload FROM tractor_state WHERE id=1").Scan(&b)
	if err == sql.ErrNoRows {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	err = json.Unmarshal(b, &s)
	return s, err
}
func (p *Postgres) Save(ctx context.Context, r *domain.Reading, s domain.Snapshot) (bool, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	if r != nil {
		b, err := json.Marshal(r)
		if err != nil {
			return false, err
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO telemetry(event_id,recorded_at,payload) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, r.EventID, r.Timestamp, string(b))
		if err != nil {
			return false, err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return false, err
		}
		if n == 0 {
			return false, nil
		}
	}
	b, err := json.Marshal(s)
	if err != nil {
		return false, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO tractor_state(id,payload) VALUES(1,$1) ON CONFLICT(id) DO UPDATE SET payload=excluded.payload`, string(b))
	if err != nil {
		return false, err
	}
	return true, tx.Commit()
}
func (p *Postgres) Ping(ctx context.Context) error { return p.db.PingContext(ctx) }
func (p *Postgres) Close() error                   { return p.db.Close() }
