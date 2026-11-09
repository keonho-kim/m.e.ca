# Progress, pending actions and streaming

Use the operation's real scope and source of truth. A spinner says work is pending; a progress value measures known work; a meter describes an existing bounded quantity. Step counts, stage shapes and elapsed display time do not substitute for measured work.

## Give the status the right prominence

A compact action indicator can keep a form usable; a long-running task may need a durable status/result region. Linear and circular shapes use space differently but do not change what the value means. Show phase, scope and actionable uncertainty before decorative animation.

Keep current navigation, selected choice and actual completed work visually distinct. A neutral base can use accents for real running/failed/confirmed states with text and suitable marks. Do not assign a new color to every phase solely for variety or make the final connector an artificial success signal.

## Visual variations

| Possible treatment | Job | Effect and constraint |
| --- | --- | --- |
| Inline Pending | A save/send or local action | Stable text/button with a quiet indicator; retain the draft and unrelated work |
| Linear Determinate | Measured transfer or processing | Track/fill represents actual numerator/denominator and units; display actual phase |
| Circular Determinate | Measured work in compact space | The same real value drives arc and readable value, without duplicate announcements |
| Indeterminate Track / Spinner | The service cannot report an amount | Restrained loop plus meaningful status, not an invented percentage |
| Step Status | A stage-based task reports actual phases | Show queued/running/confirmed/failed and other real states; equal phase counts do not imply equal work |
| Per-item Queue | Upload/import has partial outcomes | Preserve successes, pending identity and supported failed-item recovery |
| Meter | Current used capacity or another bounded quantity | Show real units/range; this is not a task-progress animation |

Do not smooth invented values, run a countdown without a real estimate, or keep an animated 99% while the outcome is unknown. Stable labels and actual scope matter more than a dramatic fill animation. Reduced motion retains static unknown-work status and direct measured updates.

## Truth, cancellation and outcome

Use [Step Status](../components/step-status.md) for actual multi-stage work, separate from a navigable stepper and measured progress. Bind state to the workflow/job owner and expose delayed/unknown observation honestly. Do not make an animation, heartbeat or local stage index the completion authority.

For determinate progress, state what the number measures: bytes uploaded, records validated or items processed. Use backend totals and documented changes to them; expose a changed phase or denominator rather than smoothing fabricated values. Transfer at 100% can still require server processing. Show that phase separately and wait for actual confirmation before offering the finished asset. A counted phase reaching 100% is not proof of whole-job success.

Keep per-item outcomes for uploads and batch work. Preserve successes and retry only supported failed work. If status stops arriving, say that updates are delayed or the connection is lost using the actual service contract. Elapsed time is not a failure, cancel confirmation or permission to replay a mutation. Give a supported status/recovery path for uncertain results.

For AI output, distinguish waiting for the first content, receiving content, disconnected/reconnecting, stopped, failed and finished. Reveal real output as it arrives while preserving reading position and draft input. Do not simulate thoughts, tokens, percentages or logs. A heartbeat proves only transport activity. Cancel/stop UI must reflect what the server actually acknowledges.

## Accessible status

Prefer the project's accessible primitive or native `progress`/`meter` when suitable. A custom progressbar needs an accessible name, real `aria-valuenow` when determinate, and `aria-valuemin`/`aria-valuemax` when the default 0–100 range does not apply; omit `aria-valuenow` when the amount is unknown. Clarify units or phase with visible text and `aria-valuetext` when needed. Put cancellation controls next to the indicator, not inside progressbar semantics. Announce meaningful phase/result changes without speaking every animation frame or token.

A delayed-indicator timer may avoid flicker only under an established policy; it cannot delay ready content, confirmation or recovery. Cancel it on completion/teardown. Clear busy state on all terminal paths. Bound timers, subscriptions and retained event/output buffers with their actual owner.

Sources: [Carbon progress bar](https://www.carbondesignsystem.com/building-blocks/core/components/progress-bar/guidelines), [MUI Progress](https://mui.com/material-ui/react-progress/) and [WAI-ARIA progressbar](https://www.w3.org/TR/wai-aria-1.2/#progressbar).
