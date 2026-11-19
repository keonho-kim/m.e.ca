# Location, hierarchy and continuation

Choose the relationship: breadcrumb for real ancestors, sidebar for work destinations, tree for genuine nested objects, timeline for chronology and paging for result continuation. Do not mix stage, location and history merely because they can all use arrows or dots.

## Choose the navigation pattern

| Pattern | Choose when | Composition and behavior |
| --- | --- | --- |
| Breadcrumb | The current destination has useful ancestors | Use a named navigation landmark and an ordered trail of ancestor links with decorative separators. Give the current page a distinct cue and `aria-current="page"` when linked. At narrow widths, keep the current page and parent visible and give collapsed ancestors a named opener. |
| Hamburger menu | Primary navigation needs a compact entry | Use a real button labeled for navigation with an expanded state. Open a named navigation disclosure or drawer containing real links. Escape or Close returns focus. Choosing a link uses the destination's focus and history behavior. |
| Navigation disclosure | A modest hierarchy groups destination links | Use one expand button and separate links when both actions exist. Keep the normal Tab sequence and show the current link inside its branch. |
| Object tree | Users repeatedly explore nested objects | Keep focus, expansion and selection distinct. Up and Down move among visible nodes. Right expands or enters a branch and Left collapses or returns to the parent in the established pattern. Define what activation opens. |
| Numbered pagination | Known pages can be revisited or linked | Show Previous, Next and useful page numbers with a named current page. Keep ellipses passive unless they open a named picker. Disable unavailable boundaries or omit them consistently. Preserve query, sort and browser return context. |
| Load more or cursor continuation | People need more of a useful existing result set | Append confirmed items through a reachable control and retain scroll and selection. Announce meaningful counts and keep an intelligible end state. For automatic loading, keep a usable continuation and return path. |

A navigation trigger carries real state across entry and dismissal. When a hamburger drawer becomes persistent navigation at a breakpoint, clear modal blocking and keep a sensible focus location. Keep menu entry, close and all destinations reachable without swiping. Use the [drawer and scrim guidance](dialogs-and-drawers.md).

For an in-place page change, keep focus on the pagination control if it stays available and announce the resulting page or range. If the control disappears, use an available neighboring control or the results heading. Apply the new current-page cue only when the corresponding results are shown. A route-based page selection follows the target route's normal focus and browser history.


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

Test deep links and back navigation, group collapse while focused, long labels, account and permission changes, failed lazy branches, stale queries, invalid last pages and return from detail. Reuse the existing router and navigation primitives. Release observers, subscriptions, retained pages and effects with their owner.

Sources: WAI APG breadcrumb and tree view.
