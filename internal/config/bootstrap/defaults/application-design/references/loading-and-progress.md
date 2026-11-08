# Choose feedback for the actual wait

Identify the current resource/action, whether useful content exists, what work can be observed and what ends the state. Feedback should answer what the person can understand or safely do next, not decorate every request.

| Situation | Possible treatment | Design boundary |
| --- | --- | --- |
| First content with predictable shape | Local Skeleton Pulse or Shimmer | Reserve useful geometry, not fabricated data/counts |
| Work with little predictable content shape | Named waiting status/indicator | Explain the pending task without several competing spinners |
| Save/send/local commitment | Inline or action-level pending | Retain input and context; motion cannot confirm success |
| Same-resource refresh | Retain correct content with refreshing scope | Old results must not look like a newly applied query |
| Measurable work | Linear/circular determinate display | Name the actual numerator/denominator and phase |
| Unknown amount | Indeterminate cue with meaningful status | No guessed percentage, countdown or permanent 99% |
| Actual stages | Step Status with phase progress if available | Stage count is not total work or duration |
| Independent regions | Show useful content as ready | Do not block everything on the slowest secondary fetch |
| Meaningful raster imagery | Matching Blur-up preview and reserved ratio | Ready/cached media appears directly; provide failure alternatives |
| Continuation | Feedback at the collection boundary | Keep existing rows, applied scope and return position |
| Reversible reconciliable mutation | Explicit pending optimistic presentation | Confirmation, rollback and uncertainty follow the real contract |
| Existing capacity/quantity | Meter or labeled value | A bounded quantity is not execution progress |

## Read the relevant family

- [Content](loading/content.md): placeholder geometry, first load, refresh and identity.
- [Progress](loading/progress.md): measured/unknown work, action feedback, stages and streams.
- [Delivery](loading/delivery.md): images, independent regions, continuation and optimistic continuity.

These are level-3 resources from this map. Select one; do not add every technique to the page. The peer [interaction contract](interaction-contracts.md) owns state meaning and recovery.

## Design the attention hierarchy

Keep known context and safe actions available. Place feedback by the region or operation it describes. A quiet refresh may need less emphasis than an uncertain commitment; a durable failure needs more persistence than routine acknowledgement. Reserve geometry so arriving content does not move the primary target repeatedly.

Choose Pulse versus Shimmer from density and visual character. Do not combine a moving highlight, floating image, row stagger and several spinners in the same region. Minimalism still communicates waiting; every material needs readable status. Use accent to distinguish real state, not to imply every pending operation succeeded.

Logical state owns feedback. Ready, cached, empty, denied, failed and cancelled outcomes do not wait for a full animation cycle. Loading is not a reason to manufacture latency or simulated AI output. A prototype and a real operation must remain honestly distinguishable.
