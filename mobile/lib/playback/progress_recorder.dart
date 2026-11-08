import 'dart:async';

import 'package:audio_service/audio_service.dart';
import 'package:flutter/foundation.dart';

import '../data/api.dart';
import '../data/models.dart';
import 'coverage.dart';
import 'position.dart';
import 'handler.dart';

class _PendingProgress {
  final Briefing item;
  final int segment;
  final List<PlayedInterval> intervals;
  final int position;
  final bool viewed;
  _PendingProgress(
    this.item,
    this.segment,
    this.intervals,
    this.position,
    this.viewed,
  );
}

// One recorder accompanies the playback handler, including while the app is in
// the background. It never reports human progress to an agent endpoint.
class ProgressRecorder {
  final ProsodyApi api;
  final ProsodyPlayback handler;
  final ValueNotifier<String?> failure = ValueNotifier(null);
  final CoverageSampler _sampler = CoverageSampler();
  final Stopwatch _clock = Stopwatch()..start();
  final Map<String, int> _revisions = {};
  final List<_PendingProgress> _pending = [];
  Timer? _timer;
  StreamSubscription<PlaybackState>? _playbackSubscription;
  bool _wasActive = false;
  Briefing? _item;
  int _segment = 0, _ticks = 0;
  final ResumePosition _resumePosition = ResumePosition();
  int get _position => _resumePosition.value;
  void _observePosition(int milliseconds) {
    final item = _item;
    if (item == null) return;
    _resumePosition.observe(
      milliseconds,
      durationMs: item.segments[_segment].duration.inMilliseconds,
    );
  }

  bool _visible = true, _viewed = false, _sending = false, _closed = false;
  bool _unknownResume = false, _discardPending = false;
  static const _unknownResumeMessage =
      'Your saved place could not be loaded. Playback can continue, but progress syncing is paused.';

  // Called before installing a queue whose initial resume read has completed.
  // Unknown is not an authoritative zero: even visible images must not cause
  // the fallback position to overwrite an existing server position.
  void setResumeWriteGate({required bool resumeKnown}) {
    _unknownResume = !resumeKnown;
    if (_unknownResume) {
      failure.value = _unknownResumeMessage;
    } else if (failure.value == _unknownResumeMessage) {
      failure.value = null;
    }
    _sampler.take();
    _sampler.discontinuity();
    _viewed = false;
    _resumePosition.reset();
  }

  ProgressRecorder(this.api, this.handler) {
    handler.location.addListener(_changed);
    handler.discontinuity.addListener(_reset);
    handler.completed.addListener(_complete);
    _changed();
    _playbackSubscription = handler.playbackState.listen(_observePlayback);
    _timer = Timer.periodic(
      const Duration(milliseconds: 500),
      (_) => _sample(),
    );
  }
  void _observePlayback(PlaybackState state) {
    if (_closed ||
        _item == null ||
        failure.value != null ||
        state.processingState == AudioProcessingState.completed) {
      return;
    }
    final active =
        state.playing && state.processingState == AudioProcessingState.ready;
    if (active == _wasActive) return;
    _observePosition(handler.position.inMilliseconds);
    if (_wasActive) {
      _sampler.sample(
        positionMs: _position,
        monotonicMs: _clock.elapsedMilliseconds,
        playing: true,
        ready: true,
        speed: state.speed,
      );
    }
    _sampler.discontinuity();
    if (active) {
      _sampler.sample(
        positionMs: _position,
        monotonicMs: _clock.elapsedMilliseconds,
        playing: true,
        ready: true,
        speed: state.speed,
      );
    }
    _wasActive = active;
  }

  void setVisible(bool value) {
    _visible = value;
  }

  void _complete() {
    if (_item == null || _unknownResume) return;
    _observePosition(handler.duration.inMilliseconds);
    _sampler.sample(
      positionMs: _position,
      monotonicMs: _clock.elapsedMilliseconds,
      playing: true,
      ready: true,
      speed: handler.playbackState.value.speed,
    );
    _capture();
    unawaited(_send());
  }

  void _reset() {
    _capture();
    _sampler.discontinuity();
  }

  void _changed() {
    _capture();
    _sampler.discontinuity();
    _wasActive = false;
    _item = handler.current;
    _segment = handler.location.value.segment;
    _resumePosition.reset();
    _viewed = false;
    unawaited(_send());
  }

  void _sample() {
    if (_closed || _item == null || failure.value != null) return;
    final state = handler.playbackState.value;
    _observePosition(handler.position.inMilliseconds);
    _sampler.sample(
      positionMs: _position,
      monotonicMs: _clock.elapsedMilliseconds,
      playing: state.playing,
      ready: state.processingState == AudioProcessingState.ready,
      speed: state.speed,
    );
    final segment = _item!.segments[_segment];
    if (_visible &&
        segment.kind == 'image' &&
        segment.images.isNotEmpty &&
        segment.images.every(
          (uri) =>
              handler.visibleImages.contains('${_item!.id}/${segment.id}/$uri'),
        )) {
      _viewed = true;
    }
    if (++_ticks % 10 == 0) {
      _capture();
      unawaited(_send());
    }
  }

  void _capture() {
    final item = _item;
    final intervals = _sampler.take();
    if (_discardPending ||
        _unknownResume ||
        item == null ||
        (intervals.isEmpty && !_viewed && !_resumePosition.dirty)) {
      return;
    }
    if (_pending.length >= 128) {
      failure.value = 'Progress syncing is paused. Refresh before continuing to record your place.';
      return;
    }
    _pending.add(
      _PendingProgress(item, _segment, intervals, _position, _viewed),
    );
    _viewed = false;
    _resumePosition.captured();
  }

  Future<void> _send() async {
    if (_discardPending || _sending || failure.value != null) return;
    _sending = true;
    try {
      while (_pending.isNotEmpty) {
        if (_discardPending || _unknownResume || failure.value != null) return;
        final update = _pending.first;
        final identity = '${update.item.id}/${update.item.renditionId}';
        if (!_revisions.containsKey(identity)) {
          final progress = await api.progress(
            update.item.id,
            update.item.renditionId,
          );
          if (_discardPending) return;
          _revisions[identity] = SavedProgress.fromJson(
            progress,
            update.item,
          ).revision;
        }
        if (_discardPending || _unknownResume || failure.value != null) return;
        final progress = await api.updateProgress(update.item.id, {
          'renditionId': update.item.renditionId,
          'expectedRevision': _revisions[identity],
          'segment': update.segment,
          'positionMs': update.position,
          'played': update.intervals.map((e) => e.toJson()).toList(),
          'viewed': update.viewed,
        });
        if (_discardPending) return;
        _revisions[identity] = SavedProgress.fromJson(
          progress,
          update.item,
          expectedPreviousRevision: _revisions[identity],
        ).revision;
        _pending.removeAt(0);
      }
    } catch (e) {
      if (_discardPending) return;
      // A transport failure may follow commit. Never blindly advance a local
      // revision or retry an uncertain mutation with a manufactured revision.
      failure.value = e is ApiFailure && e.status == 409
          ? 'Progress changed on another device. Refresh to continue syncing.'
          : 'Listening progress could not be synced. Your current playback can continue.';
    } finally {
      _sending = false;
    }
  }

  // Explicit recovery reconciles the server revision before resending coverage.
  // On conflict the user opts into restoring this device's position.
  Future<void> resumeSync() async {
    if (_sending || _closed) return;
    _unknownResume = false;
    _sampler.take();
    _viewed = false;
    _resumePosition.reset();
    _revisions.clear();
    failure.value = null;
    _sampler.discontinuity();
    await _send();
  }

  // Ordinary teardown retains its final flush. Session end instead discards
  // unsent progress before any await; a request already handed off may commit.
  Future<void> close({bool discardPending = false}) async {
    if (discardPending) {
      _discardPending = true;
      _pending.clear();
      _revisions.clear();
      _sampler.take();
      _sampler.discontinuity();
      _resumePosition.reset();
      _viewed = false;
      _item = null;
    }
    _closed = true;
    _timer?.cancel();
    await _playbackSubscription?.cancel();
    handler.location.removeListener(_changed);
    handler.discontinuity.removeListener(_reset);
    handler.completed.removeListener(_complete);
    _capture();
    await _send();
    // Do not dispose failure while an already-running network write may finish.
  }
}
