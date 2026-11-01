import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:moonrider/data/models.dart';
import 'package:moonrider/state/providers.dart';
import 'package:moonrider/ui/moonrider_app.dart';

void main() {
  final fault = FaultAlert(
    id: 'heat-1',
    kind: 'overheating',
    title: 'Motor overheating',
    message: 'Motor above 80°C for 5s.',
    openedAt: DateTime(2026, 11, 1, 12),
  );
  final snapshot = TractorSnapshot(
    version: 1,
    online: true,
    demo: true,
    alerts: [fault],
    reading: const TractorReading(battery: 76, temperature: 88, speed: 6),
    lastSeen: DateTime(2026, 11, 1, 12),
  );

  testWidgets('shows live health and an active alert on a narrow phone', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          liveStateProvider.overrideWith(
            (ref) =>
                Stream.value(LiveState(snapshot: snapshot, connected: true)),
          ),
        ],
        child: const MoonriderApp(),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.byType(SvgPicture), findsOneWidget);
    expect(find.text('Online'), findsOneWidget);
    await tester.tap(find.text('Alerts'));
    await tester.pumpAndSettle();
    expect(find.text('Motor overheating'), findsOneWidget);
    expect(find.text('Active'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('disconnected app never claims tractor is currently online', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          liveStateProvider.overrideWith(
            (ref) => Stream.value(
              LiveState(snapshot: snapshot, message: 'Reconnecting…'),
            ),
          ),
        ],
        child: const MoonriderApp(),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Online'), findsNothing);
    expect(find.text('Status unavailable'), findsOneWidget);
    expect(find.textContaining('Showing last known readings'), findsOneWidget);
  });
}
