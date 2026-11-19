---
name: ui-design
description: Design or review WebApp and SaaS interfaces by choosing UI components and defining their interactions, states and animation. Use for forms, overlays, navigation, drag and drop, pointer gestures, feedback and transitions in new or existing screens.
license: Apache-2.0; bundled materials retain their notices; see LICENSE.txt and NOTICE.md
---

# UI design

Start with what the person needs to do, what must stay visible and when a choice takes effect. Inspect the actual content, routes, state and design system. Use `#ui-design` followed by the requested design, implementation or review. Match the deliverable and level of detail to that request.

## Choose the component

1. **Identify its job and scope.** A control can navigate to a destination, run a command, edit a value, switch a view, reveal content or report status. Identify the affected object and what the user needs to see while acting.
2. **Decide what stays visible.** Weigh comparison, frequency, space, option count, label length and whether choices are known or searched. Give important actions and comparison criteria a visible home. Show the current value and separate focus, preview and selection where they differ.
3. **Define when it takes effect.** Distinguish changes that apply immediately, edits held until Save and work pending confirmation. Choose interruption, dismissal and recovery from consequence and reversibility. Preserve the context and draft the user needs to resume.
4. **Choose a component and its variant.** Reuse the target system's accessible component or a suitable native control. Define its trigger, focus destination, key behavior, commit and dismiss paths, and what stays selected or editable. Verify labels and usable placement at supported widths and zoom.

Start from the conventional interaction for the chosen pattern. State what must be visible, where focus starts, what activates or commits, how the user closes or recovers, and what focus and input survive. Compare variants on the requirement that warrants a different behavior. Use the [component selection guide](references/components/index.md).

## Design the animation

Describe the trigger, logical state change, affected region, visual cue, interruption and final state. Give repeated actions immediate feedback. Use a small indicator change for selection, an anchored reveal for a contextual layer, or spatial continuity when the same object moves. Direct manipulation tracks input and may settle on release. A static change can be the complete response.

Make input, focus and status available when the actual state changes. Keep pending and confirmed feedback tied to work events. Handle quick reversal, dismissal and route exit at the owning component. Provide the same usable final state with reduced motion and where optional animation features are unavailable. Use the [motion map](references/motion.md).

## Read what changes the decision

Read **entry → relevant map → selected resource**. Use [the resource guide](references/resource-layout.md) for the bundle's reading depth and size budgets.

| Need | Read |
| --- | --- |
| Reordering, moving across groups, file drop or canvas placement | [Drag and drop](references/interactions/drag-and-drop.md) |
| Right click, long press, contextual menus and inspectors | [Context menus and panels](references/interactions/contextual-actions.md) |
| Resizing, moving panels, swipe reveal and area selection | [Direct manipulation](references/interactions/direct-manipulation.md) |
| Drafts, selection identity, async responses and recovery | [Interaction contracts](references/interaction-contracts.md) |
| Initial load, refresh, media, progress and streaming | [Loading map](references/loading-and-progress.md) |
| Task goal, required comparison and success criteria | [Design brief](references/brief.md), [usability](references/usability.md), [app scenarios](references/app-scenarios.md) |
| Page regions, navigation hierarchy and retained context | [Layout map](references/layout-map.md) |
| Typography, hierarchy, color and material | [Visual system](references/visual-system.md), [service craft](references/service-craft.md), [design review](references/anti-slop.md) |
| Cases with different constraints or an implemented example | [Reasoning examples](references/examples.md), [application source](references/code-examples.md), [workspaces](references/product-workspaces.md), [visual studies](references/visual-studies.md) |
| Actual interaction and rendered checks | [Browser tools](references/browser-tools.md), [quality checks](references/quality.md) |

For whole-screen work, identify the regions the task needs and follow one complete path through result and return or recovery. Use fictional studies to examine a specific choice and apply its reasoning to the target project's content, components and service contracts.

## Verify and refine

Exercise the chosen control with representative values, keyboard, pointer and relevant text composition. Check the current value or target, focus, dismissal, long labels and failure or recovery where applicable. Test motion at normal and reduced settings, including repeated activation and interruption. Verify supported browsers and the functional fallback for any optional enhancement.

For implemented work, use the available MECA browser tools and collect evidence from the rendered screen and the completed interaction. Use optional Playwright CLI diagnostics when needed. Report the decision, observed behavior and remaining limits in the user's language.
