import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:moonrider/data/tractor_repository.dart';

void main() {
  test(
    'REST bootstrap, live snapshots, and reconnect after backend restart',
    () async {
      final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
      final sockets = <WebSocket>[];
      var connections = 0;
      Map<String, Object?> snapshot(int version) => {
        'version': version,
        'online': true,
        'demo': true,
        'alerts': [],
        'last_seen': '2026-11-01T10:00:00Z',
        'reading': {'battery_pct': 76, 'motor_temp_c': 50, 'speed_kmh': 6},
      };
      server.listen((request) async {
        if (request.uri.path == '/api/live') {
          final socket = await WebSocketTransformer.upgrade(request);
          sockets.add(socket);
          connections++;
          socket.add(jsonEncode(snapshot(connections == 1 ? 10 : 1)));
        } else {
          request.response.headers.contentType = ContentType.json;
          request.response.write(jsonEncode(snapshot(0)));
          await request.response.close();
        }
      });
      final repository = TractorRepository('http://127.0.0.1:${server.port}');
      final first = Completer<void>();
      final recovered = Completer<void>();
      var sawDisconnect = false;
      final sub = repository.watch().listen((state) {
        if (state.connected &&
            state.snapshot?.version == 10 &&
            !first.isCompleted) {
          first.complete();
        }
        if (first.isCompleted && !state.connected) sawDisconnect = true;
        if (state.connected &&
            state.snapshot?.version == 1 &&
            !recovered.isCompleted) {
          recovered.complete();
        }
      });
      try {
        await first.future.timeout(const Duration(seconds: 5));
        await sockets.first.close();
        await recovered.future.timeout(const Duration(seconds: 5));
        expect(sawDisconnect, isTrue);
        expect(connections, 2);
      } finally {
        repository.close();
        await sub.cancel();
        for (final socket in sockets) {
          await socket.close();
        }
        await server.close(force: true);
      }
    },
  );
  test('disposing during retry stops further network requests', () async {
    final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    var requests = 0;
    server.listen((request) async {
      requests++;
      request.response.statusCode = 503;
      await request.response.close();
    });
    final repository = TractorRepository('http://127.0.0.1:${server.port}');
    final retrying = Completer<void>();
    final subscription = repository.watch().listen((state) {
      if (state.message.contains('Retrying') && !retrying.isCompleted) {
        retrying.complete();
      }
    });
    try {
      await retrying.future.timeout(const Duration(seconds: 5));
      repository.close();
      await subscription.cancel();
      await Future<void>.delayed(const Duration(milliseconds: 1200));
      expect(requests, 1);
    } finally {
      repository.close();
      await subscription.cancel();
      await server.close(force: true);
    }
  });
}
