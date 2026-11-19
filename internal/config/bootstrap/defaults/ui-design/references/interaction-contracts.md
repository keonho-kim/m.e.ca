# Make state understandable and continuous

Use this when a design depends on changing content, input, selection or work. Read the actual product's behavior and service contracts when integration is in scope. The guide defines what the interface must communicate; a sample controller or client-side state store is not a required architecture.

## Define visible meanings before effects

| Question | Design consequence |
| --- | --- |
| What object and scope are current? | Identity, applied query, environment and selection stay recognizable |
| What can be edited or chosen? | Labels, eligibility, local draft and committed value differ where needed |
| What is pending? | Feedback belongs to the actual region/action, with useful work retained |
| What is confirmed, partial or unknown? | Result and recovery cannot be inferred from a timer or connection state |
| What survives leaving/returning? | Relevant draft, query, selection, position and result context have a deliberate policy |
| What action is safe next? | Clear retry, reconcile, edit, return or cancellation meaning |

Use only states the feature can encounter. A design shell can show representative arrangements; it need not build a service or simulate background activity. A functional frontend must follow the actual contract rather than leave a polished control pretending to work.

## Interactive states

| State | What to show and preserve |
| --- | --- |
| Rest and hover | Make the label, purpose and target identifiable at rest. Hover can add local emphasis. Reveal hover actions on keyboard focus and provide them directly on touch. |
| Keyboard focus | Keep a visible `:focus-visible` cue or the native ring. It survives hover, pressed, selected and error styling and remains clear at zoom and in forced colors. |
| Pressed and dragging | Give immediate contact feedback in a stable hit area. Press feedback lasts for the gesture. A drag shows the real target or insertion point and has a cancellation and non-drag path. |
| Selected, checked or pressed toggle | Show the current value with a mark, boundary or text as well as color. Expose the state that matches the control's semantics and preserve an independent focus indicator. |
| Expanded or current | Mirror the actual open region or current destination. Keep labels and content reachable, and restore useful focus when a region closes. |
| Disabled or read-only | Explain a consequential prerequisite. Disabled prevents activation. Where a focusable `aria-disabled` control is appropriate, block activation in behavior too. Keep a read-only value available to inspect or copy where useful. |
| Pending, success or uncertain | Name the affected action. Keep the last confirmed value and pending intent legible. Show confirmation from the outcome and give uncertainty a recovery path. |
| Invalid or failed | Show specific text near the cause with a next step. Keep the user's current focus or deliberately move it after a failed submit. Preserve recoverable input. |

Use combinations the feature can encounter. For example, a selected row with a pending change still shows its selection and focused control. A state cue can appear immediately with its final geometry. Motion may reinforce it while focus, selection and announcements continue to reflect actual state.

## Collections and evidence

Distinguish edited search/filter values from the conditions that produced current results. Several dimensions may narrow the same collection independently: workspace, saved scope, category, query and sort. Keep the relevant context instead of resetting another dimension accidentally.

Selection follows stable identity, not a displayed position or label. Detail, citations and previews must resolve to the actual selected object. Older responses must not repaint newer scope or show A's content as B. Count and continuation semantics come from the real source; unknown is not zero.

Choose retained-content refresh, local loading or clearing from identity and safety. Keeping old work can preserve continuity only when it is labeled and still appropriate. Use [loading guidance](loading-and-progress.md) for the visual representation.

## Input and commitment

Keep the question, current value, draft and saved state readable. Read-only, disabled, unresolved and invalid have different meanings. Respect IME, caret/selection and valid partial input. Validation explains a correctable cause near its field/task and preserves entered work.

An apparent success is a claim. Confirm it only from the supported outcome. A local selection does not complete a workflow, an optimistic change is still pending and a lost response can follow a completed write. A meaningful uncertain state may need durable identity/status lookup rather than automatic retry.

Define what close, Escape, navigation and cancel do to dirty or pending work. Use proportional review and recovery; do not interrupt unchanged or ordinary reversible work by habit. Actual authorization, concurrency and persistence belong to their real service when implemented, not a disabled UI or animation callback.

## Ongoing, partial and streamed work

Separate first wait, arriving content, observation loss/reconnect and actual task outcome. A heartbeat conveys transport activity, not progress or completion. Streaming text may be partial while its artifact/task continues. Preserve reading position, draft and useful current controls.

A transfer percentage describes its phase; processing and availability may follow it. Batch work can have confirmed successes and failed/unknown items. Keep useful results, show scoped recovery and avoid replaying completed side effects. Cancellation request, acknowledged cancellation and stopped visual motion are separate events.

## Keep ownership proportional

Reuse the project's router, components and state/service mechanisms. Share stable rules across actual consumers; keep local presentation with its region. Avoid duplicate live forms/collections solely for animation or separate data models for alternate views.

Bound requests, subscriptions, retained output, previews and transient effects with an explicit owner. Release them when their work or view ends. Do not introduce caching, polling or a new transport merely to copy a study. Test relevant interruption, stale data, failure and return through a real path; a diagram of states or screenshot cannot prove the integration.
