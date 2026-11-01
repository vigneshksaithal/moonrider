import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';

import '../data/models.dart';
import '../state/providers.dart';
import 'brand.dart';

const sectionGap = SizedBox(height: 20);

class MoonriderApp extends StatelessWidget {
  const MoonriderApp({super.key});

  @override
  Widget build(BuildContext context) => MaterialApp(
    title: 'Moonrider',
    debugShowCheckedModeBanner: false,
    theme: moonriderTheme(),
    home: const FleetHome(),
  );
}

/// Reads live state once and chooses which of the two screens to display.
class FleetHome extends ConsumerStatefulWidget {
  const FleetHome({super.key});

  @override
  ConsumerState<FleetHome> createState() => _FleetHomeState();
}

class _FleetHomeState extends ConsumerState<FleetHome> {
  int _page = 0;

  @override
  Widget build(BuildContext context) {
    final live =
        ref.watch(liveStateProvider).asData?.value ?? const LiveState();
    final alerts = live.snapshot?.activeAlerts.length ?? 0;
    return Scaffold(
      appBar: AppBar(
        title: SvgPicture.asset(
          'assets/brand/moonrider-logo.svg',
          width: 180,
          height: 42,
          semanticsLabel: 'Moonrider',
        ),
        actions: [
          Padding(
            padding: const EdgeInsets.only(right: 24),
            child: _Pill(
              live.connected ? 'LIVE' : 'CONNECTING',
              color: brandInk,
              background: live.connected ? brandOrange : appBackground,
            ),
          ),
        ],
        bottom: const PreferredSize(
          preferredSize: Size.fromHeight(2),
          child: ColoredBox(
            color: brandOrange,
            child: SizedBox(height: 2, width: double.infinity),
          ),
        ),
      ),
      body: Column(
        children: [
          if (!live.connected)
            Container(
              width: double.infinity,
              color: brandTint,
              padding: const EdgeInsets.all(12),
              child: Text(
                '${live.message}${live.snapshot == null ? '' : ' Showing last known readings.'}',
                textAlign: TextAlign.center,
                style: const TextStyle(fontSize: 12),
              ),
            ),
          Expanded(
            child: Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 808),
                child: ListView(
                  padding: const EdgeInsets.all(24),
                  children: _page == 0 ? _health(live) : _alerts(live.snapshot),
                ),
              ),
            ),
          ),
        ],
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _page,
        onDestinationSelected: (page) => setState(() => _page = page),
        destinations: [
          const NavigationDestination(
            icon: Icon(Icons.space_dashboard_outlined),
            label: 'Tractor health',
          ),
          NavigationDestination(
            icon: Badge(
              isLabelVisible: alerts > 0,
              label: Text('$alerts'),
              child: const Icon(Icons.notifications_none_rounded),
            ),
            label: 'Alerts',
          ),
        ],
      ),
    );
  }

  List<Widget> _health(LiveState live) {
    final snapshot = live.snapshot;
    final reading = snapshot?.reading;
    final active = snapshot?.activeAlerts ?? [];
    final status = !live.connected
        ? 'Status unavailable'
        : reading == null
        ? 'Awaiting telemetry'
        : snapshot!.online
        ? 'Online'
        : 'Offline';
    return [
      const Text('Tractor health', style: pageTitleStyle),
      sectionGap,
      _Panel(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            ListTile(
              contentPadding: EdgeInsets.zero,
              title: const Text(
                'MR-001',
                style: TextStyle(fontSize: 23, fontWeight: FontWeight.w700),
              ),
              subtitle: const Text('Electric tractor · Demo vehicle'),
              trailing: _Pill(
                status,
                color: live.connected && (snapshot?.online ?? false)
                    ? success
                    : muted,
              ),
            ),
            Text(
              snapshot?.lastSeen == null
                  ? 'Waiting for the first reading'
                  : 'Last update ${_time(snapshot!.lastSeen!)}',
              style: const TextStyle(color: muted, fontSize: 12),
            ),
          ],
        ),
      ),
      sectionGap,
      if (reading != null && (!live.connected || !snapshot!.online))
        const Text('Last known values', style: TextStyle(color: warning)),
      _Metric(
        'Battery charge',
        reading?.battery.toStringAsFixed(0),
        '%',
        Icons.battery_charging_full,
        progress: reading == null ? null : reading.battery / 100,
        alert:
            reading != null && reading.battery < (snapshot?.lowThreshold ?? 20),
      ),
      const SizedBox(height: 12),
      _Metric(
        'Motor temperature',
        reading?.temperature.toStringAsFixed(0),
        '°C',
        Icons.thermostat,
        progress: reading == null ? null : reading.temperature / 120,
        alert:
            reading != null &&
            reading.temperature > (snapshot?.hotThreshold ?? 80),
      ),
      const SizedBox(height: 12),
      _Metric(
        'Ground speed',
        reading?.speed.toStringAsFixed(1),
        'km/h',
        Icons.speed,
      ),
      sectionGap,
      _Panel(
        child: ListTile(
          contentPadding: EdgeInsets.zero,
          leading: Icon(
            active.isEmpty ? Icons.verified_outlined : Icons.warning_amber,
            color: active.isEmpty ? success : warning,
          ),
          title: Text(
            !live.connected || reading == null
                ? 'Monitoring is waiting for data'
                : active.isEmpty
                ? 'No active alerts'
                : '${active.length} active alerts',
          ),
          subtitle: Text(
            active.isEmpty
                ? 'Battery, temperature, and connectivity are monitored automatically.'
                : active.map((a) => a.title).join(' · '),
          ),
        ),
      ),
      sectionGap,
      const Text(
        'Portfolio prototype · Synthetic telemetry',
        textAlign: TextAlign.center,
        style: TextStyle(color: muted, fontSize: 11),
      ),
    ];
  }

  List<Widget> _alerts(TractorSnapshot? snapshot) {
    final active = snapshot?.activeAlerts ?? [];
    final resolved =
        snapshot?.alerts.where((a) => !a.active).toList().reversed.toList() ??
        [];
    return [
      const Text('Alerts & recovery', style: pageTitleStyle),
      sectionGap,
      Text('ACTIVE · ${active.length}'),
      const SizedBox(height: 12),
      if (active.isEmpty)
        _Panel(
          child: ListTile(
            leading: const Icon(Icons.check_circle_outline, color: success),
            title: Text(
              snapshot?.reading == null
                  ? 'Waiting for tractor data'
                  : 'No active alerts',
            ),
            subtitle: const Text('New faults will appear here automatically.'),
          ),
        ),
      for (final alert in active.reversed) _AlertCard(alert),
      sectionGap,
      Text('RESOLVED · ${resolved.length}'),
      const SizedBox(height: 12),
      if (resolved.isEmpty)
        const Text(
          'Resolved alerts will stay here for review.',
          style: TextStyle(color: muted),
        ),
      for (final alert in resolved) _AlertCard(alert),
    ];
  }
}

class _Panel extends StatelessWidget {
  const _Panel({required this.child});
  final Widget child;
  @override
  Widget build(BuildContext context) => Card(
    child: Padding(padding: const EdgeInsets.all(20), child: child),
  );
}

class _Pill extends StatelessWidget {
  const _Pill(this.text, {required this.color, this.background});
  final String text;
  final Color color;
  final Color? background;
  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
    decoration: BoxDecoration(
      color: background ?? color.withValues(alpha: .12),
      borderRadius: BorderRadius.circular(30),
    ),
    child: Text(
      text,
      style: TextStyle(color: color, fontSize: 10, fontWeight: FontWeight.w700),
    ),
  );
}

class _Metric extends StatelessWidget {
  const _Metric(
    this.label,
    this.value,
    this.unit,
    this.icon, {
    this.progress,
    this.alert = false,
  });
  final String label;
  final String? value;
  final String unit;
  final IconData icon;
  final double? progress;
  final bool alert;
  @override
  Widget build(BuildContext context) => _Panel(
    child: Column(
      children: [
        ListTile(
          contentPadding: EdgeInsets.zero,
          leading: Icon(icon, color: alert ? warning : brandOrange),
          title: Text(
            label,
            style: const TextStyle(color: muted, fontSize: 12),
          ),
          trailing: Text(
            '${value ?? '—'} $unit',
            style: TextStyle(
              fontSize: 24,
              fontWeight: FontWeight.w600,
              color: alert ? warning : brandInk,
            ),
          ),
        ),
        if (progress != null)
          LinearProgressIndicator(
            value: progress!.clamp(0, 1),
            color: alert ? warning : brandOrange,
            backgroundColor: appBackground,
            minHeight: 4,
            borderRadius: BorderRadius.circular(8),
          ),
      ],
    ),
  );
}

class _AlertCard extends StatelessWidget {
  const _AlertCard(this.alert);
  final FaultAlert alert;
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(bottom: 12),
    child: _Panel(
      child: ListTile(
        contentPadding: EdgeInsets.zero,
        title: Row(
          children: [
            Expanded(child: Text(alert.title)),
            _Pill(
              alert.active ? 'Active' : 'Resolved',
              color: alert.active ? warning : success,
            ),
          ],
        ),
        subtitle: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(alert.message),
            const SizedBox(height: 8),
            Text(
              'MR-001 · ${_time(alert.openedAt)}${alert.resolvedAt == null ? '' : ' → ${_time(alert.resolvedAt!)}'}',
              style: const TextStyle(fontSize: 11),
            ),
          ],
        ),
      ),
    ),
  );
}

String _time(DateTime time) =>
    '${time.hour.toString().padLeft(2, '0')}:${time.minute.toString().padLeft(2, '0')}:${time.second.toString().padLeft(2, '0')}';
