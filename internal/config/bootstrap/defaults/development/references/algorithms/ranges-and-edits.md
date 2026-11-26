# Ranges, edits and reusable representations

Choose the representation from the queries and updates the owner actually receives. State whether coordinates are stable, what one revision represents and when derived state becomes obsolete.

## Mapping scroll offsets to variable-size items

- **Use when** a long list or terminal transcript has variable item heights and repeatedly scanning heights to find the visible item dominates layout.
- **Build** cumulative heights to locate an offset by binary search for stable heights. When individual heights change frequently on a stable item order, maintain prefix totals in a Fenwick tree. Map an offset to the item whose cumulative boundary contains it, and retain the viewport anchor by stable item identity plus its intra-item offset.
- **Cost and ownership.** Prefix arrays are cheap to scan and build. A Fenwick tree supports point updates and prefix sums in logarithmic time, with monotone totals required for prefix selection. Define zero-height items and boundary ties. Insertions and reorderings change the coordinate map and may justify a rebuild or an existing order-statistics structure.
- **Verify** a height change before the viewport, offscreen updates, zero-height items, append, deletion and reordering. Compare offsets with a direct prefix scan. Apply the [UI interaction guide](../ui/interaction-contracts.md) to retained position and late measurements.

## Finding overlap and availability over time

- **Use when** calendars, worker utilization or resource timelines need overlap counts, occupied duration or range maxima.
- **Build** start/end events and sort them for a static report. Treat a zero-length half-open interval as empty and reject inverted boundaries. Define the shared endpoint behavior explicitly and process endings before starts when reporting concurrency. For repeated online additions over a fixed coordinate set, a segment tree can maintain range-add and range-maximum state. Store real endpoints when reporting elapsed duration.
- **Cost and ownership.** A static sweep takes sorting plus a linear pass. An online range structure adds maintenance and coordinate-mapping cost. Coordinate ranks preserve order, while duration calculations require original values. Handle new boundaries through a defined rebuild or a structure that accepts them. Keep timezone/recurrence expansion with the [calendar workflow](../applications/calendar/app-guide.md).
- **Verify** adjacent intervals, identical endpoints, zero-length and invalid intervals, cancellation and new coordinates. Recheck actual capacity at the [backend transaction boundary](../backend/storage/transactions.md) when an availability query becomes a reservation.

## Incrementally parsing or highlighting changed text

- **Use when** typing or streaming repeatedly reparses a large document while most of it is unchanged.
- **Build** change ranges in a defined coordinate system and retain parser checkpoints at safe boundaries. Restart before the edit from a checkpoint with valid incoming state, then continue until both output and parser state converge with reusable content. For grammars with nonlocal effects, invalidate the affected enclosing scope or use the parser's supported full fallback.
- **Cost and ownership.** Track document revision, grammar/options and checkpoint lifetime. An edit can change the rest of a file, so budget the fallback as well as the usual small region. Distinguish byte offsets, code units and user-perceived character boundaries. For a parsed result, retain source identity and remap ranges after edits.
- **Verify** an inserted opening delimiter, multiline strings/comments, edits at chunk boundaries, Unicode input, deletions and a stale worker result. Compare incremental output with a full parse of the same revision. Use the [UI quality guide](../ui/quality.md) for actual editing and rendering checks.

## Intersecting many filters over stable object IDs

- **Use when** repeated local filtering combines many membership predicates over a reasonably bounded ID set, and rescanning all object fields dominates the interaction.
- **Build** a stable dense position map and a membership bitset per useful predicate. Combine bitsets with intersections or unions, then resolve surviving positions to canonical records. Use sparse sets or a compressed bitmap when the key distribution makes dense storage wasteful. Keep ranking and display order with the result's owner.
- **Cost and ownership.** Include position-map storage, each predicate's capacity and update cost. Publish predicate state and its position map for the same revision. Define how removed positions are retired or reused so old bits cannot identify a new object. Bound the number of retained predicate combinations.
- **Verify** empty and universal predicates, deletions, position reuse, reordered records and changes arriving during a query. Compare with direct predicate evaluation. Use the [backend search guide](../backend/performance/search-and-ranking.md) when the index and authoritative records live behind a service.
