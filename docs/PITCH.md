# Pitch and demo

## 30-second introduction

“I built Moonrider, an electric-tractor telemetry prototype, to demonstrate the device-to-cloud-to-app work in your Full Stack Developer role. A Go simulator publishes MQTT readings, the Go backend validates and processes them, PostgreSQL commits the readings and health state together, and Flutter displays live diagnostics and alerts. The demo focuses on what happens when readings are duplicated or a tractor goes silent, as well as the normal live updates.”

Say “AWS IoT Core” in this pitch only after you have run the AWS setup and demonstrated that path. The no-setup `--demo` path does not use AWS or MQTT.

## 90-second walkthrough

Before recording, run the full local pipeline or the AWS integration, start the app, and then start `go run ./cmd/simulator --scenario cycle --duplicate` in a fresh terminal. The cycle begins at simulator startup. Docker/compilation/startup can take time; rehearse once first.

| Time from simulator start | Show | Explain |
| --- | --- | --- |
| 0–10s | Health screen: battery, temperature, speed, online | “These are synthetic readings arriving through the same ingestion path used by the cloud integration.” |
| 10–23s | Motor becomes hot; open Alerts after roughly 16s | “A sustained temperature rule opens one alert. Repeated and duplicate readings do not create new episodes.” |
| 23–34s | Cooling; alert moves to resolved after roughly 29s | “Recovery uses a lower threshold and a sustained window, reducing alert chatter.” |
| 34–44s | Low-battery alert | “Battery monitoring uses a separate rule with recovery hysteresis.” |
| 44–50s | Battery returns to normal | “The backend resolves the existing episode automatically.” |
| 50–65s | Silent telemetry uplink; offline appears roughly 10s after the last reading | “The app connection remains live, but the tractor's telemetry is stale. Those are different failures.” |
| 65–80s | Replay plus fresh readings restore online status | “An old reading cannot restore freshness. The next fresh sample resolves the single offline episode.” |
| 80–90s | Architecture diagram | “This implementation has one ordered processor. Kafka partitions and Flink keyed state are the planned fleet-scale evolution.” |

To demonstrate the Flutter connection-recovery behavior separately, stop the Go backend, show the reconnecting banner with last-known readings, then restart it. `--demo` resets all state on restart; normal PostgreSQL mode restores persisted state.

## Match the job description honestly

| Role requirement | Evidence in this repository |
| --- | --- |
| Go concurrency | Bounded delivery channel, independent MQTT ingestion, ordered worker, context cancellation, race-tested state access |
| MQTT / IoT | QoS 1 publisher/subscriber, manual subscriber ACK, persistent-session settings, re-subscription, actual broker integration test |
| Flutter / API integration | Riverpod state, Dio bootstrap, typed data models, WebSocket stream and reconnect behavior |
| Relational storage | PostgreSQL adapter, unique event ID, atomic event/state transaction, isolated database integration test |
| AWS | IoT Core certificate/TLS adapter, separate restricted policy templates, manual setup runbook; cloud execution requires your account setup |
| Clean Architecture / SOLID | Domain/application/adapter boundaries, application-owned repository port, injected dependencies, thin composition roots |
| Testing | Fault behavior, storage failure/retry, stale replay, API/socket consistency, narrow-screen widgets, transport reconnect |
| Docker / CI | Compose with broker/database, non-root backend container, format/analyze/vet/test/build workflow |
| Kafka / Flink / Redis | **Design only**, with concrete partitioning, time-window, replay, and caching responsibilities in the architecture document |
| RDS / Firebase / Terraform / Kubernetes | **Not implemented**; do not claim hands-on evidence from this MVP |

The tests provide automated regression coverage. Do not describe the project as strict TDD unless that was the actual development process.

## Questions to prepare for

**Why one processor?**
One tractor needs ordered state transitions. Concurrent workers for the same tractor would add coordination and ordering problems. The next stage partitions work by tractor ID, allowing different tractors to process in parallel.

**Why no microservices yet?**
The first deliverable has one vehicle and one complete workflow. Internal boundaries isolate ingestion, rules, persistence, and delivery so they can be separated when throughput or operational ownership warrants it. Microservices are an explicit next-stage design, not a claim about the current deployment.

**What guarantees do you provide?**
Broker delivery is at least once. Unique event IDs and atomic database transactions prevent a duplicate accepted event from changing state twice. Alert episodes are transition-based. Live snapshots are best-effort and replaceable: the app recovers using a full snapshot. This is not a claim of end-to-end exactly-once delivery or complete offline history.

**What happens during a database outage?**
The processor retries without acknowledging the delivery; the channel bounds backlog. The MVP also rejects data once it ages beyond the freshness window, so extended outages can discard old samples. For complete history, separate a durable raw-event log from the live-health projection.

**What are the first production changes?**
Identity and authorization, per-tractor persistence and ownership, durable raw ingestion, schema migrations/retention, TLS for the app API, observability, and measured load testing. After that, use the scaling plan rather than simply increasing backend replicas.

**Is overheating predictive maintenance?**
No. It is rule-based fault detection on synthetic telemetry. The thresholds are illustrative and would need validation by vehicle and battery engineers before any real use.

## Deliverables to share

- A GitHub repository after you initialize and publish it.
- The README with quick-start commands and a clear implemented/deferred feature list.
- The architecture diagram and scaling discussion.
- A short recording showing the fault, recovery, and duplicate-delivery behavior.
- If AWS is configured, evidence of MQTT messages in IoT Core and corresponding app updates.

No repository has been published, video recorded, or AWS infrastructure created automatically.
