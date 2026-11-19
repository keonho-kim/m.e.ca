# Tables, lists and cards

Use a table to compare repeated fields, a list to scan identity with short context, and cards when real imagery/narrative drives independent exploration. Anatomy: query/toolbar, meaningful item fields, clear next action, stable identity, applicable selection, continuation and region states. An operational table does not need to become sparse cards for variety.

Derive fields and arrangement from the comparison or exploration people actually need. Data volume, repeated use and retained context can justify a dense table or a visual collection; a template count of cards, columns or badges cannot.

## Choose the collection's interaction model

| Variant | Composition and behavior |
| --- | --- |
| List or card collection | Give each item a name and distinct inspect and command targets. Keep Tab order aligned with the items. Use cards where preview content has decision value. |
| Sortable table | Keep a table caption and semantic headers. Use buttons in sortable headers and expose the current single-column sort with `aria-sort`. Keep focus on the activated header and update the actual row order. |
| Selectable or expandable row | Give selection and disclosure separate controls. Keep the selected ID, expanded detail and next action distinct. Keep an explicit link for a destination. |
| Editable grid | Give cells an arrow-key navigation model. Use the chosen primitive's Enter or F2 edit path. Inside an editor, standard editing keys operate it. Define commit, cancellation and return to cell focus. |
| [Reorderable list](../interactions/drag-and-drop.md) | Use a handle when row controls or text coexist with drag. Show the proposed insertion point and request a move on a valid drop. Provide named Move controls reachable by keyboard and single pointer, retain focus on the moved item and announce its confirmed position. Escape cancels an active move. |

Preserve object identity through sort and reorder. Keep visual, reading and focus order aligned. Apply any optional move animation after the new order is accepted. For a pending or rejected reorder, make the saved order and recovery clear. The WCAG 2.2 Dragging Movements criterion informs the single-pointer alternative.

## Choose hierarchy inside the collection

Lead with the information used to distinguish items. Comparable records need aligned fields and stable column meanings; a visual library can give images and captions more scale. Keep a complete object name available and place a concise inspect/action target where useful, rather than shrinking names into button-sized fragments.

Use accent for a real selection, resource category or state, with stable meanings and readable text. Several role-based accents can improve scanning on a neutral base. Avoid a saturated badge for every field or a different color for each card position.

Keep selection actions, row navigation and embedded controls distinct. A saved collection and file-type filter may narrow different dimensions; preserve their scope independently. One query can filter one collection without duplicated live tables for each visible filter choice.

## Adaptable treatments and optional motion

| Possible treatment | Job | Effect and constraint |
| --- | --- | --- |
| Stable Rows | Dense comparison | Keep headers/alignment while values arrive; no repeated row entrances on refresh |
| Row Expansion | Occasional secondary details | Expand real row detail with preserved selection and readable column context |
| Master-detail Selection | Frequent object inspection | Update a reserved inspector with correct selected identity and local loading |
| List/Card View | Both comparison and visual discovery are real needs | Preserve IDs, query and selection; optional bounded layout transition instead of duplicate active collections |
| Contextual Toolbar | Selection unlocks bulk operations | Reveal a stable reserved action area and actual scope rather than shifting the whole page |
| Bounded Entry | A small meaningful batch was added | Brief local fade/layout reflow; do not stagger hundreds of records or replay on every virtualized mount |
| Live Update Cue | New/changed real records need notice | Subtle temporary highlight with text/context; preserve reading position and avoid fake liveness |

## Data and interaction

Establish field priority, readable names, units, numeric alignment and intentional widths. Sorting needs actual direction, scope and accessible state. Give wide tables deliberate horizontal scrolling and sticky behavior where needed. Use semantic tables by default. Choose an interactive grid when the task needs cell-level interaction and the corresponding keyboard model.

Keep row navigation distinct from checkbox/menu actions. Whole-card links cannot contain conflicting interactive elements. Selected, focused, unread, pending and error are different meanings, not random badge colors. Sticky regions must leave focused controls and the last row reachable.

Selection uses stable IDs with explicit page/IDs/all-matching scope. Changing the query must preserve or reset scope deliberately before an action. Useful density/column/export controls depend on the actual task and API; a table template does not authorize unsupported services.

Initial loading uses known row/card geometry with useful headers retained. Refetch retains safely identified content; continuation appends boundary feedback. Empty, no matches, denied, failed and stale are distinct. Null/unknown is not a fabricated zero. Actual counts and continuation semantics come from the API.

## Resources and checks

Virtualize only for measured rendering volume, preserving selection, focus, restoration and meaningful table context. Do not mount invisible placeholder rows or retain every historical page. Avoid expensive layout choreography while scrolling or measuring every cell on each animation frame.

Logical selection/query/state stays authoritative during effects. Reduced motion keeps direct state and layout changes. Test stale results, long text, null/large values, deletion on the last page, partial batch result, selection after refresh, focused offscreen rows and return from detail. Cleanup owns observers, requests, retained collections and animation handles.

Sources: Carbon data table, WAI APG table and grid.
