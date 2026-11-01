# Run Moonrider on the iOS simulator

This Mac has Xcode 27.2 and an iOS 27.2 iPhone 16e simulator installed. In Xcode 27, Apple's Simulator app is called **Device Hub**.

## Open the simulator

Use Spotlight (`Command + Space`), search for **Device Hub**, and open it. Select the existing iPhone 16e. You can also run this yourself in Terminal:

```sh
open -a DeviceHub
```

On Xcode 26 or earlier, use `open -a Simulator` instead.

## Run the backend

If the local preview's backend is still running on port 8080, reuse it. Otherwise open a Terminal tab and run:

```sh
cd /Users/vigneshaithal/Code/moonrider/backend
go run ./cmd/api --demo
```

Leave this terminal running. `--demo` supplies synthetic tractor readings without AWS or Docker. It deliberately bypasses MQTT and PostgreSQL.

## Run the Flutter app

In a separate Terminal tab:

```sh
cd /Users/vigneshaithal/Code/moonrider
export DEVELOPER_DIR=/Applications/Xcode.app/Contents/Developer
flutter devices
flutter run -d 27EAC75D-95AC-417E-9AA3-6B28DB64C675
```

That device ID identifies the existing iPhone 16e on this Mac. If you later recreate the simulator or choose another model, copy its ID from `flutter devices`.

The `DEVELOPER_DIR` setting is needed because this Mac's system selection currently points at `/Library/Developer/CommandLineTools`. This command selects full Xcode for the current terminal without changing the system-wide setting.

The native iOS debug build and the app's live connection were verified on this simulator. A screenshot is available at `screenshots/ios.jpg`.

If you only want to reopen the same installed app, open Device Hub, select the iPhone 16e, and tap Moonrider on its home screen. Use `flutter run` when developing; it supplies the terminal session for hot reload.

The iOS simulator can access the Mac's backend at the app's default `http://127.0.0.1:8080`. No API URL change is necessary. The first launch can take longer while Flutter downloads its iOS engine artifacts and Xcode builds the app.

Keep `flutter run` attached while developing. Type `r` for hot reload, `R` for hot restart, or `q` to stop the debug session.

## If setup is incomplete

Open Xcode and finish any initial setup or license prompts yourself. If there are no available iPhone simulators, install the iOS platform/runtime in Xcode's Components settings. `flutter doctor -v` reports remaining setup issues. Native Flutter plugins may require CocoaPods or Swift Package Manager support; follow the message from Flutter for the relevant plugin.

## Official reference

[Flutter iOS setup and simulator instructions](https://docs.flutter.dev/platform-integration/ios/setup)

## M1 MacBook Air with 8 GB RAM

Use only one simulator at a time. The existing iPhone 16e is the selected target. Smaller displays can reduce rendering work; changing the iPhone model does not substantially shrink the iOS operating system itself.

Keep the Xcode editor closed once setup is complete; `flutter run` invokes its build tools directly. Use the small Go `--demo` backend while developing the interface, and pause the web preview and Docker workloads during native builds. Reuse the simulator and use hot reload instead of repeatedly creating or resetting devices. A connected physical iPhone avoids the simulator's OS overhead if one is available.

### Reuse an existing build without compiling

When the code has not changed and the compiled simulator bundle still exists:

```sh
flutter run -d 27EAC75D-95AC-417E-9AA3-6B28DB64C675 --use-application-binary=build/ios/iphonesimulator/Runner.app
```

Run this after the simulator has reached its home screen. The first attempt during initial iOS setup failed; the successfully verified launch occurred after the simulator finished restarting. For ordinary code changes, use the normal `flutter run` command instead so the binary stays current.
