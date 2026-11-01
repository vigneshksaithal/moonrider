class TractorReading {
  const TractorReading({
    required this.battery,
    required this.temperature,
    required this.speed,
  });
  final double battery;
  final double temperature;
  final double speed;
  factory TractorReading.fromJson(Map<String, dynamic> json) => TractorReading(
    battery: (json['battery_pct'] as num).toDouble(),
    temperature: (json['motor_temp_c'] as num).toDouble(),
    speed: (json['speed_kmh'] as num).toDouble(),
  );
}

class FaultAlert {
  const FaultAlert({
    required this.id,
    required this.kind,
    required this.title,
    required this.message,
    required this.openedAt,
    this.resolvedAt,
  });
  final String id;
  final String kind;
  final String title;
  final String message;
  final DateTime openedAt;
  final DateTime? resolvedAt;
  bool get active => resolvedAt == null;
  factory FaultAlert.fromJson(Map<String, dynamic> json) => FaultAlert(
    id: json['id'] as String,
    kind: json['kind'] as String,
    title: json['title'] as String,
    message: json['message'] as String,
    openedAt: DateTime.parse(json['opened_at'] as String).toLocal(),
    resolvedAt: json['resolved_at'] == null
        ? null
        : DateTime.parse(json['resolved_at'] as String).toLocal(),
  );
}

class TractorSnapshot {
  const TractorSnapshot({
    required this.version,
    required this.online,
    required this.demo,
    required this.alerts,
    this.hotThreshold = 80,
    this.lowThreshold = 20,
    this.reading,
    this.lastSeen,
  });
  final int version;
  final bool online;
  final bool demo;
  final TractorReading? reading;
  final DateTime? lastSeen;
  final List<FaultAlert> alerts;
  final double hotThreshold;
  final double lowThreshold;
  List<FaultAlert> get activeAlerts => alerts.where((a) => a.active).toList();
  factory TractorSnapshot.fromJson(Map<String, dynamic> json) {
    final rules = json['rules'] as Map<String, dynamic>? ?? {};
    return TractorSnapshot(
      version: json['version'] as int,
      online: json['online'] as bool,
      demo: json['demo'] as bool,
      hotThreshold: (rules['hot_c'] as num?)?.toDouble() ?? 80,
      lowThreshold: (rules['low_battery_pct'] as num?)?.toDouble() ?? 20,
      reading: json['reading'] == null
          ? null
          : TractorReading.fromJson(json['reading'] as Map<String, dynamic>),
      lastSeen: json['last_seen'] == null
          ? null
          : DateTime.parse(json['last_seen'] as String).toLocal(),
      alerts: (json['alerts'] as List)
          .map((a) => FaultAlert.fromJson(a as Map<String, dynamic>))
          .toList(growable: false),
    );
  }
}

class LiveState {
  const LiveState({
    this.snapshot,
    this.connected = false,
    this.message = 'Connecting to your tractor…',
  });
  final TractorSnapshot? snapshot;
  final bool connected;
  final String message;
}
