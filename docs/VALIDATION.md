# Validation record

Verified locally on 1 November 2026 with Go 1.27.3 and Flutter 3.47.9 / Dart 3.13.7.

## Passed

- `go vet ./...`
- `go test -race -cover ./...` and a final `go test -race ./...` regression run
- Fault tests for sustained heat, hysteresis, episode reopening, battery recovery, stale replay, deduplication, silence, restart restoration, detached snapshots, and failed commits
- Input tests for missing/null sensors, ranges, timestamp validity, and invalid rule configuration
- REST and actual WebSocket integration tests, including initial state, committed updates, readiness, and browser-origin rejection
- Actual MQTT publish/subscribe integration against a temporary local Mochi broker, including malformed events and duplicate deliveries
- `flutter analyze` with no issues
- `flutter test`: four tests, including narrow-phone widgets and an actual local HTTP/WebSocket reconnect after a backend version reset
- `flutter build web`: release output in `build/web`
- Visual inspection of the running Flutter health and alerts screens with live synthetic readings; screenshots saved in `docs/screenshots`
- iOS plist, Android manifest XML, AWS policy JSON, web manifest JSON, and Markdown code-fence checks

## Not executed in this environment

- PostgreSQL integration: the test is present and CI supplies a disposable database, but PostgreSQL and Docker were unavailable locally. The test skips without `TEST_DATABASE_URL`.
- Docker image/Compose startup: Docker was not installed.
- AWS certificate provisioning and live IoT Core traffic: requires your AWS account setup. No AWS resources were created.
- Native Android builds and physical-device execution remain unverified. The native iOS simulator debug build and launch were subsequently verified below.
- GitHub Actions: the workflow is supplied but has not run on GitHub.
- Fleet-scale load, distributed processing, public deployment, and disaster recovery: outside the implemented MVP. No throughput or latency capacity claims are made.

The running quick preview uses `--demo`, which exercises the Go processor, HTTP/WebSocket API, and Flutter UI with memory persistence. It is distinct from the independently tested MQTT adapter and the not-yet-run cloud/database deployment.

## Branding update

Applied the official website palette, original SVG logo, website icon, and tractor artwork, with bundled Poppins/Orbitron fonts and their licenses. Source references are recorded in `BRANDING.md`. Flutter analysis, all three tests, and the release web build passed after the theme update. The narrow-phone widget test continues to pass with the new logo and typography.

## Native iOS simulator verification

The native iOS debug build succeeded with Xcode 27.2. Moonrider was then launched on the installed iPhone 16e simulator (iOS 27.2), and its original branding, online state, and live readings from the local Go demo backend were visually verified. Screenshot: `screenshots/ios.jpg`.

The first simulator boot stalled while the 8 GB M1 Mac was under memory pressure. The unused iPhone 18 Pro simulator and web preview process were stopped. After a single simulator restart and reduced memory pressure, the already compiled app launched successfully without another build. The final session runs only the iPhone 16e simulator and the local Go demo backend; the Flutter build process has ended.

## Code simplification

Application source was reduced from 1,918 to 1,631 lines (287 removed, about 15%). The main UI file dropped from 621 to 361 lines by using shared theme styles, Material widgets, and focused health/alert views. Decorative artwork and the unused Cupertino icon dependency were removed. Both simulator modes now share `simulator.Run`.

Threshold environment overrides were removed; `domain.DefaultRules()` is the single configuration point. Flutter stream cleanup now uses one stop signal and a cancellable retry wait. Four Flutter tests and static analysis passed, including backend restart recovery and disposal during retry. Go vet and race tests passed with sequential package execution, including a new duplicate-delivery test for the shared simulator.

The in-memory backend was restarted with the simplified code. AWS provisioning, Docker startup, PostgreSQL integration, and native Android builds remain unverified in this environment as documented above.

After simplification, the native iOS incremental build completed successfully, and the updated health screen, original Moonrider logo, live connection, and readings were visually verified on the same iPhone 16e. The latest screenshot replaces the old README screenshots. Flutter was detached afterward to free its development process while leaving the app running.
