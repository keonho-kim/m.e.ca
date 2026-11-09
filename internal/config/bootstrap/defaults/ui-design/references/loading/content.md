# Skeletons and retained content

## Where skeletons belong

| Layout | Initial content that may need skeletons | Keep usable or show explicitly |
| --- | --- | --- |
| App shell | Fetched workspace/account information when its shape is known | Safe static frame and navigation; unresolved access is not permission granted |
| Query/results | Data rows or card contents, not an invented exact total | Page title, column labels and supported query controls |
| Master-detail | Selected record's fetched body | Correct selected identity and return path; never present the previous record as current |
| Analytics | Value slots and a neutral evidence frame | Question, units, known scope; no fake zero or drawn sample trend |
| Detail, edit, settings | Fetched values and dependent sections | Known labels and actions whose prerequisites are ready; no editable fake defaults |
| Inbox, knowledge, AI workspace | First history/list/body fetch | Composer/draft when safe; an active AI answer uses real stream/status instead of endless text bars |
| Catalog, order, booking | Fetched item details or availability region | Applied date/resource/timezone, existing cart/input and confirmed selections |
| Public product page | Truly asynchronous product content if needed | Authored copy, navigation and hero; reserve real media dimensions rather than skeletonizing static content |

Size placeholders from the expected row, line, thumbnail or chart geometry. Approximate text structure without pretending to know exact content or a result count. Keep normal container spacing, column alignment and image ratio so replacement does not jump. Do not clone a whole table worth of expensive DOM solely to imitate an unknown collection.

On refresh, retain content only while its identity, permission and freshness make that safe. Make the last applied scope clear while another query is pending. On account changes, selection changes or other identity changes, clear or mark the affected region appropriately; cached data for A must not appear to be confirmed data for B. Prevent late responses from restoring obsolete content.

## Match the withheld information

A skeleton is a temporary composition of the expected content, not a disabled form or a second layout. Keep known headings, useful scope and control labels stable. Approximate a meaningful visible group rather than drawing the entire unknown dataset. A long document, row list and image collection need different placeholders.

Use surface contrast and rhythm that fit the product without competing with available work. Pulse may suit restrained repeated structure; Shimmer can suggest a brief directional wait. Both are optional expressions of the same state, not requirements to display every time. Preserve understandable pending/status text when motion or material effects are unavailable.

## Lifecycle

Use the relevant path, not every state for every feature:

~~~text
No content → pending → confirmed content / confirmed empty / denied / failure
Valid content → refreshing → updated content / retained stale content with explanation
Draft → submitting → confirmed / correctable failure / uncertain result
Job → queued / running phase → confirmed result / partial failure / cancellation result
~~~

Replace a loader when its read or phase ends. Empty, denied and failed are distinct outcomes, not excuses to leave a skeleton running. Avoid artificial minimum display time or delaying ready content to finish an animation. If an established delayed-indicator policy prevents flicker, it must not delay the operation or its result; cancel its timer on completion and teardown.


## Skeleton Pulse and Skeleton Shimmer

For known first-load structure, consider **Skeleton Pulse** as a quiet opacity variation or **Skeleton Shimmer** as a restrained moving highlight. Choose by density, surface and useful attention: many repeated rows rarely need many simultaneous highlights. Geometry should match the eventual content, not an arbitrary animated card grid.

Pulse changes opacity without resizing text/rows. Shimmer moves a highlight within the reserved shape rather than animating layout. Both stop on real content, empty, denied or failed outcome. Reduced motion and forced-color needs receive a static readable placeholder/status. Never delay already-ready content for a full animation cycle or use motion as the only sign of pending work.

An optional [loading study](../../assets/examples/motion/motion-lab.html) contains both variants; [motion CSS](../../assets/examples/shared/motion.css) defines their animation and reduced-motion rules. Study its geometry and terminal states, then use the target project’s primitive and data lifecycle. Its selectors and timing are not a reusable specification.

## Accessibility and material

Keep decorative skeleton pieces out of the accessibility tree. Provide a named loading status and mark the affected region busy when appropriate; keep useful existing content and actions accessible. A status outside a busy subtree can announce its transition without repeatedly describing placeholder shapes. End busy state on all terminal paths. Never move keyboard focus onto a skeleton.

Use the theme's neutral surface and feedback roles. Skeletons should resemble withheld structure rather than selected or disabled controls. Progress needs a visible track/fill relationship and readable status without color alone. Glass needs a legible underlying surface; minimalism still needs status; futurism must not add fake telemetry.

Avoid full-page waves and flashing regions. Check forced colors, contrast, zoom and overlay backgrounds. Stop animations, timers and observers when content resolves or the region leaves. Measure expensive blur, large animated gradients and repeated placeholders before adding more visual work.

Sources: [Carbon loading](https://www.carbondesignsystem.com/building-blocks/core/patterns/loading) and [MUI Skeleton](https://mui.com/material-ui/react-skeleton/). Static, pulse and shimmer are alternatives; decoration must not extend the pending state.
