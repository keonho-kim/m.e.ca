# Location, hierarchy and continuation

Choose the relationship: breadcrumb for real ancestors, sidebar for work destinations, tree for genuine nested objects, timeline for chronology and paging for result continuation. Do not mix stage, location and history merely because they can all use arrows or dots.

Derive destinations and hierarchy from real recurring work. Keep the context people need while minimizing unnecessary travel; a sidebar, breadcrumb or extra rail is not required merely because another SaaS product has one.


## Make each scope recognizable

Global destinations, workspace/project identity, task-object navigation and local views answer different questions. Give them distinct location, labels and emphasis. A schema tree, document outline or slide filmstrip can be useful beside the object without becoming another global sidebar.

Choose top or side navigation from destination depth, repeated access, label length and available work width. There is no universal destination-count threshold. Secondary administration may be contextual, while frequent core destinations should remain visible and discoverable.

Minimal chrome must retain current scope and return context. Use stable selected/current cues, readable labels and deliberate overflow instead of several identical pill rails. A large editorial detail can reduce chrome while preserving a clear path back to its collection.

## Adaptable treatments and optional motion

| Possible treatment | Job | Effect and constraint |
| --- | --- | --- |
| Active Rail | Stable repeated destinations | Move/fade a current marker without moving labels or keyboard focus |
| Group Disclosure | A substantial navigation set has real groups | Expand the relevant branch with retained open state and meaningful current destination |
| Compact Sidebar | More canvas is needed | Collapse labels only with keyboard-discoverable names and a usable expand control |
| Branch Reveal | Actual lazy hierarchy | Open only the requested branch; show local pending/error and preserve object identity |
| Breadcrumb Overflow | A deep real hierarchy exceeds width | Retain useful current/parent context and offer operable hidden ancestors; arrow shape alone is insufficient |
| Timeline Detail | Actual event history needs explanation | Reveal per-event detail with timestamps and pending/occurred/planned meanings kept distinct |
| Scoped Continuation | More results under a stable query | Show boundary feedback while retaining existing rows, scope and return position |

Hamburger Morph may reflect the open state of a Side Drawer at narrow widths or for secondary navigation. Keep a real labeled button and actual aria-expanded/aria-controls. Choose modal versus persistent behavior, retain Escape/close/focus return, and keep primary desktop destinations visible when space allows. The morph presents state; it never owns navigation or permission.

## Navigation contract

Use real links for destinations and separate expand/select controls when both exist. Ordinary nested navigation lists are often simpler than an ARIA tree; use a tree only when its object interaction and keyboard model are needed. Keep expanded state, current route and selected object as distinct meanings.

Breadcrumbs reflect ancestors, not visit history. Browser back is not necessarily parent. Hide redundant top-level breadcrumbs rather than inventing hierarchy. A collapsed sidebar must still reveal useful labels on keyboard focus; an unexplained icon strip is not enough.

Query, filters, sort, page/cursor and selection follow the project's navigation model. Cursor continuation need not invent arbitrary page numbers or an exact last page. Reset dependent paging on a changed query and handle invalid last-page state after deletion through the supported API. Keep search text distinct from applied query and reject obsolete responses.

Choose paging, load-more or infinite continuation by revisiting, comparison and scale. An infinite feed needs meaningful boundaries, accessible continuation and bounded retention; it is not the default for an operational table. Do not hijack the vertical wheel to create an unnecessary horizontal journey.

## Motion and checks

Current location updates independently of a moving marker. Open branches cannot strand focus in collapsed content; route changes restore appropriate destination focus and scroll. A late lazy-child response cannot attach children to a different branch. Reduced motion keeps static location and immediate disclosure/continuation.

Test deep links/back, group collapse while focused, long Korean labels, account/permission changes, failed lazy branch, stale query, invalid last page and return from detail. Reuse existing router and navigation primitives rather than a second history model. Release observers, subscriptions, retained pages and effects with their owner.

Sources: [WAI APG breadcrumb](https://www.w3.org/WAI/ARIA/apg/patterns/breadcrumb/) and [tree view](https://www.w3.org/WAI/ARIA/apg/patterns/treeview/).
