# Buttons and floating actions

Use a button for an action and a real link for a destination. Anatomy: specific action verb or accessible icon name, stable hit area, priority, focus and applicable disabled/pending/result state. Keep primary, supporting, destructive, toggle and split behavior distinguishable; ordinary cancel is not destructive.

Place actions where the task needs them: near a local decision, in a stable repeated toolbar or persistently available across a long canvas. A floating action should earn its attention and space; do not add competing primaries simply because a recipe supports them.


## Establish action hierarchy

Use fill, ink, boundary and placement to distinguish commitment, supporting action and destructive consequence. Several actions can exist across a page; make the next decision clear within each region rather than making every control equally prominent. Distribute accent by stable role, not a different hue for each button.

Keep commands concise while preserving complete object names as content. A long request/article title can wrap; a nearby summary action can have a short visible verb and an accessible target-specific name. Do not shrink or truncate identity to imitate a compact demo. Put repeated work near its object and reserve floating placement for a real persistent need.

A minimal button still needs discoverability, focus and useful hit area. Surface treatment cannot replace the actual pressed, selected, pending or disabled state. Keep the ordinary target stable while decoration moves.

## Adaptable treatments and optional motion

| Possible treatment | Useful composition | Effect and constraint |
| --- | --- | --- |
| Quiet Press | Most working actions | Small surface/opacity/press cue without moving the hit area or delaying execution |
| State Morph | A bounded async action | Label/icon/pending state change within stable width; confirmed/failed follows actual results |
| Split Action | Common default plus related alternatives | Separate default and menu targets; animate the menu independently of the executed action |
| Extended FAB | One frequent primary action while the canvas scrolls | Stable floating icon+label; intentional label collapse must retain the name, hit area and focus |
| Speed Dial | A small, related set of real actions merits a floating group | Reveal a bounded action list near its trigger; use appropriate menu/disclosure semantics, Escape and keyboard access |
| Contextual Floating Rail | Actions depend on a current selection | Enter near relevant work, state the selection scope and disappear only when safe |
| Origin-to-Composer | A primary create action opens a related editor | Purposeful container continuity; one real editor and deliberate focus transfer |
| Magnetic / Tilt Accent | An explicitly expressive public surface | Only bounded decorative motion; ordinary target, pointer and keyboard operation stay reliable |

A desktop FAB is optional. A normal page toolbar often provides a clearer create action. Floating placement must not cover table continuation, focused fields, close controls, notifications or the main work at zoom. Do not add several competing floating primaries or a speed dial for one action.

## Behavior and async truth

Keep one clear primary commitment per task region. Secondary/quiet actions remain discoverable, and icon buttons have actual names; tooltip is supplementary. Toggle buttons expose their state; momentary actions do not acquire fake selected appearance. Disabled controls need meaningful prerequisite context when consequential and are not permission checks.

Pending identifies the actual operation, preserves input and prevents accidental duplicate interaction where useful. Do not make a success checkmark after a fixed duration, change a saving action into an unrelated target mid-click, or announce completion after only part of a batch succeeded. An uncertain result requires reconciliation, not an automatic repeated command.

## Motion ownership and checks

Use standard focus/pressed/selected roles on the actual theme. Prefer visual decoration inside a stable target rather than translating the entire hit region. A FAB hiding on scroll cannot disappear while focused or while its action list is in use. Do not install a global pointer/wheel listener simply to copy a demo; measure the need and clean up owned listeners, captures and frames.

Reduced motion uses static press/selection cues, immediate action-list reveal and no magnetic movement. Keyboard activation cannot require hover, drag or a custom cursor. Test fast double interaction, pending reversal, lost response, speed-dial escape/focus return, long Korean labels, scroll/zoom occlusion and route exit. Check focus does not land on an exiting invisible action.

Sources: [WAI APG button](https://www.w3.org/WAI/ARIA/apg/patterns/button/), [menu button](https://www.w3.org/WAI/ARIA/apg/patterns/menu-button/) and [MUI floating action button](https://mui.com/material-ui/react-floating-action-button/). Recipes do not require MUI or an animation framework.
