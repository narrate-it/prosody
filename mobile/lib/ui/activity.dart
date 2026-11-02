import 'dart:async';

import 'package:flutter/material.dart';

import '../data/api.dart';
import '../data/activity.dart';
import '../data/models.dart';
import '../data/pagination.dart';

// Uses the proposed /feeds/{feed}/activity route and ReadActivity projection.
// Production composition remains gated. This screen neither starts generation
// nor changes the playable queue. Status updates are fetched explicitly.
class ActivityScreen extends StatefulWidget {
  final ProsodyApi api;
  final Feed feed;

  const ActivityScreen({super.key, required this.api, required this.feed});

  @override
  State<ActivityScreen> createState() => _ActivityScreenState();
}

class _ActivityScreenState extends State<ActivityScreen> {
  static const _maxPages = 20;
  static const _maxItems = 500;

  List<ActivitySummary> _items = [];
  FeedPagination _pagination = FeedPagination();
  String? _cursor, _error;
  bool _loading = true, _more = false, _limited = false;
  int _generation = 0, _pageCount = 0;

  @override
  void initState() {
    super.initState();
    unawaited(_refresh());
  }

  Future<void> _refresh() async {
    final generation = ++_generation;
    setState(() {
      _loading = true;
      _more = false;
      _error = null;
    });
    try {
      final page = await widget.api.activity(widget.feed.id);
      if (!mounted || generation != _generation) return;
      final pagination = FeedPagination();
      pagination.accept(requested: null, next: page.nextCursor);
      final items = _merge(const [], page.items);
      setState(() {
        _items = items.take(_maxItems).toList();
        _pagination = pagination;
        _cursor = page.nextCursor;
        _pageCount = 1;
        _limited =
            items.length > _maxItems ||
            (_items.length == _maxItems && _cursor != null);
      });
    } catch (error) {
      if (mounted && generation == _generation) {
        setState(() {
          _error = error is ApiFailure
              ? error.message
              : 'Activity could not be loaded. Please try again.';
        });
      }
    } finally {
      if (mounted && generation == _generation) {
        setState(() => _loading = false);
      }
    }
  }

  Future<void> _loadMore() async {
    if (_loading || _more || _limited || _cursor == null) return;
    final generation = _generation;
    final cursor = _cursor;
    setState(() {
      _more = true;
      _error = null;
    });
    try {
      final page = await widget.api.activity(widget.feed.id, cursor: cursor);
      if (!mounted || generation != _generation) return;
      _pagination.accept(requested: cursor, next: page.nextCursor);
      final items = _merge(_items, page.items);
      setState(() {
        _items = items.take(_maxItems).toList();
        _cursor = page.nextCursor;
        _pageCount++;
        _limited =
            items.length > _maxItems ||
            (_cursor != null &&
                (_pageCount >= _maxPages || _items.length >= _maxItems));
      });
    } catch (error) {
      if (mounted && generation == _generation) {
        setState(() {
          _error = error is ApiFailure
              ? error.message
              : 'More activity could not be loaded. Please try again.';
        });
      }
    } finally {
      if (mounted && generation == _generation) {
        setState(() => _more = false);
      }
    }
  }

  // Repeated item IDs update their status/source in place rather than creating
  // a duplicate card. List order remains the order supplied by the service.
  List<ActivitySummary> _merge(
    List<ActivitySummary> previous,
    List<ActivitySummary> incoming,
  ) {
    final indexed = <String, ActivitySummary>{};
    for (final item in [...previous, ...incoming]) {
      indexed[item.id] = item;
    }
    return indexed.values.toList();
  }

  void _readOriginal(ActivitySummary item) {
    unawaited(
      Navigator.of(context).push<void>(
        MaterialPageRoute<void>(
          builder: (context) => Scaffold(
            appBar: AppBar(title: const Text('Original')),
            body: SafeArea(
              child: SingleChildScrollView(
                padding: const EdgeInsets.all(24),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Semantics(
                      header: true,
                      child: Text(
                        item.title,
                        style: Theme.of(context).textTheme.headlineMedium,
                      ),
                    ),
                    const SizedBox(height: 12),
                    const SizedBox(height: 28),
                    SelectableText(
                      item.source,
                      style: const TextStyle(fontSize: 17, height: 1.6),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(
      title: const Text('Activity'),
      actions: [
        IconButton(
          tooltip: 'Refresh activity',
          onPressed: _loading ? null : _refresh,
          icon: const Icon(Icons.refresh),
        ),
      ],
    ),
    body: SafeArea(
      top: false,
      child: RefreshIndicator(
        onRefresh: _refresh,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(24, 16, 24, 32),
          physics: const AlwaysScrollableScrollPhysics(),
          children: [
            Text(
              widget.feed.name,
              style: Theme.of(context).textTheme.headlineMedium,
            ),
            const SizedBox(height: 12),
            const Text(
              'A place to check on your briefings, even before they’re ready.',
            ),
            const SizedBox(height: 24),
            if (_loading)
              const Padding(
                padding: EdgeInsets.only(bottom: 20),
                child: LinearProgressIndicator(
                  semanticsLabel: 'Loading activity',
                ),
              ),
            if (_error != null) ...[
              Semantics(
                liveRegion: true,
                child: Text(
                  _error!,
                  style: TextStyle(color: Theme.of(context).colorScheme.error),
                ),
              ),
              if (_items.isNotEmpty)
                const Padding(
                  padding: EdgeInsets.only(top: 8),
                  child: Text('Showing the activity already loaded.'),
                ),
              Align(
                alignment: Alignment.centerLeft,
                child: TextButton.icon(
                  onPressed: _loading ? null : _refresh,
                  icon: const Icon(Icons.refresh),
                  label: const Text('Refresh activity'),
                ),
              ),
              const SizedBox(height: 16),
            ],
            if (!_loading && _error == null && _items.isEmpty) ...[
              const Icon(Icons.inbox_outlined, size: 40),
              const SizedBox(height: 20),
              Text(
                'No briefings here yet.',
                textAlign: TextAlign.center,
                style: Theme.of(context).textTheme.titleLarge,
              ),
              const SizedBox(height: 12),
              const Text(
                'Accepted briefings will appear here while they’re being prepared.',
                textAlign: TextAlign.center,
              ),
            ],
            for (final item in _items)
              _ActivityCard(
                key: ValueKey('activity-${item.id}'),
                item: item,
                onRead: () => _readOriginal(item),
              ),
            if (_limited) ...[
              const SizedBox(height: 16),
              const Text(
                'This view is limited to 20 pages or 500 briefings. More activity may be available.',
              ),
            ] else if (_cursor != null) ...[
              const SizedBox(height: 16),
              OutlinedButton(
                onPressed: _loading || _more ? null : _loadMore,
                child: Text(_more ? 'Loading more…' : 'Load more activity'),
              ),
            ] else if (_items.isNotEmpty && !_loading && _error == null) ...[
              const SizedBox(height: 16),
              Text(
                'All activity returned for this space is shown.',
                style: Theme.of(context).textTheme.bodySmall,
              ),
            ],
          ],
        ),
      ),
    ),
  );
}

class _ActivityCard extends StatelessWidget {
  final ActivitySummary item;
  final VoidCallback onRead;

  const _ActivityCard({super.key, required this.item, required this.onRead});

  @override
  Widget build(BuildContext context) {
    final (label, explanation, icon) = switch (item.status) {
      'processing' => (
        'Preparing',
        item.source.isEmpty
            ? 'Your briefing is being prepared. Refresh to check for an update.'
            : 'Your briefing is being prepared. You can read the original while you wait.',
        Icons.hourglass_top_rounded,
      ),
      'failed' => (
        'Couldn’t prepare briefing',
        item.source.isEmpty
            ? 'A detailed reason is not available. Refresh to check for an update.'
            : 'A detailed reason is not available. Read the original or refresh to check for an update.',
        Icons.error_outline,
      ),
      'ready' => (
        'Ready',
        'This briefing is ready in your space’s listening feed.',
        Icons.check_circle_outline,
      ),
      'deleted' => (
        'Removed',
        'This briefing has been removed.',
        Icons.remove_circle_outline,
      ),
      _ => (
        'Unavailable',
        'This briefing is not available to play.',
        Icons.info_outline,
      ),
    };
    final canRead =
        const ['processing', 'failed', 'ready'].contains(item.status) &&
        item.source.isNotEmpty;
    final color = item.status == 'failed'
        ? Theme.of(context).colorScheme.error
        : Theme.of(context).colorScheme.primary;
    return Card(
      elevation: 0,
      margin: const EdgeInsets.only(bottom: 16),
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                ExcludeSemantics(child: Icon(icon, size: 20, color: color)),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    label,
                    style: TextStyle(color: color, fontWeight: FontWeight.w600),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 16),
            Semantics(
              header: true,
              child: Text(
                item.title,
                style: Theme.of(context).textTheme.titleLarge,
              ),
            ),
            const SizedBox(height: 6),
            const SizedBox(height: 12),
            Text(explanation),
            if (canRead) ...[
              const SizedBox(height: 12),
              TextButton.icon(
                onPressed: onRead,
                icon: const Icon(Icons.notes, size: 18),
                label: const Text('Read original'),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
