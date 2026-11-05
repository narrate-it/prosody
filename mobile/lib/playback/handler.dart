import 'dart:async';
import 'dart:io';

import '../data/authorized_media.dart';
import 'authorized_media_buffer.dart';

import 'package:audio_service/audio_service.dart';
import 'package:audio_session/audio_session.dart';
import 'package:flutter/foundation.dart';
import 'package:just_audio/just_audio.dart' as audio;
import 'package:video_player/video_player.dart';

import '../data/models.dart';
import 'queue.dart';
import 'timeline.dart';

class PlaybackLocation {
  final int item, segment;
  const PlaybackLocation(this.item, this.segment);
}

// One handler owns both the visible player and OS/background controls.
class ProsodyPlayback extends BaseAudioHandler with SeekHandler {
  final audio.AudioPlayer _audio;
  final AuthorizedMediaBuffer? authorizedMedia;
  File? _authorizedFile;
  VideoPlayerController? _initializingVideo;

  final Map<VideoPlayerController, Future<void>> _videoDisposals = {};

  Future<void> _disposeVideo(VideoPlayerController? controller) {
    if (controller == null) {
      return Future.value();
    }
    final existing = _videoDisposals[controller];
    if (existing != null) {
      return existing;
    }
    final completion = Completer<void>();
    _videoDisposals[controller] = completion.future;
    unawaited(() async {
      try {
        await controller.dispose();
        // This is our own completion future; awaiting it would deadlock.
        unawaited(_videoDisposals.remove(controller));
        completion.complete();
      } catch (error, stack) {
        // Retain failed disposal as a barrier against unsafe file deletion.
        completion.completeError(error, stack);
      }
    }());
    return completion.future;
  }

  Future<void>? _initializingDisposal;
  final VideoPlayerController Function(Uri, File?, VideoPlayerOptions)
  _videoFactory;

  Future<void> _disposeInitializing(VideoPlayerController? controller) {
    if (controller == null || _initializingVideo != controller) {
      return Future.value();
    }
    final existing = _initializingDisposal;
    if (existing != null) {
      return existing;
    }
    final completion = Completer<void>();
    // Publish the barrier before calling dispose, including synchronous effects.
    // Keep the matching controller/file visible until disposal actually ends.
    _initializingDisposal = completion.future;
    unawaited(() async {
      try {
        await _disposeVideo(controller);
        if (_initializingVideo == controller) {
          _initializingVideo = null;
        }
        _initializingDisposal = null;
        completion.complete();
      } catch (error, stack) {
        // Retain the failed barrier: another transition cannot treat a failed
        // disposal as evidence that deleting the decoder's file is safe.
        completion.completeError(error, stack);
      }
    }());
    return completion.future;
  }

  static VideoPlayerController _createVideo(
    Uri uri,
    File? file,
    VideoPlayerOptions options,
  ) => file == null
      ? VideoPlayerController.networkUrl(uri, videoPlayerOptions: options)
      : VideoPlayerController.file(file, videoPlayerOptions: options);

  ProsodyPlayback({
    audio.AudioPlayer? audioPlayer,
    this.authorizedMedia,
    VideoPlayerController Function(Uri, File?, VideoPlayerOptions)?
    videoFactory,
  }) : _audio = audioPlayer ?? audio.AudioPlayer(),
       _videoFactory = videoFactory ?? _createVideo;

  factory ProsodyPlayback.authorized({
    required AuthorizedMediaTransport transport,
    required int maxBufferedBytes,
    audio.AudioPlayer? audioPlayer,
  }) {
    late final ProsodyPlayback playback;
    final buffer = AuthorizedMediaBuffer(
      transport: transport,
      maxAssetBytes: maxBufferedBytes,
      onAccessEnded: () async {
        await playback.stop();
        playback.failure.value = 'Media access ended. Sign in again.';
        playback._broadcast();
      },
    );
    playback = ProsodyPlayback(
      audioPlayer: audioPlayer,
      authorizedMedia: buffer,
    );
    return playback;
  }

  /// Session owner must await this on logout/workspace removal/revocation.
  Future<void> revokeMediaAccess() async {
    await authorizedMedia?.close();
  }

  final ValueNotifier<PlaybackLocation> location = ValueNotifier(
    const PlaybackLocation(0, 0),
  );
  final ValueNotifier<VideoPlayerController?> video = ValueNotifier(null);
  final ValueNotifier<int> completed = ValueNotifier(0);
  bool _completed = false;
  bool _waitingForMore = false;
  Timer? _visualTimer;
  final Set<String> visibleImages = {};
  Duration get itemOffset => current == null
      ? Duration.zero
      : current!.segments
            .take(location.value.segment)
            .fold(Duration.zero, (sum, segment) => sum + segment.duration);
  final ValueNotifier<int> discontinuity = ValueNotifier(0);
  Briefing? get current => _items.isEmpty ? null : _items[location.value.item];
  final ValueNotifier<String?> failure = ValueNotifier(null);
  final List<StreamSubscription<dynamic>> _subscriptions = [];
  List<Briefing> _items = [];
  int _generation = 0;
  int? _advancingGeneration;
  bool get _advancing => _advancingGeneration == _generation;
  bool _transitioning = false;
  bool _playing = false;
  bool get isPlaying => _playing;
  Duration get position => _segment == null || _segment!.kind == 'image'
      ? Duration.zero
      : video.value?.value.position ?? _audio.position;
  Duration get duration => _segment?.duration ?? Duration.zero;
  Segment? get _segment => _items.isEmpty
      ? null
      : _items[location.value.item].segments[location.value.segment];
  Future<void> configure() async {
    final session = await AudioSession.instance;
    await session.configure(const AudioSessionConfiguration.speech());
    _subscriptions.add(
      _audio.playerStateStream.listen((s) {
        if (_transitioning ||
            video.value != null ||
            _segment?.kind == 'image') {
          return;
        }
        _playing = s.playing;
        _broadcast();
        if (s.processingState == audio.ProcessingState.completed) {
          _decoderCompleted();
        }
      }),
    );
    _subscriptions.add(
      _audio.positionStream.listen((_) {
        if (video.value == null) _broadcast();
      }),
    );
    _subscriptions.add(
      _audio.errorStream.listen((_) {
        if (!_transitioning) _fail();
      }),
    );
    _subscriptions.add(
      session.becomingNoisyEventStream.listen((_) {
        unawaited(pause());
      }),
    );
    _subscriptions.add(
      session.interruptionEventStream.listen((event) {
        if (event.begin) unawaited(pause());
      }),
    );
  }

  void _broadcast() {
    final index = _items.isEmpty ? null : location.value.item;
    playbackState.add(
      PlaybackState(
        controls: [
          MediaControl.skipToPrevious,
          _playing ? MediaControl.pause : MediaControl.play,
          MediaControl.skipToNext,
        ],
        systemActions: const {
          MediaAction.seek,
          MediaAction.seekForward,
          MediaAction.seekBackward,
        },
        androidCompactActionIndices: const [0, 1, 2],
        processingState: failure.value != null
            ? AudioProcessingState.error
            : _segment == null
            ? AudioProcessingState.idle
            : video.value != null
            ? (video.value!.value.isBuffering
                  ? AudioProcessingState.buffering
                  : AudioProcessingState.ready)
            : _segment!.kind == 'image'
            ? AudioProcessingState.ready
            : switch (_audio.processingState) {
                audio.ProcessingState.idle => AudioProcessingState.idle,
                audio.ProcessingState.loading => AudioProcessingState.loading,
                audio.ProcessingState.buffering =>
                  AudioProcessingState.buffering,
                audio.ProcessingState.ready => AudioProcessingState.ready,
                audio.ProcessingState.completed =>
                  AudioProcessingState.completed,
              },
        playing: _playing,
        updatePosition: itemOffset + position,
        bufferedPosition: _segment == null || _segment!.kind == 'image'
            ? itemOffset
            : video.value == null
            ? itemOffset + _audio.bufferedPosition
            : itemOffset + position,
        speed: video.value?.value.playbackSpeed ?? _audio.speed,
        queueIndex: index,
        errorMessage: failure.value,
      ),
    );
  }

  void _fail() {
    failure.value = 'Playback stopped. Check your connection and try again.';
    _playing = false;
    _broadcast();
  }

  Future<void> replaceFeed(
    List<Briefing> items, {
    int initial = 0,
    int initialSegment = 0,
    Duration initialPosition = Duration.zero,
  }) async {
    final replacementGeneration = _generation + 1;
    await stop();
    if (_generation != replacementGeneration) return;
    _items = newPlayableItems(const [], items);
    queue.add(
      _items
          .map(
            (e) => MediaItem(
              id: e.id,
              title: e.title,
              artist: e.agent,
              duration: e.duration,
            ),
          )
          .toList(),
    );
    if (_items.isNotEmpty) {
      final index = initial.clamp(0, _items.length - 1);
      final segment = initialSegment.clamp(
        0,
        _items[index].segments.length - 1,
      );
      final expectedGeneration = _generation + 1;
      await select(index, segment: segment, autoplay: false);
      if (_generation != expectedGeneration) return;
      if (initialPosition > Duration.zero) await seekSegment(initialPosition);
      if (_generation == expectedGeneration) await play();
    }
  }

  void appendItems(List<Briefing> items) {
    final oldCount = _items.length;
    final resume = _waitingForMore;
    _items = [..._items, ...newPlayableItems(_items, items)];
    queue.add(
      _items
          .map(
            (e) => MediaItem(
              id: e.id,
              title: e.title,
              artist: e.agent,
              duration: e.duration,
            ),
          )
          .toList(),
    );
    if (resume && _items.length > oldCount) {
      _waitingForMore = false;
      unawaited(select(oldCount));
    }
  }

  Future<void> select(
    int index, {
    int segment = 0,
    bool autoplay = true,
  }) async {
    if (index < 0 ||
        index >= _items.length ||
        segment < 0 ||
        segment >= _items[index].segments.length) {
      return;
    }
    final generation = ++_generation;
    authorizedMedia?.cancelPending();
    _transitioning = true;
    _visualTimer?.cancel();
    _visualTimer = null;
    _waitingForMore = false;
    _playing = false;
    await _audio.stop();
    if (generation != _generation) return;
    await _disposeInitializing(_initializingVideo);
    await Future.wait(_videoDisposals.values.toList());
    if (generation != _generation) return;
    final previous = video.value;
    video.value = null;
    final previousFile = _authorizedFile;
    _authorizedFile = null;
    await _disposeVideo(previous);
    if (previousFile != null) await authorizedMedia?.release(previousFile);
    if (generation != _generation) return;
    _completed = false;
    location.value = PlaybackLocation(index, segment);
    mediaItem.add(queue.value[index]);
    failure.value = null;
    final selected = _segment!;
    VideoPlayerController? initializing;
    try {
      if (selected.kind == 'image') {
        _transitioning = false;
        if (autoplay) {
          await play();
        } else {
          _broadcast();
        }
        return;
      }
      if (selected.media == null) {
        throw const FormatException('Missing playable asset');
      }
      final buffer = authorizedMedia;
      final local = buffer == null
          ? null
          : await buffer.prepare(selected.media!);
      if (generation != _generation) {
        if (local != null) await buffer!.release(local);
        return;
      }
      _authorizedFile = local;
      if (selected.kind == 'video') {
        final options = VideoPlayerOptions(allowBackgroundPlayback: true);
        final controller = _videoFactory(selected.media!, local, options);
        initializing = controller;
        _initializingVideo = controller;
        await controller.initialize();
        if (generation != _generation) {
          await _disposeInitializing(controller);
          initializing = null;
          return;
        }
        video.value = controller;
        _initializingVideo = null;
        initializing = null;
        controller.addListener(() => _videoChanged(controller));
      } else {
        if (local == null) {
          await _audio.setUrl(selected.media.toString());
        } else {
          await _audio.setFilePath(local.path);
        }
        if (generation != _generation) return;
      }
      _transitioning = false;
      if (autoplay) {
        await play();
      } else {
        _broadcast();
      }
    } catch (_) {
      await _disposeInitializing(initializing);
      if (generation == _generation) {
        _transitioning = false;
        _fail();
      }
    }
  }

  void _videoChanged(VideoPlayerController controller) {
    if (video.value != controller) return;
    if (controller.value.hasError) {
      _fail();
      return;
    }
    _playing = controller.value.isPlaying;
    _broadcast();
    if (controller.value.isCompleted) _decoderCompleted();
  }

  void _decoderCompleted() {
    if (_transitioning || _completed || _advancing || _segment == null) return;
    _completed = true;
    completed.value++;
    unawaited(_advanceSegment());
  }

  Future<void> _advanceSegment() async {
    if (_advancing || _items.isEmpty) return;
    final advancingGeneration = _generation;
    _advancingGeneration = advancingGeneration;
    try {
      final p = location.value;
      if (p.segment + 1 < _items[p.item].segments.length) {
        await select(p.item, segment: p.segment + 1);
      } else if (p.item + 1 < _items.length) {
        await select(p.item + 1);
      } else {
        final generation = _generation;
        await pause();
        if (generation == _generation) _waitingForMore = true;
      }
    } finally {
      if (_advancingGeneration == advancingGeneration) {
        _advancingGeneration = null;
      }
    }
  }

  Future<void> advanceVisual() => _advanceSegment();
  @override
  Future<void> play() async {
    if (_segment == null) return;
    final generation = _generation;
    if (_segment!.kind == 'image') {
      // A visual-only segment has no audio duration. Give it a default dwell
      // so locked-screen queues continue; elapsed time never credits a view.
      // Resuming a paused visual gives the person a fresh five-second dwell.
      if (_visualTimer?.isActive == true) return;
      _playing = true;
      _visualTimer = Timer(const Duration(seconds: 5), () {
        _visualTimer = null;
        if (generation == _generation && _playing) {
          unawaited(_advanceSegment());
        }
      });
      _broadcast();
      return;
    }
    if (_completed && position >= duration) await seekSegment(Duration.zero);
    if (generation != _generation) return;
    final session = await AudioSession.instance;
    if (!await session.setActive(true) ||
        generation != _generation ||
        _segment == null) {
      return;
    }
    if (video.value != null) {
      await video.value!.play();
    } else {
      unawaited(
        _audio.play().catchError((Object _) {
          if (generation == _generation) _fail();
        }),
      );
    }
    if (generation != _generation) return;
    _playing = true;
    _broadcast();
  }

  @override
  Future<void> pause() async {
    final generation = _generation;
    _visualTimer?.cancel();
    _visualTimer = null;
    _waitingForMore = false;
    await _audio.pause();
    if (generation != _generation) return;
    await video.value?.pause();
    if (generation != _generation) return;
    _playing = false;
    _broadcast();
  }

  @override
  Future<void> seek(Duration position) async {
    final item = current;
    if (item == null) return;
    final target = seekInTimeline(
      item.segments.map((segment) => segment.duration).toList(),
      position,
    );
    final index = target.segment;
    final resume = _playing;
    final targetItem = location.value.item;
    var expectedGeneration = _generation;
    discontinuity.value++;
    if (index != location.value.segment) {
      expectedGeneration++;
      await select(targetItem, segment: index, autoplay: false);
    }
    if (_generation != expectedGeneration ||
        current?.id != item.id ||
        location.value.segment != index) {
      return;
    }
    await seekSegment(target.position);
    if (_generation == expectedGeneration && resume) await play();
  }

  Future<void> seekSegment(Duration position) async {
    if (_segment == null || _segment!.kind == 'image') {
      return;
    }
    _completed = false;
    discontinuity.value++;
    final bounded = Duration(
      milliseconds: position.inMilliseconds.clamp(0, duration.inMilliseconds),
    );
    if (video.value != null) {
      await video.value!.seekTo(bounded);
    } else {
      await _audio.seek(bounded);
    }
    _broadcast();
  }

  @override
  Future<void> skipToNext() async {
    if (location.value.item + 1 < _items.length) {
      await select(location.value.item + 1);
    }
  }

  @override
  Future<void> skipToPrevious() async {
    if (location.value.item > 0) await select(location.value.item - 1);
  }

  @override
  Future<void> skipToQueueItem(int index) => select(index);
  @override
  Future<void> stop() async {
    final generation = ++_generation;
    authorizedMedia?.cancelPending();
    _transitioning = true;
    _visualTimer?.cancel();
    _visualTimer = null;
    _waitingForMore = false;
    await _audio.stop();
    if (generation != _generation) return;
    await _disposeInitializing(_initializingVideo);
    await Future.wait(_videoDisposals.values.toList());
    if (generation != _generation) return;
    final previous = video.value;
    video.value = null;
    final previousFile = _authorizedFile;
    _authorizedFile = null;
    await _disposeVideo(previous);
    if (previousFile != null) await authorizedMedia?.release(previousFile);
    if (generation != _generation) return;
    _playing = false;
    _items = [];
    visibleImages.clear();
    location.value = const PlaybackLocation(0, 0);
    queue.add([]);
    mediaItem.add(null);
    _transitioning = false;
    _broadcast();
    await super.stop();
  }

  Future<void> dispose() async {
    await revokeMediaAccess();
    await stop();
    for (final subscription in _subscriptions) {
      await subscription.cancel();
    }
    await _audio.dispose();
    location.dispose();
    video.dispose();
    failure.dispose();
    discontinuity.dispose();
    completed.dispose();
  }
}
