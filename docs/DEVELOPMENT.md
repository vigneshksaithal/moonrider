# Development guide

## Run modes

| Mode | Command | Dependencies | Persistence |
| --- | --- | --- | --- |
| Quick visual demo | `go run ./cmd/api --demo` from `backend/` | Go | In-memory, lost on restart; bypasses MQTT |
| Full local pipeline | `docker compose up --build`, then simulator | Docker, Go | PostgreSQL volume; local MQTT broker |
| AWS integration | Go backend + simulator configured for AWS | Go, local PostgreSQL, your IoT resources | PostgreSQL; AWS MQTT transport |

For host-based Go development:

```sh
# Project root
docker compose up -d postgres mqtt
cp .env.example .env
source .env
cd backend
go run ./cmd/api
```

In a second terminal, source `.env` again from the root, change to `backend/`, and run:

```sh
go run ./cmd/simulator --scenario cycle --duplicate
```

Keep MQTT client IDs distinct when overriding the defaults. Start the backend first so it has a subscription before the simulator starts. A short publish-before-subscribe gap is not automatically recovered on a brand-new subscriber session.

From the root in a third terminal:

```sh
flutter pub get
flutter run -d chrome --web-port 5173
```

For a browserless preview use `flutter run -d web-server --web-hostname 127.0.0.1 --web-port 5173`, then open `http://127.0.0.1:5173`.

The API defaults to `http://127.0.0.1:8080`. Override at build time with `--dart-define=API_BASE_URL=https://YOUR_BACKEND`. Android emulator loopback is `http://10.0.2.2:8080`; the iOS simulator can use `127.0.0.1`. A physical phone requires your computer's LAN address, the Go server bound to that interface, and local network permission/firewall access. Do not expose the unauthenticated API to the public internet.

## Endpoints

| Endpoint | Response |
| --- | --- |
| `GET /healthz` | Process liveness |
| `GET /readyz` | 200 if DB reachable and MQTT subscribed; 503 otherwise. Demo has no MQTT dependency. |
| `GET /api/tractors/MR-001` | Current complete snapshot |
| `GET /api/alerts` | Up to 100 active/resolved alert episodes, oldest first |
| `GET /api/live` | WebSocket: immediate complete snapshot, then committed changes and five-second heartbeat snapshots |

The app orders snapshots within a live session by version. A fresh subscription always receives state under the same engine lock used to publish updates, so reconnect does not need an event replay cursor.

## Telemetry contract

Topic: `moonrider/tractors/MR-001/telemetry`. MQTT 3.1.1, QoS 1, non-retained publishes.

```json
{
  "event_id": "session-unique-event",
  "tractor_id": "MR-001",
  "sequence": 1,
  "timestamp": "2026-11-01T08:00:00Z",
  "battery_pct": 76.5,
  "motor_temp_c": 51.2,
  "speed_kmh": 6.4
}
```

Use a current UTC timestamp when publishing manually. This example timestamp is illustrative and becomes stale. An event ID is unique per logical reading and stays unchanged during replay. Each new simulator process has a new random session prefix. Sequence is positive and local to that simulator session; current ordering uses sample timestamps.

Payloads over 4096 bytes, wrong tractor IDs, empty/oversized event IDs, zero sequences, missing timestamps, more than five seconds of future clock skew, NaN/Infinity, and unsupported sensor ranges are rejected. Supported demo ranges: battery 0–100%, temperature −40–180°C, speed 0–80 km/h. Stale/equal/older readings do not update state. Retained telemetry is not used as a fresh heartbeat.

Snapshot fields: `version`, `tractor_id`, `reading`, `online`, `last_seen`, `alerts`, `demo`, configured `rules` thresholds, and optional persisted rule-window timestamps. Each alert has `id`, `kind`, `title`, `message`, `opened_at`, and an optional `resolved_at`. Absence of `resolved_at` means active. Snapshot history is bounded; no pagination or long-term alert archive is implemented.

## Configuration

Environment variables are explicitly read by Go. `.env` is a convenience shell file; it is **not loaded automatically**.

| Variable | Default / requirement |
| --- | --- |
| `DATABASE_URL` | Required outside `--demo`; pgx PostgreSQL connection URL |
| `HTTP_ADDR` | `127.0.0.1:8080` |
| `ALLOWED_ORIGINS` | `http://localhost:5173,http://127.0.0.1:5173` |
| `MQTT_BROKER` | `tcp://127.0.0.1:1883`; AWS uses `ssl://ATS_ENDPOINT:8883` |
| `MQTT_CLIENT_ID` | Backend `moonrider-backend`; simulator `moonrider-simulator` |
| `MQTT_CERT`, `MQTT_KEY`, `MQTT_CA` | Required for TLS broker connections; absolute file paths recommended |
| Flutter `API_BASE_URL` | Build-time define; `http://127.0.0.1:8080` |

Fault thresholds and timing are defined together in `backend/internal/domain/model.go` → `DefaultRules()`. Edit that function and restart Go to change them; threshold environment overrides were removed to keep setup simple.

Simulator flags: `--scenario normal`, `--scenario overheat`, `--scenario low-battery`, or `--scenario cycle` (default). Add `--duplicate` to publish each generated event twice. The cycle's 15-second silent interval pauses telemetry publishing rather than physically disconnecting TCP.

## Tests and checks

```sh
# Project root
flutter analyze
flutter test --concurrency=1
flutter build web

# backend/
gofmt -w .
go vet ./...
go test -p 1 -race -cover ./...
```

- Domain tests cover supported telemetry and invalid sensor/timestamp inputs.
- Engine tests cover sustained heat, hysteresis, episode recovery/reopening, low battery, stale and duplicate delivery, silence, restart state, slow subscribers, and failed database commits.
- API tests exercise REST, a real WebSocket connection, initial/current snapshots, readiness, and origin rejection.
- MQTT tests start an ephemeral in-process Mochi broker on loopback, publish malformed/duplicate/fresh MQTT messages using the actual Paho adapter, and assert committed state. The broker dependency is used only in tests.
- Flutter widget tests cover narrow-phone layout, live alerts, and stale connection presentation. A transport test uses a real local HTTP/WebSocket server to check reconnect and backend version reset.
- PostgreSQL integration uses a unique temporary schema when `TEST_DATABASE_URL` is configured; it checks atomic deduplication, restoring persisted state, rollback on a failed state write, and retrying the same event ID. It skips explicitly when the variable is absent. Run it against a disposable developer database with schema-creation rights, not production.

CI is defined in `.github/workflows/ci.yml` and supplies a PostgreSQL service. CI only runs after the repository is placed on GitHub. Docker builds and AWS integration still require their respective runtimes/accounts; a green unit-test run does not validate those deployments.

## Data and operational limits

PostgreSQL schema initialization is automatic for this prototype; there is no migration framework. Telemetry grows without automatic retention. For an extended deployment, add migrations, retention, and time partitioning before running continuously. Snapshot state includes the most recent 100 alert episodes, preserving active episodes. The demo memory store is disposable.

The local database password is a development fixture and the local broker allows anonymous connections within the Compose network. Host ports are published only on loopback. Use managed secrets, authenticated/TLS connections, and restricted networking for a hosted environment.

## Troubleshooting

- **Port 8080 already used:** stop either the demo Go process or Compose API; run one backend at a time.
- **App keeps reconnecting:** verify `/healthz`, API base URL, and your allowed browser origin/port. An HTTPS app requires HTTPS/WSS rather than HTTP/WS.
- **API ready but no reading:** start the simulator after the subscriber is ready; check the topic and client IDs.
- **No overheating alert:** wait for sustained hot samples; a single temperature spike is insufficient.
- **Offline does not appear immediately:** wait at least ten seconds after the last fresh sample, plus the one-second offline-check tick.
- **No historical state after restart:** `--demo` is intentionally ephemeral. Use the PostgreSQL mode.
- **AWS errors:** use the [AWS runbook](AWS_SETUP.md), especially certificate state, policy attachment, exact client ID, region, endpoint, and CA.
