# Forms, dates, files and commitment

Start from the selected editing/settings layout. Group related decisions with a heading and help that changes the choice. A field keeps its persistent label, control, requirement/unit and associated error. Two-column desktop forms need related short values and a coherent reading/keyboard order; wide canvas alone is not a reason.

Determine which decisions must be compared, entered together or depend on earlier input. Adapt grouping and summary around that relationship; shorter screens or fewer visible fields do not justify additional steps, repeated navigation or hidden consequences.

## Choose the commitment model

| Variant | Composition and behavior |
| --- | --- |
| Staged form | Related edits share Save and Cancel. Keep the draft and saved value distinguishable and protect useful work on close. |
| Autosave field | A local edit persists independently. Show the actual saving and saved version near the field. Preserve failed input and expose recovery. |
| Apply filters | Several criteria make one new query. Retain visible draft and applied conditions. Make Apply and Reset effects explicit. |
| Dependent fields | Enabling an option reveals eligible fields. Retain unrelated input and explain values invalidated by the change. |
| Review before commitment | Give a stable summary of the actual consequence. Offer edit and an explicit final action, and report confirmation from its result. |

## Compose inline validation

Keep the visible label, input, decisive hint and field error associated. For initial format errors, show feedback after meaningful interaction, such as editing then leaving the field, or on a submit attempt. Use the platform constraint or the project's validation mechanism. Keep `aria-invalid` and linked error text aligned with the actual field error, and clear them when the value is corrected.

Write specific corrective text and keep it near the input. On failed submission, focus the first invalid field or a meaningful error summary, reveal collapsed content as needed, and preserve all valid input. Remote checks belong to the current field value. Show a pending or retry message when a check cannot finish and avoid announcing the same error on every keystroke. Keep focus, the input and action placement stable when a message appears.

## Establish a readable decision sequence

Group fields by the decisions people make, not by an arbitrary two-column grid. Use section headings and more separation between groups than within a group. Shared input-group spacing should behave consistently across settings and editors without importing an unrelated whole page style.

Keep decisive summary information where it can be consulted during input. A long explanatory preview may deserve another mode; a compact quote can remain alongside the form. Choose that tradeoff before hiding sections or adding stages.

Use accent to identify the actual commitment, selected choice and errors with their own roles. Do not give every field group a primary-colored card. A neutral form can have clear hierarchy through type, alignment, boundaries and consistent local actions. Reflow at narrow widths in reading order, retaining labels, units and recovery.

## Adaptable compositions and optional motion

| Possible treatment | Useful task | Motion and constraint |
| --- | --- | --- |
| Grouped Editor | Related values in a focused task | Quiet group boundaries; no compulsory entrance on every field |
| Aligned Settings | Dense label/value work | Consistent label track and readable controls with room for long labels to wrap |
| Progressive Dependency | One answer genuinely unlocks another group | Reveal dependent fields from real state; preserve valid work and mark invalidated values |
| Sticky Commit | A long editor needs reachable Save/Cancel | Reserve action-bar space and avoid covering final errors/fields; appearance does not change persistence |
| Review Summary | Consequential commitment benefits from checking input | Stable summary plus real submitting/result region; no fake checkmarks |
| Validation Summary | Several errors need a recovery path | Reveal a meaningful summary and linked field explanations; focus intentionally after attempted submit |
| File Queue | A batch upload/import with partial results | Add/remove/reflow identified items; transfer progress and processing result remain separate |

Form motion must not unmount a dirty input, rearrange tab order or remount a field on every validation update. A dependency reveal does not authorize collecting additional sensitive information. A multistage flow needs the stepper's actual stage model rather than extra artificial pages.

## Submission and recovery

Distinguish idle/dirty, submitting, confirmed, field/general error, conflict and uncertain result where applicable. Preserve entered values after failure. Prevent accidental repeated interaction where appropriate, but a disabled Save is not server deduplication. Lost responses can follow successful writes; use the actual status/idempotency contract before replaying side effects.

Keep client/server validation consistent and timing useful. Use field errors near their cause, task errors at the task, and summary links for long forms. Reveal hidden/collapsed invalid sections. Distinguish reset-to-original, clear and discard. Dirty navigation, Escape, close and cancel follow one explicit draft policy; do not warn when nothing changed.

## Dates, ranges and files

Display local formats but preserve actual service meaning: date-only versus instant, timezone, precision, inclusive/exclusive range and unavailable values. Do not invent booking or recurrence rules. Provide a useful typed/keyboard path beside a calendar when suitable, and preserve valid selection after a conflict.

For [file drop](../interactions/drag-and-drop.md#file-drop), provide a named chooser and keep the same service limits and per-file results. Reserve preview geometry and release object URLs and readers. Track actual readiness after transfer. Keep successes after partial failure and retry only eligible failed work. Keep sensitive input, credentials and file contents out of debug and demo logs.

Reduced motion uses immediate changes to dependent groups and static pending and result cues. Test invalid or unknown responses, concurrent edits, unsaved close, dependent changes, timezone and range boundaries, failed previews and partial upload. Check keyboard operation, text composition, long text and zoom. Inspect cleanup of requests, previews and observers. Verify server authorization and transaction behavior at the integration boundary when in scope.

Sources: WAI forms notifications and WAI forms grouping. Reuse the application's existing form library or native controls before adding a dependency.
