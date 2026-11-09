# Use type, geometry and density to establish hierarchy

Choose roles from actual content and work. Inspect the target system before adding a font, size scale or boundary language. The example apps' local fonts and dimensions are implementation choices, not a type specification for other projects.

## Distinguish content from chrome

A file title or toolbar can stay compact while the authored page, slide or editorial subject has substantial scale. Visual prominence need not follow a rule that every application's first heading is the largest text. Preserve a meaningful semantic heading order while assigning visual roles.

| Role | Design decision | Check |
| --- | --- | --- |
| UI labels/actions | Clear familiar family, stable weight, adequate target and spacing | Long labels, Hangul coverage and focus/state visibility |
| Content title | Scale/weight or an earned contrasting family | Subject clarity, phrase wrapping and proportion to the object |
| Reading body | Comfortable measure, leading and paragraph rhythm | Real prose and sustained reading, not a Latin specimen alone |
| Supporting metadata | Secondary emphasis without disappearing | Relevant facts still readable on selected/material backgrounds |
| Data/code | Comparable alignment, units and meaningful technical typography | Tabular figures where useful; mono for actual code/identifiers |

A readable UI family can carry the product. A contrasting display or serif role can enrich an editorial/detail/artifact surface without spreading into every label. Use installed/project fonts or licensed self-hosted resources as appropriate; the bundle does not require its system-font stack or a remote font service.

## Work with real text

Use actual Korean and mixed Latin/numerals, dates, prices, identifiers, help and failure copy. Tune size, weight, leading, width and wrapping together. Desktop operating text often starts around 14–16px, but actual font and audience decide; this is not a WCAG size requirement. Small metadata must not become an excuse to hide essential information.

Latin `ch` measures do not directly describe Hangul reading length. Inspect real lines instead of mechanically enforcing a prose width everywhere. Word-aware Korean wrapping can preserve phrases; provide a safe escape for long identifiers/URLs. Avoid extreme tracking or squeezing text to reproduce a reference silhouette.

Keep command labels concise. Object names and article titles may need wrapping and a useful full-content path. If space requires separate summary/navigation actions, retain the complete identity and give the action a specific name. Do not truncate decisive information merely to pass a generic line-wrap detector.

## Choose boundaries by relationship

Rows need shared alignment and restrained separation; documents need a reading measure and deliberate rhythm; artifacts need real proportions/captions; parent/detail and tool/content need a recognizable change of surface or edge. Use whitespace, rules, surface contrast or elevation for that relationship rather than adding all four to every rectangle.

Select a small coherent radius/elevation vocabulary. Minimal surfaces still need hierarchy. Elevation cannot be the sole control boundary. Glass cannot make dense text translucent. A stable floating layer can earn separation that a whole row of ordinary cards does not need.

## Derive density

Repeated processing benefits from nearby actions and comparable rows. Reading benefits from a capped measure. Discovery may earn image scale and more breathing room. A spatial task needs more canvas than chrome. On a wider monitor choose whether more objects, detail or simultaneous context helps; do not stretch all type and sentences proportionally.

Use grouping, fewer redundant labels and aligned columns before reducing type or hit areas. Compactness and readability can coexist. Large whitespace should separate decisions, not disguise missing content.

Inspect laptop height, a wider desktop, resized windows, zoom and long content. Ensure fixed/sticky regions and nested scrollers do not cover text or focus. Keep shared roles with their system owner and local content exceptions with the relevant page; no token framework is needed merely to name one value.

Sources: [MDN font-family](https://developer.mozilla.org/en-US/docs/Web/CSS/font-family), [WCAG resize text](https://www.w3.org/WAI/WCAG22/Understanding/resize-text.html), [IBM grid reasoning](https://www.carbondesignsystem.com/building-blocks/foundations/2x-grid/overview).
