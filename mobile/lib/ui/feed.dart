import 'dart:async';
import 'package:audio_service/audio_service.dart';
import 'package:flutter/material.dart';
import 'package:video_player/video_player.dart';
import '../data/api.dart';
import '../data/models.dart';
import '../data/pagination.dart';
import '../data/resume.dart';
import '../playback/handler.dart';
import '../playback/queue.dart';
import '../playback/progress_recorder.dart';
import 'activity.dart';
import 'settings.dart';
import 'theme.dart';

class FeedScreen extends StatefulWidget {
  final ProsodyApi api;
  final ProsodyPlayback playback;
  final VoidCallback onSignOut;
  const FeedScreen({
    super.key,
    required this.api,
    required this.playback,
    required this.onSignOut,
  });
  @override
  State<FeedScreen> createState() => _FeedScreenState();
}

class _FeedScreenState extends State<FeedScreen> with WidgetsBindingObserver {
  late final ProgressRecorder _progress;
  bool _appVisible = true, _routeVisible = true;
  void _routeVisibility(bool value) {
    _routeVisible = value;
    _progress.setVisible(_appVisible && _routeVisible);
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    _appVisible = state == AppLifecycleState.resumed;
    _progress.setVisible(_appVisible && _routeVisible);
  }

  List<Feed> _feeds = [];
  Feed? _feed;
  List<Briefing> _items = [];
  String? _error, _cursor;
  FeedPagination _pagination = FeedPagination();
  bool _loading = true, _more = false;
  int _loadRevision = 0;
  final _pages = PageController();
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _progress = ProgressRecorder(widget.api, widget.playback);
    widget.playback.location.addListener(_playbackChanged);
    unawaited(_load());
  }

  void _playbackChanged() {
    if (!mounted) return;
    final index = widget.playback.location.value.item;
    if (_pages.hasClients && _pages.page?.round() != index) {
      _pages.jumpToPage(index);
    }
    setState(() {});
    if (index >= _items.length - 3) unawaited(_nextPage());
  }

  Future<void> _load() async {
    try {
      final feeds = await widget.api.feeds();
      if (!mounted) return;
      setState(() => _feeds = feeds);
      if (feeds.isEmpty) {
        setState(() => _loading = false);
        return;
      }
      await _select(feeds.first);
    } catch (e) {
      if (mounted) {
        setState(() {
          _loading = false;
          _error = e is ApiFailure ? e.message : 'Could not load your inbox.';
        });
      }
    }
  }

  Future<void> _select(Feed feed) async {
    final revision = ++_loadRevision;
    await widget.playback.stop();
    if (!mounted || revision != _loadRevision) return;
    setState(() {
      _feed = feed;
      _loading = true;
      _items = [];
      _error = null;
      _cursor = null;
      _pagination = FeedPagination();
      _more = false;
    });
    try {
      final page = await widget.api.items(feed.id);
      if (!mounted || revision != _loadRevision) return;
      final ready = newPlayableItems(const [], page.items);
      var resumeSegment = 0;
      var resumePosition = Duration.zero;
      if (ready.isNotEmpty) {
        final progress = await optionalResume(widget.api, ready.first);
        if (!mounted || revision != _loadRevision) return;
        resumeSegment = progress?.segment ?? 0;
        resumePosition = Duration(milliseconds: progress?.positionMs ?? 0);
      }
      _pagination.accept(requested: null, next: page.nextCursor);
      setState(() {
        _items = ready;
        _cursor = page.nextCursor;
        _loading = false;
      });
      await widget.playback.replaceFeed(
        ready,
        initialSegment: resumeSegment,
        initialPosition: resumePosition,
      );
      if (_cursor != null && ready.length < 3) unawaited(_nextPage());
    } catch (e) {
      if (mounted && revision == _loadRevision) {
        setState(() {
          _loading = false;
          _error = e is ApiFailure ? e.message : 'Could not load this space.';
        });
      }
    }
  }

  Future<void> _nextPage() async {
    if (_cursor == null || _more || _feed == null) return;
    _more = true;
    final revision = _loadRevision;
    final requestedCursor = _cursor;
    try {
      final page = await widget.api.items(_feed!.id, cursor: requestedCursor);
      if (!mounted || revision != _loadRevision) return;
      _pagination.accept(requested: requestedCursor, next: page.nextCursor);
      final additional = newPlayableItems(_items, page.items);
      setState(() {
        _items = [..._items, ...additional];
        _cursor = page.nextCursor;
      });
      final hadSelection = widget.playback.current != null;
      widget.playback.appendItems(additional);
      if (!hadSelection && _items.isNotEmpty) await widget.playback.select(0);
    } catch (e) {
      if (mounted && revision == _loadRevision) {
        setState(() => _error = 'Could not load more briefings.');
      }
    } finally {
      if (revision == _loadRevision) _more = false;
    }
    if (mounted &&
        revision == _loadRevision &&
        _cursor != null &&
        _items.length < 3 &&
        _error == null) {
      unawaited(_nextPage());
    }
  }

  Future<void> _spaces() async {
    final selected = await showModalBottomSheet<Feed>(
      context: context,
      showDragHandle: true,
      builder: (context) => SafeArea(
        child: ListView(
          shrinkWrap: true,
          children: [
            const Padding(
              padding: EdgeInsets.all(24),
              child: Text(
                'Your spaces',
                style: TextStyle(fontSize: 24, fontWeight: FontWeight.w600),
              ),
            ),
            for (final feed in _feeds)
              ListTile(
                leading: Icon(
                  feed.kind == 'personal'
                      ? Icons.inbox_outlined
                      : Icons.people_outline,
                ),
                title: Text(feed.name),
                selected: feed.id == _feed?.id,
                trailing: feed.id == _feed?.id ? const Icon(Icons.check) : null,
                onTap: () => Navigator.pop(context, feed),
              ),
          ],
        ),
      ),
    );
    if (selected != null && selected.id != _feed?.id) await _select(selected);
  }

  Future<void> _markRead(Briefing item) async {
    try {
      await widget.api.markRead(item);
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('Marked as read for you.')));
    } catch (error) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            error is ApiFailure
                ? error.message
                : 'Read status could not be confirmed. Refresh before trying again.',
          ),
        ),
      );
    }
  }

  Future<void> _details(Briefing item) async {
    _routeVisibility(false);
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      builder: (context) => DraggableScrollableSheet(
        expand: false,
        initialChildSize: .8,
        builder: (context, scroll) => DefaultTabController(
          length: 2,
          child: Column(
            children: [
              Padding(
                padding: const EdgeInsets.all(20),
                child: Text(
                  item.title,
                  style: Theme.of(context).textTheme.headlineSmall,
                ),
              ),
              _MarkReadButton(onMark: () => _markRead(item)),
              const TabBar(
                tabs: [
                  Tab(text: 'Original'),
                  Tab(text: 'Briefing'),
                ],
              ),
              Expanded(
                child: TabBarView(
                  children: [
                    for (final text in [item.source, item.script])
                      SingleChildScrollView(
                        padding: const EdgeInsets.all(24),
                        child: SelectableText(
                          text.isEmpty
                              ? 'No text accompanies this recording.'
                              : text,
                          style: const TextStyle(fontSize: 17, height: 1.6),
                        ),
                      ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
    if (mounted) _routeVisibility(true);
  }

  @override
  void dispose() {
    widget.playback.location.removeListener(_playbackChanged);
    _pages.dispose();
    WidgetsBinding.instance.removeObserver(this);
    unawaited(_progress.close());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(
      leading: const Padding(padding: EdgeInsets.all(12), child: ProsodyMark()),
      title: TextButton.icon(
        onPressed: _spaces,
        icon: const Icon(Icons.expand_more),
        label: Text(_feed?.name ?? 'Your inbox'),
      ),
      actions: [
        IconButton(
          tooltip: 'Briefing activity',
          onPressed: _feed == null
              ? null
              : () async {
                  final feed = _feed!;
                  _routeVisibility(false);
                  try {
                    await Navigator.of(context).push<void>(
                      MaterialPageRoute<void>(
                        builder: (_) =>
                            ActivityScreen(api: widget.api, feed: feed),
                      ),
                    );
                  } finally {
                    if (mounted) _routeVisibility(true);
                  }
                },
          icon: const Icon(Icons.inbox_outlined),
        ),
        IconButton(
          tooltip: 'Space settings',
          onPressed: _feed == null
              ? null
              : () async {
                  _routeVisibility(false);
                  await Navigator.of(context).push(
                    MaterialPageRoute<void>(
                      builder: (_) =>
                          SettingsScreen(api: widget.api, feed: _feed!),
                    ),
                  );
                  if (mounted) _routeVisibility(true);
                },
          icon: const Icon(Icons.tune),
        ),
        PopupMenuButton<String>(
          onSelected: (value) async {
            if (value == 'logout') {
              await widget.playback.stop();
              await widget.api.session.signOut();
              widget.onSignOut();
            }
          },
          itemBuilder: (_) => [
            const PopupMenuItem(value: 'logout', child: Text('Sign out')),
          ],
        ),
      ],
    ),
    bottomNavigationBar: ValueListenableBuilder<String?>(
      valueListenable: _progress.failure,
      builder: (context, failure, _) => failure == null
          ? const SizedBox.shrink()
          : SafeArea(
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(failure),
                    TextButton(
                      onPressed: _progress.resumeSync,
                      child: const Text('Use this device’s position'),
                    ),
                  ],
                ),
              ),
            ),
    ),
    body: _loading
        ? const Center(child: CircularProgressIndicator())
        : _items.isEmpty
        ? Center(
            child: Padding(
              padding: const EdgeInsets.all(32),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const ProsodyMark(size: 72),
                  const SizedBox(height: 28),
                  Text(
                    _error ?? 'A little quiet, for now.',
                    textAlign: TextAlign.center,
                    style: Theme.of(context).textTheme.headlineMedium,
                  ),
                  const SizedBox(height: 16),
                  Text(
                    _error == null
                        ? 'Your agents’ next briefings will arrive here. Check activity for briefings still being prepared.'
                        : 'Your place is safe. Try loading this space again.',
                    textAlign: TextAlign.center,
                  ),
                  const SizedBox(height: 24),
                  OutlinedButton(
                    onPressed: () =>
                        unawaited(_feed == null ? _load() : _select(_feed!)),
                    child: const Text('Refresh'),
                  ),
                ],
              ),
            ),
          )
        : PageView.builder(
            controller: _pages,
            scrollDirection: Axis.vertical,
            itemCount: _items.length,
            onPageChanged: (index) {
              if (widget.playback.location.value.item != index) {
                unawaited(widget.playback.select(index));
              }
              if (index >= _items.length - 3) unawaited(_nextPage());
            },
            itemBuilder: (context, index) => _BriefingCard(
              item: _items[index],
              handler: widget.playback,
              active: index == widget.playback.location.value.item,
              onDetails: () => _details(_items[index]),
            ),
          ),
  );
}

class _BriefingCard extends StatelessWidget {
  final Briefing item;
  final ProsodyPlayback handler;
  final bool active;
  final VoidCallback onDetails;
  const _BriefingCard({
    required this.item,
    required this.handler,
    required this.active,
    required this.onDetails,
  });
  @override
  Widget build(BuildContext context) => SafeArea(
    top: false,
    child: Padding(
      padding: const EdgeInsets.fromLTRB(28, 20, 28, 16),
      child: StreamBuilder<PlaybackState>(
        stream: handler.playbackState,
        builder: (context, snapshot) {
          final state = snapshot.data;
          final segment =
              item.segments[active ? handler.location.value.segment : 0];
          final controller = active ? handler.video.value : null;
          return Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                item.agent.toUpperCase(),
                style: const TextStyle(
                  letterSpacing: 2,
                  fontSize: 11,
                  fontWeight: FontWeight.w600,
                ),
              ),
              const SizedBox(height: 16),
              Text(
                item.title,
                style: Theme.of(context).textTheme.headlineLarge,
                maxLines: 4,
                overflow: TextOverflow.ellipsis,
              ),
              Expanded(
                child: Center(
                  child: controller != null
                      ? AspectRatio(
                          aspectRatio: controller.value.aspectRatio,
                          child: VideoPlayer(controller),
                        )
                      : segment.images.isNotEmpty
                      ? _ImageGallery(
                          key: ValueKey(
                            '${item.id}/${item.renditionId}/${segment.id}',
                          ),
                          item: item,
                          segment: segment,
                          handler: handler,
                        )
                      : ListeningWave(
                          playing: active && (state?.playing ?? false),
                        ),
                ),
              ),
              if (active && handler.failure.value != null)
                Text(
                  handler.failure.value!,
                  style: TextStyle(color: Theme.of(context).colorScheme.error),
                ),
              if (segment.kind == 'image')
                Center(
                  child: FilledButton(
                    onPressed: active ? handler.advanceVisual : null,
                    child: const Text('Continue'),
                  ),
                )
              else ...[
                Slider(
                  value: active
                      ? handler.position.inMilliseconds.toDouble().clamp(
                          0,
                          segment.duration.inMilliseconds.toDouble(),
                        )
                      : 0,
                  max: segment.duration.inMilliseconds > 0
                      ? segment.duration.inMilliseconds.toDouble()
                      : 1,
                  onChanged: active
                      ? (v) => unawaited(
                          handler.seekSegment(
                            Duration(milliseconds: v.round()),
                          ),
                        )
                      : null,
                  label: active ? _time(handler.position) : null,
                ),
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Text(active ? _time(handler.position) : '0:00'),
                    Text(_time(segment.duration)),
                  ],
                ),
                Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    IconButton(
                      tooltip: 'Back 15 seconds',
                      icon: const Icon(Icons.replay_10),
                      onPressed: active
                          ? () => unawaited(
                              handler.seekSegment(
                                handler.position - const Duration(seconds: 15),
                              ),
                            )
                          : null,
                    ),
                    const SizedBox(width: 24),
                    IconButton.filled(
                      iconSize: 40,
                      tooltip: state?.playing == true ? 'Pause' : 'Play',
                      onPressed: active
                          ? () => unawaited(
                              state?.playing == true
                                  ? handler.pause()
                                  : handler.play(),
                            )
                          : null,
                      icon: Icon(
                        active && state?.playing == true
                            ? Icons.pause_rounded
                            : Icons.play_arrow_rounded,
                      ),
                    ),
                    const SizedBox(width: 24),
                    IconButton(
                      tooltip: 'Forward 15 seconds',
                      icon: const Icon(Icons.forward_10),
                      onPressed: active
                          ? () => unawaited(
                              handler.seekSegment(
                                handler.position + const Duration(seconds: 15),
                              ),
                            )
                          : null,
                    ),
                  ],
                ),
              ],
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  TextButton.icon(
                    onPressed: onDetails,
                    icon: const Icon(Icons.notes, size: 18),
                    label: const Text('Read'),
                  ),
                  const Text(
                    'Swipe up for next',
                    style: TextStyle(fontSize: 12),
                  ),
                ],
              ),
            ],
          );
        },
      ),
    ),
  );
  String _time(Duration d) =>
      '${d.inMinutes}:${(d.inSeconds % 60).toString().padLeft(2, '0')}';
}

class _ImageGallery extends StatefulWidget {
  final Briefing item;
  final Segment segment;
  final ProsodyPlayback handler;
  const _ImageGallery({
    super.key,
    required this.item,
    required this.segment,
    required this.handler,
  });
  @override
  State<_ImageGallery> createState() => _ImageGalleryState();
}

class _ImageGalleryState extends State<_ImageGallery> {
  int _page = 0;
  final Set<int> _loaded = {};
  void _recordExposure() {
    if (!_loaded.contains(_page) ||
        ModalRoute.of(context)?.isCurrent != true ||
        widget.handler.current?.id != widget.item.id ||
        widget
                .handler
                .current
                ?.segments[widget.handler.location.value.segment]
                .id !=
            widget.segment.id) {
      return;
    }
    widget.handler.visibleImages.add(
      '${widget.item.id}/${widget.segment.id}/${widget.segment.images[_page]}',
    );
  }

  @override
  Widget build(BuildContext context) {
    _recordExposure();
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Expanded(
          child: PageView.builder(
            key: ValueKey('${widget.item.id}/${widget.segment.id}'),
            itemCount: widget.segment.images.length,
            onPageChanged: (index) {
              setState(() => _page = index);
              _recordExposure();
            },
            itemBuilder: (context, index) => ClipRRect(
              borderRadius: BorderRadius.circular(24),
              child: Image.network(
                widget.segment.images[index].toString(),
                fit: BoxFit.contain,
                frameBuilder: (context, child, frame, synchronous) {
                  if (frame != null || synchronous) {
                    _loaded.add(index);
                    _recordExposure();
                  }
                  return child;
                },
                errorBuilder: (_, __, ___) => const Text('Image unavailable.'),
              ),
            ),
          ),
        ),
        if (widget.segment.images.length > 1)
          Padding(
            padding: const EdgeInsets.only(top: 12),
            child: Text(
              '${_page + 1} of ${widget.segment.images.length} · swipe sideways',
            ),
          ),
      ],
    );
  }
}

// Each explicit action is single-flight; uncertain writes are never retried.
class _MarkReadButton extends StatefulWidget {
  final Future<void> Function() onMark;
  const _MarkReadButton({required this.onMark});
  @override
  State<_MarkReadButton> createState() => _MarkReadButtonState();
}

class _MarkReadButtonState extends State<_MarkReadButton> {
  bool _busy = false;
  Future<void> _mark() async {
    if (_busy) return;
    setState(() => _busy = true);
    try {
      await widget.onMark();
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) => TextButton.icon(
    onPressed: _busy ? null : _mark,
    icon: const Icon(Icons.done),
    label: Text(_busy ? 'Saving…' : 'Mark as read'),
  );
}
