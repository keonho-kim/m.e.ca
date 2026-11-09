# Steppers: task stages and status compositions

Choose meaningful stages, not decorative division of a short form. Anatomy: task identity, ordered stage labels, current state, content, local errors, valid back/continue or commitment, and result/return. Stage location is not elapsed time or measured work.

For stage-based work, expose the owning workflow's [Step Status](step-status.md). Keep navigation, local validity, confirmed stage state and measured execution progress distinct; styling the next stage does not complete the previous one.

Derive sequencing from actual dependencies and recovery needs. Independent settings may work better as grouped sections with a summary, while a genuine ordered commitment can benefit from stages. Do not add artificial steps or completed-looking markers to make a simple task appear substantial.


## Represent the dependency rather than symmetry

Write stage labels that identify useful decisions. Use horizontal segments only while labels and meaning fit; a vertical arrangement or ordinary sections can serve long explanations better. Equal segments locate stages, not equal work or elapsed time.

Separate current navigation, local validity and acknowledged outcome visually. A segment can be current without completed, and an optional stage can be skipped without success. Read-only tracking needs status, not dead navigation buttons.

Parallel modes or outputs should stay parallel. Do not make document, slides and review look like a compulsory chain merely because a stepper is available. Keep the summary and next commitment easy to find; do not repeat the full navigation inside every stage.

## Adaptable treatments and optional motion

| Possible treatment | Useful job | Treatment and guardrail |
| --- | --- | --- |
| Dots and connectors | A compact wizard with short labels | Move current emphasis; update a completion connector only for a real satisfied prerequisite |
| Segment Bar | Discrete stages with little vertical space | Highlight labeled segments; equal widths do not imply equal duration or percentages |
| Order Tracking | Read-only shipment/request/service status | Show acknowledged events, timestamps and exceptions; reveal confirmed history without fabricated ETA |
| Breadcrumb Arrow | Ordered wizard stages needing broad labels | Arrow-shaped stage blocks with restrained active fill; shape does not create ancestor navigation |
| Vertical Detail | Long Korean labels or substantial explanations | Keep a side/vertical sequence and expand current detail rather than shrink a horizontal strip |
| Checklist Setup | Independent setup requirements people revisit | Show real completion and navigation; skipped optional work is distinct from complete |
| Review and Result | A consequential final commitment | Preserve a summary; submitting and confirmed result are different states |

Order Tracking is often an ordered history/status list, not an interactive wizard. Breadcrumb Arrow is a shape; actual breadcrumbs need parent/child hierarchy. Segment Bar must not become a progressbar whose number is guessed from the stage index.

## Stage model and navigation

Distinguish current, valid/completed, available but incomplete, unavailable future, error, optional and skipped. Visiting a stage is not completing it. Identify current/error/done beyond color and keep keyboard focus distinct from stage emphasis.

Linear navigation needs genuine dependencies; nonlinear navigation needs safe independent sections. Only reachable stages are links/buttons. Back retains input. Earlier dependency changes mark affected later values for review without erasing valid unrelated work. Distinguish local validity, draft persistence and final server confirmation.

Use an ordered structure and `aria-current="step"` where suitable; there is no `stepper` role. Read-only trackers do not need dead buttons. Follow ordinary link/button behavior or the existing primitive. Stage changes update heading/focus and error summary deliberately. Preserve URL, browser back and supported draft recovery.

## Panel transition options

Simple replacement is the baseline. Crossfade can soften unrelated panels; shared-axis slide can communicate forward/back in a sequence. Direction follows stage movement and writing direction rather than always entering from the right. Newly received tracking detail must not push someone away from what they are reading.

Set target state independently of the effect. Outgoing content cannot remain a duplicate active form; do not focus an incoming field while it is clipped offscreen. Validation/submission cannot depend on `animationend`. Rapid back/forward replaces old visual work without obsolete callbacks overwriting the latest stage.

Reduced motion keeps immediate stage changes and static current/done cues. Execution can have separate real measured progress. A connector filling, a final segment selected or transfer at 100% cannot confirm an entire job.

Test invalid continue, optional skip, changed dependency, failed commitment, browser back, draft resume, late status and reversal. Check desktop width, long labels and zoom before selecting horizontal/vertical geometry.

Sources: [MUI Stepper](https://mui.com/material-ui/react-stepper/) and [WAI-ARIA current state](https://www.w3.org/TR/wai-aria-1.2/#aria-current). These are compositions; MUI support does not imply a universal current Material requirement.
