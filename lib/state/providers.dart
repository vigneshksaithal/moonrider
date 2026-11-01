import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../data/models.dart';
import '../data/tractor_repository.dart';

const apiBaseUrl = String.fromEnvironment(
  'API_BASE_URL',
  defaultValue: 'http://127.0.0.1:8080',
);
final tractorRepositoryProvider = Provider.autoDispose<TractorRepository>((
  ref,
) {
  final repository = TractorRepository(apiBaseUrl);
  ref.onDispose(repository.close);
  return repository;
});
final liveStateProvider = StreamProvider.autoDispose<LiveState>(
  (ref) => ref.watch(tractorRepositoryProvider).watch(),
);
