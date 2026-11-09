# Input boxes and text editing

Choose text, textarea, numeric value or search behavior from the actual field meaning. Identifiers/telephone codes are not arithmetic quantities. Anatomy: persistent label, control/value, applicable format/unit/help, validation and clear/read-only/disabled behavior. Placeholder text is an example, not the label. Keep empty, null, zero and unresolved values distinct.

Use actual values, edit behavior and error risks to determine field size, format guidance and help. A compact or animated field can increase effort if it hides its label, unit or recovery. More decorative feedback cannot compensate for an unclear question.


## Fit the question and value

Use field width, label position and grouping to suggest the expected information. Short exact values, search and long prose need different geometry. Align related fields without spreading a small form across an entire monitor. Keep units and decisive help close to the value.

Choose outlined, filled or quieter fields within the target system; minimal or tactile styling still needs a recognizable entry area. Use strong enough functional ink, field edges and focus. Placeholder text does not carry the label, and a decorative success mark cannot stand in for validation.

Keep the caret, composition and partially entered value stable. Help/error changes should not repeatedly move the field or hide the label. Reserve an appropriate message relationship so visual polish does not increase correction effort.

## Adaptable treatments and optional motion

| Possible treatment | Use | Effect and constraint |
| --- | --- | --- |
| Outlined / Filled / Plain | The incumbent system's normal fields | Stable boundaries and label placement; plain material still needs discoverability |
| Focus Ring | Keyboard/pointer entry | Emphasize a real focus boundary without changing field dimensions or losing forced-color visibility |
| Floating Label | Compact fields with an established labeled pattern | Move only the label into a reserved slot; autofill, prefilled and nonempty states cannot overlap it |
| Underline Growth | A restrained supporting focus cue | Extend a line beneath a stable field; do not make the line the sole focus/error signal |
| Help Reveal | Contextual format explanation | Reveal nearby help without covering input or moving it repeatedly during typing |
| Validation Crossfade | Corrected/failed field feedback | Change the real message/state while keeping readable text and programmatic association |
| Search Expansion | A compact search affordance becomes a real input | Expand a reserved region with persistent label and focus; never replace arbitrary page navigation |
| Growing Textarea | Variable prose input | Grow to an intentional bound, then scroll; retain caret/reading position and avoid resize loops |

Do not animate typed characters, caret location or value transforms as decorative output. Avoid shake as the only error message; ordinary field correction should not require a moving target. Keep icons, suffix units, clear/password controls and long Korean labels from colliding.

## Editing and validation contract

Preserve selection, caret, composition and input across re-render/animation. Enter completing Korean IME composition must not also submit search or choose an option. The project's multiline submit/line-break behavior must be explicit; do not copy a chat shortcut into every textarea.

Parse transport values according to actual units, locale and precision. Formatting on every keystroke can destroy partial valid input or move the caret; separate edit representation from committed value where required. Support the existing autofill/password-manager and autocomplete contract, with accessible named clear/reveal actions for sensitive fields when used.

Use meaningful validation timing, such as submission or relevant completed interaction. Do not flash errors on untouched or still-composing values. Keep a specific associated error plus task-level feedback when necessary; update corrected errors according to truth. Read-only may preserve copying/reading, whereas disabled denotes unavailable interaction. Do not use either to disguise an unresolved fetched value.

Remote checks need current field identity and pending/error handling. Debounce reduces frequency but cannot prevent stale validation responses. A success icon needs real current meaning, not a timer after typing. Preserve entered input on failed/unknown save, and never log sensitive field values.

Reduced motion keeps direct focus/state changes and static help. Cleanup owns resize observers, validation requests, debounce timers and effects. Test autofill, prefilled values, long help, null/zero, invalid server response, IME, rapid edits, paste, zoom and forced colors. A beautiful floating label cannot establish a working save or usable screen-reader label.

Sources: [WAI forms labels](https://www.w3.org/WAI/tutorials/forms/labels/) and [MDN input](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/input). Rich remote choice behavior belongs to the dropdown recipe, not a second suggestion engine inside this field.
