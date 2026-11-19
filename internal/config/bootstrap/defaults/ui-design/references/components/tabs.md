# Tabs and segmented choices

Tabs switch peer panels in one context. A segmented choice represents a short exclusive value/view set, using radio/toggle/tab semantics according to its job. Real route destinations keep navigation semantics; a decorative underline does not turn every header into ARIA tabs.

Provide a named set, readable labels, selected state, separate keyboard focus and corresponding content or value. Retain real panel, query and draft identity. For long labels, use an appropriate vertical rail or a scrollable tab row that keeps the selected tab visible and exposes overflow.

If the task requires comparing facts across panels, consider concurrent views, a retained summary or another arrangement that reduces memory burden. Tabs can serve independent peer views; they are not a default way to make every dense page look cleaner.

## Choose the activation model

| Variant | Composition and behavior |
| --- | --- |
| Automatically activated tabs | When the panels are ready, arrows can move focus and select the panel in one step. Expose the selected tab and panel relationship. |
| Manually activated tabs | For fetched or costly content, arrows move focus. Enter or Space activates the focused tab. Distinguish focused and selected tabs until then. |
| Document tabs | Label each open document and provide an operable close action. Handle dirty documents and move selection and focus to a suitable neighbor after closure. |
| Segmented filter or view control | Use one named selection group and show its selected option. Change the filter or display mode in place while retaining the query scope. |
| Destinations styled as tabs | Use navigation links and mark the current route. Preserve normal link actions and browser history. |

When tabs no longer fit, keep the current selection visible and the other labels discoverable. A scrollable row or vertical tablist preserves the underlying activation model.

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

Use the existing accessible tablist/tab/tabpanel relationships and keyboard model. For panels that appear immediately and have no side effects, activation can follow arrow-key focus. For slow or consequential changes, let arrows move focus and activate the panel on Enter or Space. Segmented radios or toggles keep their respective selection model.

Keep per-panel draft/query and intentional scroll state. Define whether inactive content remains mounted from real persistence/resource needs, not an animation library default. Hidden panels cannot retain keyboard targets. Do not duplicate the same form with live IDs in outgoing and incoming layers to crossfade it.

Remote loading has pending/empty/error and stale-response guards for its actual panel identity. Retained data must remain labeled by the correct scope. Re-selecting a tab must not replay a mutation or reset input merely to reproduce an entrance effect.

Set selected state independently of the effect. On rapid changes, retarget the indicator and discard old visual callbacks; panel focus goes only to visible, intended content. Reduced motion keeps immediate indicator/panel state with no lateral movement. Cleanup owns measurements, animations and requests.

Test arrow/focus versus selection, manual activation, long labels, overflow at zoom, lazy/failed panels, quick A→B→A, dirty content and route return. Inspect actual theme contrast: a faint or translucent selected pill cannot be the sole signal.

Source: WAI APG tabs. Visual recipes do not change the chosen semantic pattern.
