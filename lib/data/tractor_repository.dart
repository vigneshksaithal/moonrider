import 'dart:async';
import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import 'models.dart';

/// One repository owns one connection: REST first, then live socket updates.
class TractorRepository {
  TractorRepository(String baseUrl)
    : _base = Uri.parse(baseUrl),
      _dio = Dio(
        BaseOptions(
          baseUrl: baseUrl,
          connectTimeout: const Duration(seconds: 5),
          receiveTimeout: const Duration(seconds: 5),
        ),
      );

  final Uri _base;
  final Dio _dio;
  final _updates = StreamController<LiveState>();
  final _stopped = Completer<void>();
  WebSocketChannel? _channel;
  TractorSnapshot? _snapshot;
  bool get _closed => _stopped.isCompleted;

  Stream<LiveState> watch() {
    _updates.onListen = _connect;
    _updates.onCancel = close;
    return _updates.stream;
  }

  void _emit(String message, {bool connected = false}) {
    if (!_closed) {
      _updates.add(
        LiveState(snapshot: _snapshot, connected: connected, message: message),
      );
    }
  }

  Future<void> _connect() async {
    var failures = 0;
    while (!_closed) {
      _emit(
        _snapshot == null
            ? 'Connecting to your tractor…'
            : 'Connection interrupted. Reconnecting…',
      );
      try {
        final response = await _dio.get<Map<String, dynamic>>(
          '/api/tractors/MR-001',
        );
        if (_closed) return;
        _snapshot = TractorSnapshot.fromJson(response.data!);
        _emit('Opening live connection…');
        final socket = WebSocketChannel.connect(
          _base.replace(
            scheme: _base.scheme == 'https' ? 'wss' : 'ws',
            path: '/api/live',
            query: null,
            fragment: null,
          ),
        );
        _channel = socket;
        await socket.ready.timeout(const Duration(seconds: 8));
        // The server sends a full snapshot immediately, then ordered updates.
        await for (final message in socket.stream.timeout(
          const Duration(seconds: 15),
        )) {
          if (_closed) return;
          _snapshot = TractorSnapshot.fromJson(
            jsonDecode(message as String) as Map<String, dynamic>,
          );
          failures = 0;
          _emit('Live connection', connected: true);
        }
      } on Object {
        // Bad payloads and network failures keep the last known readings on screen.
      } finally {
        unawaited(_channel?.sink.close());
        _channel = null;
      }
      if (_closed) return;
      _emit('Unable to reach live monitoring. Retrying…');
      final seconds = [1, 2, 4, 8][failures.clamp(0, 3)];
      failures++;
      await _waitToRetry(Duration(seconds: seconds));
    }
  }

  Future<void> _waitToRetry(Duration delay) async {
    final elapsed = Completer<void>();
    final timer = Timer(delay, elapsed.complete);
    await Future.any([elapsed.future, _stopped.future]);
    timer.cancel();
  }

  void close() {
    if (_closed) return;
    _stopped.complete();
    _dio.close(force: true);
    unawaited(_channel?.sink.close());
    unawaited(_updates.close());
  }
}
