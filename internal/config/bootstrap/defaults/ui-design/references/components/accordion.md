# Accordion: disclosure with several compositions

Use an accordion when revealing related sections helps the task. Anatomy: section heading with a real disclosure button, clear expanded state, associated content, optional summary and separate secondary actions. Short content may need only a heading. Long Korean headings must wrap without hiding the trigger.

Decide what people must see together before hiding sections. Occasional detail can be disclosed; information essential to the current decision may need visible rows, open groups or another composition. A Price Breakdown recipe alone does not justify concealing prices or inventing line items.


## Give disclosure a content reason

Use heading, summary and spacing to make the closed section understandable. Keep the trigger distinct from any secondary action. Essential terms, totals or current errors may need to remain visible; opening and remembering several sections is a poor substitute for simultaneous comparison.

Choose the emphasis from the content relationship: a price summary, stage explanation, optional chart evidence or a shared preview. An accent line is one possible cue, not a required strip on every rounded panel. A minimal page still needs a recognizable disclosure affordance and clear open state.

Choose single-open or multiple-open behavior from the task, then make group controls truthful. Expand-all is useful for a meaningful comparable set; it is not an instruction to initially expand every disclosure in an application. Typography and grouping should continue to work without the expansion effect.

## Adaptable treatments and optional motion

| Possible treatment | Useful composition | Motion and state |
| --- | --- | --- |
| Expand all / Collapse all | Baseline multi-section work where comparison matters | Multiple sections can stay open; an overall control changes eligible sections together with restrained expansion |
| Price Breakdown | A readable total with tax/fee/line-item explanation | Expand rows beneath a stable total; reveal separators/amounts together without animating invented prices |
| Accent Bar | Dense settings, FAQ or policies whose open section needs emphasis | Grow/fade a side accent as the body expands; accent is not the sole state cue |
| Step Timeline | Sequential explanation, delivery details or process history | Open stage detail beside a persistent sequence line; extend disclosed detail without implying fictional completion |
| Chart Reveal | A summary with optional evidence/definitions | Reserve a chart region, expand the panel, then reveal actual marks once data and dimensions exist |
| Split Preview | Explanatory headings beside a shared desktop preview | Change the selected explanation and real preview with a restrained crossfade and a clear relationship |
| Nested Detail | A genuine hierarchy with occasional deeper disclosure | Indent relevant descendants; prefer a tree/navigation model if hierarchy is the main task |

Expand all is a supported baseline variation, not a requirement that every accordion initially expose everything. Choose initial sections from the task. Single-open behavior suits one active explanation; it cannot promise simultaneous comparison or a functioning Expand all action. The overall control needs truthful action text and scope for disabled or not-yet-loaded sections.

Price totals, timeline events and chart values come from actual data. Disclosure never recalculates a quote or marks a stage done. Step Timeline is not automatically a navigable stepper. A shared preview needs a useful reading order and an alternative for unavailable media/evidence.

## Expansion mechanics

Measure real block size or use an established intrinsic-size/layout technique with an instant fallback. A guessed `max-height` can truncate translations, errors, images or later streamed content. Scaling the body vertically distorts text and leaves wrong layout space. If measuring, handle real content changes without a read/write loop; release observers and let final open content size naturally.

For Chart Reveal, avoid starting at zero width or redrawing expensive charts on every expansion frame. Retain a readable summary/data alternative and do not replay an elaborate series animation on every reopen. Expand-all may affect many regions: avoid a long stagger sequence or unnecessary per-panel effects.

## Semantics and failure cases

Use heading/button relationships and `aria-expanded` with panel identity from the existing primitive. Keep header links/actions separate; do not nest interactive controls. Enter/Space operates the trigger. Follow the primitive for optional arrow navigation rather than inventing a keyboard model.

A collapsed body cannot leave invisible fields in the tab sequence. If collapse hides current focus, return it to a suitable trigger. Preserve dirty fields and reveal a section containing a validation target. Lazy content has independent pending, empty and failure states; closing it does not prove a mutation stopped.

Reduced motion gives immediate expansion with static expanded/selected cues. Test expand-all mixed state, rapid reversal, late content, long labels, nested focus, validation navigation, partial loading and route exit. In every theme preserve text, boundary and focus legibility; chevron rotation or an accent cannot carry the only meaning.

Sources: [WAI APG accordion](https://www.w3.org/WAI/ARIA/apg/patterns/accordion/) and [Radix content-size animation](https://www.radix-ui.com/primitives/docs/components/accordion#animating-content-size). Named compositions are original recipes, not standards or required library APIs.
