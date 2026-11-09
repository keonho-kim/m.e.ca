# Forms, dates, files and commitment

Start from the selected editing/settings layout. Group related decisions with a heading and help that changes the choice. A field keeps its persistent label, control, requirement/unit and associated error. Two-column desktop forms need related short values and a coherent reading/keyboard order; wide canvas alone is not a reason.

Determine which decisions must be compared, entered together or depend on earlier input. Adapt grouping and summary around that relationship; shorter screens or fewer visible fields do not justify additional steps, repeated navigation or hidden consequences.


## Establish a readable decision sequence

Group fields by the decisions people make, not by an arbitrary two-column grid. Use section headings and more separation between groups than within a group. Shared input-group spacing should behave consistently across settings and editors without importing an unrelated whole page style.

Keep decisive summary information where it can be consulted during input. A long explanatory preview may deserve another mode; a compact quote can remain alongside the form. Choose that tradeoff before hiding sections or adding stages.

Use accent to identify the actual commitment, selected choice and errors with their own roles. Do not give every field group a primary-colored card. A neutral form can have clear hierarchy through type, alignment, boundaries and consistent local actions. Reflow at narrow widths in reading order, retaining labels, units and recovery.

## Adaptable compositions and optional motion

| Possible treatment | Useful task | Motion and constraint |
| --- | --- | --- |
| Grouped Editor | Related values in a focused task | Quiet group boundaries; no compulsory entrance on every field |
| Aligned Settings | Dense Korean desktop label/value work | Consistent label track and readable controls; long labels wrap without crushing input |
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

Files need a keyboard chooser beside drag/drop, supported service types/limits, real file identity and applicable per-file result/removal. Reserve preview geometry and release object URLs/readers. Transfer at 100% may still require processing; keep successes after partial failure and retry only supported failed work. Sensitive input, credentials and file contents cannot become debug/demo logs.

Reduced motion uses immediate dependent groups and static pending/result cues. Test invalid/unknown responses, concurrent edits, unsaved close, dependent changes, timezone/range boundaries, failed previews and partial upload. Check keyboard/IME, long Korean text and zoom; inspect cleanup of requests, previews and observers. A frontend-only fixture cannot establish server authorization or transaction safety.

Sources: [WAI forms notifications](https://www.w3.org/WAI/tutorials/forms/notifications/) and [WAI forms grouping](https://www.w3.org/WAI/tutorials/forms/grouping/). Reuse the application's existing form library or native controls before adding a dependency.
