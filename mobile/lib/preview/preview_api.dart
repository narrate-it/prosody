import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

import '../data/api.dart';
import '../data/mark_read.dart';
import '../data/models.dart';

// Bundled recordings only. This fixture never synthesizes audio at runtime.
const previewNarrationScripts = <String, String>{
  'welcome': 'Welcome to Prosody. This is your private inbox for updates from your agents. A long message becomes a short spoken briefing, so you can catch up while doing something else. Swipe up to hear the next item, or open Read to compare the briefing with its original text. This preview stays on your device.',
  'design': 'The studio\'s design update is ready. We are keeping the experience simple: one briefing at a time, clear playback controls, and a quiet blue waveform when there is nothing to watch. Team members share the same feed, while each person keeps their own listening progress. You can switch back to your personal inbox from the space selector above.',
  'next': 'You are all caught up with the sample updates. Try pausing, seeking, or marking this briefing as read. The activity view shows what has arrived, and the settings screen shows where model and voice choices will live. These recordings are bundled samples. No agent connection, provider request, or account is needed for this preview.',
};

class _PreviewSession implements AccessSession {
  void Function()? onSignOut;
  @override
  Future<String> accessToken() async =>
      throw StateError('Preview has no access token');
  @override
  Future<void> signOut() async {
    onSignOut?.call();
  }
}

class _NoNetworkClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async =>
      throw StateError('Preview network access is disabled');
}

// A separate entrypoint owns this fixture. Production composition must never
// import it. All state disappears with this instance; there is no persistence,
// auth fallback, provider connection or credential storage.
class PreviewApi extends ProsodyApi {
  final _feedItems = <String, List<String>>{
    'personal': ['welcome', 'next'],
    'studio': ['design'],
  };
  static const _titles = {
    'welcome': 'A quieter way to catch up',
    'design': 'A little less interface',
    'next': 'Make yourself at home',
  };
  static const _sources = {
    'welcome': 'Product notes: give each person a private inbox for agent updates. Condense long text into a spoken briefing. Keep the original available in Read. Use an upward swipe to move to the next item. This is a local preview using bundled samples.',
    'design': 'Studio notes: use one full-screen briefing at a time. Keep playback controls clear and the blue waveform restrained. A team shares its feed, but reading and listening progress belong to each person. The space selector should make switching to a personal inbox straightforward.',
    'next': 'Preview checklist: try pause, seek, next, Read, mark as read, Activity, and the space selector. Settings are read-only in this build. Recordings are bundled on the device. No agents, accounts or paid providers are connected. Preview changes are held in memory only.',
  };
  final Map<String, Map<String, dynamic>> _items = {};
  final Map<String, Map<String, dynamic>> _progress = {};
  bool _closed = false;
  final _notificationPreferences = <String, bool>{};

  @override
  bool get notificationPreferencesAreLocal => true;
  final Future<void> Function(String id)? _onDelete;
  final Map<String, Future<void>> _deletions = {};

  Future<void> _removeItem(String id) async {
    try {
      await Future<void>.sync(() => _onDelete?.call(id));
      _items.remove(id);
      _progress.remove(id);
    } catch (_) {
      throw const ApiFailure(
        503,
        'Could not remove local media. Try deleting again.',
      );
    } finally {
      unawaited(_deletions.remove(id));
    }
  }

  PreviewApi({
    required Map<String, int> durationsMs,
    Future<void> Function(String id)? onDelete,
    // Keep the public argument named while storing its callback privately.
    // ignore: prefer_initializing_formals
  }) : _onDelete = onDelete,
       super(
         Uri.parse('https://preview.invalid'),
         _PreviewSession(),
         client: _NoNetworkClient(),
       ) {
    (session as _PreviewSession).onSignOut = _notificationPreferences.clear;
    if (durationsMs.length != previewNarrationScripts.length ||
        previewNarrationScripts.keys.any(
          (id) =>
              durationsMs[id] == null ||
              durationsMs[id]! <= 0 ||
              durationsMs[id]! > 600000,
        )) {
      throw ArgumentError(
        'Measured durations required for all bundled recordings',
      );
    }
    for (final id in previewNarrationScripts.keys) {
      _items[id] = {
        'id': id,
        'renditionId': '$id-preview-v1',
        'title': _titles[id],
        'agent': 'Bundled demo · macOS voice',
        'status': 'ready',
        'source': _sources[id],
        'script': previewNarrationScripts[id],
        'segments': [
          {
            'id': '$id-audio',
            'kind': 'narration',
            'durationMs': durationsMs[id],
            'url': 'https://preview.invalid/$id.mp3',
          },
        ],
      };
      _progress[id] = {
        'revision': 0,
        'positionSegment': 0,
        'positionMs': 0,
        'markedRead': false,
        'segments': [
          {
            'durationMs': durationsMs[id],
            'coverage': <Map<String, int>>[],
            'viewed': false,
          },
        ],
      };
    }
  }

  void addLiveBriefing({
    required String id,
    required String source,
    required String script,
    required int durationMs,
  }) {
    if (_closed ||
        !RegExp(r'^live-[0-9]+$').hasMatch(id) ||
        _items.containsKey(id) ||
        durationMs <= 0 ||
        durationMs > 600000 ||
        source.isEmpty ||
        script.isEmpty) {
      throw ArgumentError('Invalid local live briefing');
    }
    _items[id] = {
      'id': id,
      'renditionId': '$id-v1',
      'title': 'Your live briefing',
      'agent': 'OpenRouter · Laura',
      'status': 'ready',
      'source': source,
      'script': script,
      'segments': [
        {
          'id': '$id-audio',
          'kind': 'narration',
          'durationMs': durationMs,
          'url': 'https://preview.invalid/$id.mp3',
        },
      ],
    };
    _progress[id] = {
      'revision': 0,
      'positionSegment': 0,
      'positionMs': 0,
      'markedRead': false,
      'segments': [
        {
          'durationMs': durationMs,
          'coverage': <Map<String, int>>[],
          'viewed': false,
        },
      ],
    };
    _feedItems['personal']!.insert(0, id);
  }

  Map<String, dynamic> _copy(Map<String, dynamic> value) =>
      jsonDecode(jsonEncode(value)) as Map<String, dynamic>;

  void _query(Map<String, String>? query, [Set<String> allowed = const {}]) {
    if (query != null && query.keys.any((key) => !allowed.contains(key))) {
      throw const ApiFailure(400, 'This preview request is not supported.');
    }
  }

  // The offline preview has no credential. Keep this local-only operation
  // separate from the production action's captured-session transport.
  @override
  Future<void> markRead(Briefing item) async {
    final saved = SavedProgress.fromJson(
      await progress(item.id, item.renditionId),
      item,
    );
    final result = await updateProgress(item.id, markReadUpdate(item, saved));
    SavedProgress.fromJson(
      result,
      item,
      expectedPreviousRevision: saved.revision,
    );
    if (result['markedRead'] != true) {
      throw const FormatException('Read status was not confirmed');
    }
  }

  @override
  Future<Map<String, dynamic>> request(
    String method,
    String path, {
    Map<String, String>? query,
    Object? body,
  }) async {
    if (_closed) throw const ApiFailure(410, 'This preview has ended.');
    if (method == 'GET' && path == '/prosody/v1/feeds') {
      _query(query);
      return {
        'feeds': [
          {
            'id': 'personal',
            'name': 'Your inbox',
            'kind': 'personal',
            'canManage': false,
          },
          {
            'id': 'studio',
            'name': 'Studio',
            'kind': 'team',
            'canManage': false,
          },
        ],
        'nextCursor': null,
      };
    }
    if (method == 'GET' && path == '/prosody/v1/models') {
      _query(query, {'kind'});
      final kind = query?['kind'];
      if (kind != 'text' && kind != 'speech') {
        throw const ApiFailure(400, 'Unknown preview model kind.');
      }
      return {
        'models': [
          kind == 'text'
              ? {
                  'id': 'preview-text',
                  'name': 'Sample briefing (offline)',
                  'voices': <String>[],
                }
              : {
                  'id': 'preview-voice',
                  'name': 'Bundled recording (offline)',
                  'voices': ['bundled'],
                },
        ],
      };
    }
    final notificationRoute = RegExp(
      r'^/prosody/v1/feeds/(personal|studio)/notifications$',
    ).firstMatch(path);
    if (notificationRoute != null) {
      _query(query);
      if (method != 'PUT' ||
          body is! Map<String, dynamic> ||
          body.length != 1 ||
          body['enabled'] is! bool) {
        throw const ApiFailure(
          400,
          'This preview preference request is not supported.',
        );
      }
      _notificationPreferences[notificationRoute[1]!] = body['enabled'] as bool;
      return {};
    }
    final feedRoute = RegExp(
      r'^/prosody/v1/feeds/(personal|studio)/(items|activity|settings)$',
    ).firstMatch(path);
    if (feedRoute != null) {
      if (method != 'GET') {
        throw const ApiFailure(
          403,
          'Preview settings are read-only. No credentials are accepted.',
        );
      }
      _query(query);
      final feed = feedRoute[1]!, kind = feedRoute[2]!;
      if (kind == 'settings') {
        return {
          'revision': 0,
          'textModel': 'preview-text',
          'speechModel': 'preview-voice',
          'voice': 'bundled',
          'hasKey': false,
          'notifications': _notificationPreferences[feed] ?? false,
        };
      }
      final entries = _feedItems[feed]!.where(_items.containsKey).map((id) {
        if (kind == 'items') return _copy(_items[id]!);
        return <String, dynamic>{
          'id': id,
          'title': _items[id]!['title'],
          'source': _items[id]!['source'],
          'status': 'ready',
          'acceptedAt': '2026-11-01T12:00:00Z',
        };
      }).toList();
      return {'items': entries, 'nextCursor': null};
    }
    final progressRoute = RegExp(r'^/prosody/v1/items/([a-z0-9-]+)/progress$')
        .firstMatch(path);
    if (progressRoute != null && _items.containsKey(progressRoute[1])) {
      final id = progressRoute[1]!;
      if (method == 'GET') {
        _query(query, {'renditionId'});
        if (query?['renditionId'] != _items[id]!['renditionId']) {
          throw const ApiFailure(404, 'Preview rendition not found.');
        }
        return _copy(_progress[id]!);
      }
      if (method == 'PUT') {
        _query(query);
        return _updateProgress(id, body);
      }
    }
    final deletion = RegExp(r'^/prosody/v1/items/([a-z0-9-]+)$')
        .firstMatch(path);
    if (method == 'DELETE' && deletion != null) {
      _query(query);
      final id = deletion[1]!;
      if (_items.containsKey(id)) {
        await _deletions.putIfAbsent(id, () => _removeItem(id));
      }
      return {};
    }
    throw const ApiFailure(
      404,
      'This route is not available in the offline preview.',
    );
  }

  Map<String, dynamic> _updateProgress(String id, Object? body) {
    const allowed = {
      'renditionId',
      'expectedRevision',
      'segment',
      'positionMs',
      'played',
      'viewed',
      'markedRead',
    };
    if (body is! Map<String, dynamic> ||
        body.keys.any((key) => !allowed.contains(key))) {
      throw const ApiFailure(400, 'Invalid preview progress update.');
    }
    final current = _progress[id]!;
    if (body['renditionId'] != _items[id]!['renditionId'] ||
        body['expectedRevision'] != current['revision']) {
      throw const ApiFailure(
        409,
        'Preview progress changed. Refresh and try again.',
      );
    }
    final segment =
        (current['segments'] as List).single as Map<String, dynamic>;
    final duration = segment['durationMs'] as int;
    final position = body['positionMs'], played = body['played'];
    if (body['expectedRevision'] is! int ||
        body['segment'] is! int ||
        body['segment'] != 0 ||
        position is! int ||
        position < 0 ||
        position > duration ||
        played is! List ||
        played.length > 256 ||
        body['viewed'] is! bool ||
        (body.containsKey('markedRead') && body['markedRead'] is! bool)) {
      throw const ApiFailure(400, 'Invalid preview progress update.');
    }
    final coverage = <Map<String, int>>[
      for (final interval in segment['coverage'] as List)
        Map<String, int>.from(interval as Map),
    ];
    for (final interval in played) {
      if (interval is! Map ||
          interval['startMs'] is! int ||
          interval['endMs'] is! int ||
          interval['startMs'] < 0 ||
          interval['endMs'] <= interval['startMs'] ||
          interval['endMs'] > duration) {
        throw const ApiFailure(400, 'Invalid preview listening interval.');
      }
      coverage.add({
        'startMs': interval['startMs'] as int,
        'endMs': interval['endMs'] as int,
      });
    }
    coverage.sort((a, b) => a['startMs']!.compareTo(b['startMs']!));
    final merged = <Map<String, int>>[];
    for (final interval in coverage) {
      if (merged.isEmpty || interval['startMs']! > merged.last['endMs']!) {
        merged.add(Map.of(interval));
      } else if (interval['endMs']! > merged.last['endMs']!) {
        merged.last['endMs'] = interval['endMs']!;
      }
    }
    if (merged.length > 4096) {
      throw const ApiFailure(400, 'Preview progress limit reached.');
    }
    final updated = <String, dynamic>{
      'revision': (current['revision'] as int) + 1,
      'positionSegment': 0,
      'positionMs': position,
      'markedRead': body['markedRead'] ?? current['markedRead'],
      'segments': [
        {'durationMs': duration, 'coverage': merged, 'viewed': false},
      ],
    };
    _progress[id] = updated;
    return _copy(updated);
  }

  @override
  void close() {
    _closed = true;
    _items.clear();
    _progress.clear();
    _notificationPreferences.clear();
    super.close();
  }
}
