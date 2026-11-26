# Command palette

Use a palette for repeated access to actual commands/destinations across a substantial workspace. It supplements discoverable navigation/actions; a shortcut-only interface is not sufficient. Keep record search, command execution and value selection distinct when their outcomes differ.

Anatomy: named dialog/surface, search input, meaningful grouped results, current candidate, optional real shortcut, unavailable/empty/error states and dismissal. A highlighted command is not executed. Context and permission must be current at execution.

Judge the palette by real command volume and repeated-work needs. It can accelerate learned tasks while visible routes still support discovery; do not invent commands, shortcuts or a global search service just to reproduce a polished application shell.

## Choose what a result does

| Variant | Composition and behavior |
| --- | --- |
| Global commands | A named trigger and optional shortcut open a search field and grouped commands. Move the active candidate with arrows and explicitly activate it to execute. |
| Contextual commands | Show the current target and scope alongside the query. If the target changes, refresh or dismiss the candidates before activation. |
| Object switcher or search | Show distinguishing record metadata. Selecting a destination follows normal navigation. Selecting a field value uses a picker and its commit path. |

Keep the caret and input editing keys available in a searchable surface. A listbox suggestion popup can expose the active option while focus stays in the input. When results contain independent controls, choose a pattern that lets people reach those controls.

## Keep the palette supplementary and scoped

Make the current search/command scope clear: workspace, selection or global destinations. Group candidates by meaningful outcomes and show enough identity/context to distinguish similar labels. A highlighted candidate is a proposal, not an executed command.

Use stable readable rows, restrained hierarchy and a clear focused candidate. Frequent routes still need visible access outside the palette. Do not build a global command layer merely because it appears in a polished example.

Let the target system own dialog, search and keyboard behavior. Accent can identify the current candidate while labels, shortcut hints and unavailable reasons retain readable priority. Results should become useful immediately rather than waiting through an item-by-item entrance.

| Possible treatment | Useful surface | Motion and constraint |
| --- | --- | --- |
| Quiet Center | A familiar bounded global command surface | Fade/small scale with deliberate initial focus, not a dramatic full-page entrance |
| Anchor Search | A visible header search opens richer commands | Restrained origin continuity; preserve one real input and entered query |
| Grouped Results | Commands and destinations need clear grouping | Local content update without staggering results before they are operable |
| Contextual Scope | Commands depend on current project/selection | Show the actual scope and invalidate obsolete candidates; animation cannot retarget an action |
| Secondary Detail | A result needs explanation before execution | A stable contextual detail region or accessible secondary step, not hidden hover-only instructions |

Use existing dialog plus command/suggestion primitives with their actual focus/keyboard model. Opening has a visible entry as well as an optional supported shortcut. Escape closes the intended layer and focus returns logically. Keyboard movement changes the candidate, and explicit selection executes; typing/composition cannot run commands.

Preserve active text composition and the input caret through result changes. Guard against obsolete responses, unavailable choices and a removed selected record. Scope commands and results to relevant, authorized choices. Execute a command on deliberate activation.

Reduced motion gives immediate open/result state. Rapid open-close or context change cancels stale visual callbacks and queries without restoring old results. Release timers, requests, focus/scroll locks and event handlers on every exit. Test shortcut conflicts, keyboard selection, IME Enter, no matches, failed lookup, permission change and removed opener.

Sources: WAI APG modal dialog and combobox. Use the actual palette primitive's semantics rather than adding one universal invented command-palette role.
