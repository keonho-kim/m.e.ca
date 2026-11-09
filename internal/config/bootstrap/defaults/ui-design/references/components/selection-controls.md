# Checkbox, radio, switch and range

Choose by meaning: checkbox for independent inclusion/consent, radio for one of a visible option set, switch for a binary setting, range/slider for continuous adjustment. A switch must make immediate versus saved-with-form application clear. Show actual units and an exact-value path when precision matters.

Anatomy: accessible label, operable target, visible current state, useful help/group legend and applicable pending/error. Make focus distinct from checked/selected and do not rely on color alone. Native controls or the established accessible primitive remain the behavior owner.

The decision model comes first: one choice, independent choices, immediate setting or precise range. Choose visible comparison and grouping from actual option content. An attractive selection card or animated mark cannot justify changing those semantics or adding unnecessary choices.


## Make the choice model visible

Keep the group question, option explanation and current choice readable. Independent choices, an exclusive value and an immediate setting need different semantics even if the visual treatment is a pill or card. Use larger selection surfaces only when their preview or explanation contributes to the decision.

Show selected state with a persistent mark/boundary as well as color. Keep focus, press and pending separate: a darker fill is not enough to explain which value was committed. Foreground and control edges remain legible when decorative effects or transparency are removed.

Accent can distinguish the active choice while content categories use their own stable meaning. Do not reuse success color to imply that selecting an option completed a task. In dense tables preserve readable labels and independent row/action targets before adding animated marks.

## Adaptable treatments and optional motion

| Possible treatment | Use | Effect and constraint |
| --- | --- | --- |
| Static Check | Dense repeated selections | Immediate mark and focus boundary; preserve the normal hit area |
| Check Draw | A purposeful local acknowledgement | Brief stroke reveal after selection; checked state changes immediately, not after the stroke |
| Fill Fade | Quiet selected/unselected states | Fade background and mark without hiding mixed state or focus |
| Mixed Aggregate | A page/group selected partly | Show a distinct dash/mixed indicator based on actual eligible selection scope |
| Selection Card | Options need explanation or a preview | A whole labeled choice surface with border/indicator; underlying semantics remain radio or checkbox |
| Radio Dot | Mutually exclusive choices | Short dot opacity/scale; never animate two choices into appearing simultaneously committed |
| Switch Thumb | A binary preference | Translate thumb within a stable track, retaining on/off text/context and real pending/result meaning |
| Range Fill | Continuous adjustment | Track/thumb follow the actual value directly; do not ease behind live keyboard/pointer input |

An optional ripple/press cue is separate from the selected value; it cannot be the only feedback. Avoid bounce or moving hit areas in dense tables. Theme-specific shadow/glow stays subordinate to actual selected, disabled, invalid and focus states.

## Group and asynchronous behavior

Use `fieldset`/`legend` or the existing labeled group pattern where choices share a question. Checkbox mixed state represents real partial selection; it is not a loading indicator. A select-all action names its scope: current page, explicit IDs or all matching results. Disabled members and unreturned pages need the actual selection contract.

Respect native Space/arrow behavior for the selected pattern. Do not make a checkbox act exclusively like a radio or intercept arrows outside a slider/radio group. Labels should activate their intended input without double toggling from competing click handlers. A visual checkmark does not enforce access or consent persistence.

For server-backed settings, show submitting/confirmed/failed/unknown as relevant. Reconcile rapid toggles using the actual API ordering/identity; an older failure cannot roll back a newer confirmed value. Preserve prior confirmed state and pending intent separately when needed. If changes apply only on Save, do not pretend the switch persisted immediately.

Reduced motion retains static marks and direct thumb/value updates. Stop stale selection animations on reversal, unmount or changed group. Test keyboard, long Korean option text, mixed→all→none, disabled members, server failure, rapid toggles, range endpoints and forced colors. Hidden decorative marks cannot replace the input's accessible state.

Sources: [WAI APG checkbox](https://www.w3.org/WAI/ARIA/apg/patterns/checkbox/), [radio](https://www.w3.org/WAI/ARIA/apg/patterns/radio/), [switch](https://www.w3.org/WAI/ARIA/apg/patterns/switch/) and [slider](https://www.w3.org/WAI/ARIA/apg/patterns/slider/).
