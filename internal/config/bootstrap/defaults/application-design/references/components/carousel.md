# Carousel: controlled slides and collections

A carousel reveals part of a meaningful set with deliberate continuation. Prefer a table/list or ordinary gallery when comparison, finding one record or stable location matters more than presentation. It is conditional on public/media surfaces, not default SaaS chrome.

Anatomy: named region/group, identifiable slides, previous/next, useful position or slide pickers, and rotation control if automatic motion exists. Reserve geometry and keep navigation available when media fails.

If all items must be found or compared, consider a visible collection rather than serially hiding them. Use a carousel only when its exploration/presentation benefit fits the content; do not conceal essential product facts merely to add movement or fill a hero.


## Let the set determine presentation

A large current image can support editorial inspection, while thumbnails give direct access to meaningful alternatives. Keep caption, selection and navigation stable as the image changes. A group of slides in an authoring tool is object navigation, not a reason to turn global application areas into a carousel.

Choose the image scale and visible neighboring items from what must be compared. A serial presentation adds recall cost; show a collection when several facts must be inspected together. Keep decisive metadata nearby when it affects the choice, or deliberately accept more scrolling for image-first reading.

Use accent for the active item/control and real state, preserving legibility over actual media. Different images supply variety without changing chrome color or animating every caption. Manual navigation is a complete design, not an unfinished animation.

## Adaptable treatments and optional motion

| Possible treatment | Useful composition | Motion and constraints |
| --- | --- | --- |
| Manual Snap Track | Product images or related items | Native scroll/snap or a measured track with buttons; pointer drag is optional |
| Crossfade | Same-frame imagery with stable framing | Fade media while retaining one interactive slide and stable captions/controls |
| Peek Cards | A visible edge explains continuation | Translate the track; peeking decoration cannot expose clipped keyboard targets |
| Thumbnail Gallery | Main media with meaningful direct choice | Update actual image, alt/caption and selected thumbnail while reserving aspect ratio |
| Center Emphasis | A small visual set with one active item | Subtle neighboring scale; avoid unreadable text or moving action targets |
| Depth Stack | Purposeful layered stories/previews | Restrained scale/translation, only active layer interactive, immediate alternative |
| Grouped Pages | Comparable visual cards shown in groups | Advance by intended group with real positions and truthful last-page behavior |

Depth and fade change presentation, not identity. Choose finite ends versus wrapping from the task. Avoid infinite clones with duplicate IDs, links, announcements or hidden focus stops. Counts follow the true set; unknown continuation is not a fake total.

## Interaction and accessibility

Provide buttons and optional direct selection with meaningful names. Repeated next/previous ordinarily keeps focus on the control. Tab order follows controls; do not hijack page arrow keys or vertical wheel input. Drag/swipe needs a non-gesture path and must not accidentally follow a card link after dragging.

Manual navigation is the baseline. If automatic rotation serves the brief, offer a discoverable stop/start control; stop on focus entry and hover, and do not restart after focus interruption without an explicit request. Do not advance essential copy, forms or a slide holding focus automatically. Reduced motion disables automatic rotation and retains a complete static/manual path.

Use the established accessible pattern rather than inventing `role="carousel"`. Inactive slides cannot retain controls in the tab sequence. Announce meaningful manual changes without continuous automatic live updates. Do not mix tabbed and button-picker keyboard models.

## Delivery, reversal and checks

Prioritize current media and reserve geometry. Bounded adjacent preloading depends on cost/context; do not fetch every full-resolution image or retain all previous slides. Failed media needs fallback and working navigation. Cleanup owns timers, observers, pointer capture and animations.

Rapid navigation re-targets visual state without leaving the logical index behind. Keep a valid item after removal, resize or route return. Snapshots are not a second active slideshow. Reduced motion preserves the active item without depth, parallax or long translation.

Test finite ends, one item, long captions, failed/lazy media, focused content during next, canceled drag, resize, route return and pause/resume. Controls over real images need legible backgrounds and keyboard discovery, not hover-only arrows.

Source: [WAI APG carousel](https://www.w3.org/WAI/ARIA/apg/patterns/carousel/). Visual recipe names are choices, not new accessibility roles.
