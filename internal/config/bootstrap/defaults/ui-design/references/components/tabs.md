# Tabs and segmented choices

Tabs switch peer panels in one context. A segmented choice represents a short exclusive value/view set, using radio/toggle/tab semantics according to its job. Real route destinations keep navigation semantics; a decorative underline does not turn every header into ARIA tabs.

Anatomy: named set, readable labels, selected state, separate keyboard focus, corresponding content/value and a useful overflow strategy. Retain real panel/query/draft identity. Use select or another layout when long Korean labels no longer fit; tiny icon pills cannot rescue an unsuitable option set.

If the task requires comparing facts across panels, consider concurrent views, a retained summary or another arrangement that reduces memory burden. Tabs can serve independent peer views; they are not a default way to make every dense page look cleaner.


## Separate scope and visual emphasis

First identify whether the control navigates a destination, selects an object, filters a collection or switches a peer panel. Global areas, open documents and result panels need visibly different scopes. A group of status filters may be a single-choice filter over one list rather than several copies of that list as tab panels.

Choose underline, contained selection or a vertical arrangement from density, label length and the surrounding hierarchy. Keep the active view clear without making each layer of navigation look equally primary. A content outline or filmstrip can deserve a side pane even when global navigation is above the work.

When evidence must be compared at once, retain it beside the result or in a stable summary instead of forcing repeated tab changes. Parallel output formats are not completion stages. Accents should clarify selected context; they do not define routing or completion.

## Adaptable treatments and optional motion

| Possible treatment | Use | Effect and constraint |
| --- | --- | --- |
| Static Underline | Familiar peer panels in operational UI | Immediate active boundary; content can replace without animation |
| Underline Glide | Same repeated tab rail | Translate/resize one indicator to the actual label; focus is still independently visible |
| Moving Pill | Compact exclusive views | A shared background moves beneath the selected label; do not morph text or hit targets |
| Panel Crossfade | Panels share a stable region but little spatial meaning | Fade only valid panel content with one active interactive subtree |
| Shared-axis Panels | Ordered peer views where direction helps | Short horizontal/vertical transition following actual selection order and writing direction |
| Vertical Rail | Detail/settings sections with longer labels | Stable side navigation plus selected panel; keep useful work width and reading order |
| Count/Badge Update | Real changing quantities within a stable set | Crossfade the value without shifting labels or making false count-up animation |

An animated indicator is not the current keyboard focus. Disabled or unavailable content needs truthful affordance. A tab with an error can signal the issue, but the actual explanation remains with its field/task.

## Activation and panel lifecycle

Use the existing accessible tablist/tab/tabpanel relationships and keyboard model. Automatic activation can suit readily available panels; manual activation avoids costly requests on each arrow focus. Segmented radios or toggles keep their respective selection model; do not copy tab roles solely for a pill appearance.

Keep per-panel draft/query and intentional scroll state. Define whether inactive content remains mounted from real persistence/resource needs, not an animation library default. Hidden panels cannot retain keyboard targets. Do not duplicate the same form with live IDs in outgoing and incoming layers to crossfade it.

Remote loading has pending/empty/error and stale-response guards for its actual panel identity. Retained data must remain labeled by the correct scope. Re-selecting a tab must not replay a mutation or reset input merely to reproduce an entrance effect.

Set selected state independently of the effect. On rapid changes, retarget the indicator and discard old visual callbacks; panel focus goes only to visible, intended content. Reduced motion keeps immediate indicator/panel state with no lateral movement. Cleanup owns measurements, animations and requests.

Test arrow/focus versus selection, manual activation, long labels, overflow at zoom, lazy/failed panels, quick A→B→A, dirty content and route return. Inspect actual theme contrast: a faint or translucent selected pill cannot be the sole signal.

Source: [WAI APG tabs](https://www.w3.org/WAI/ARIA/apg/patterns/tabs/). Visual recipes do not change the chosen semantic pattern.
