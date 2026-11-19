# Choose a component and its behavior

Identify the target, the decision and what must stay available during the task. Compare the relevant patterns and read the linked variants for keyboard behavior, state and motion.

## Commands and values

| Pattern | Choose when | Key behavior |
| --- | --- | --- |
| Split button | One frequent command has related variants | The main target runs the default. A separate menu trigger opens alternatives. |
| Meatballs menu | An item has secondary actions | A labeled ellipsis button opens commands scoped to that item. Escape returns focus. |
| [Context menu](../interactions/contextual-actions.md) | Actions depend on an invoked item or canvas point | Support pointer and keyboard invocation and a visible action path. Keep the original target while open. |
| Toolbar | Related commands act on the same work | Keep common actions visible. A composite toolbar uses one Tab stop with movement among its controls. |
| Destructive action | A command removes or overwrites something | Name the target and consequence. Use reliable Undo for reversible effects or confirmation when the cost warrants it. |
| Dropdown | Commands or values need a compact surface | A menu runs a command. A select presents the current value. A combobox supports typing to find or enter a value. |
| Radio, checkbox or switch | Compare exclusive values, mark inclusion or change an on/off setting | Expose selected or checked state and whether changes apply immediately or on Save. |
| Chip | A compact item adds a filter, runs an action or represents a chosen value | Match activation or removal to the job. Give a remove control a distinct label. |
| Badge | A small label communicates a category, count or status | Keep the meaning visible in text or an accessible name. Provide a separate control when an action is available. |
| Slider or exact field | Explore a range or enter a precise value | Update the visible value immediately and keep units clear. Offer exact entry where needed. |
| Command palette | Typing accelerates frequent workspace commands | Show a scoped candidate and run it on deliberate activation. |

[Buttons](buttons-and-floating-actions.md) · [Menus and pickers](dropdowns.md) · [Selection controls](selection-controls.md) · [Chips and toolbar](actions-and-status.md) · [Command palette](command-palette.md)

## Help, disclosure and overlays

| Pattern | Choose when | Key behavior |
| --- | --- | --- |
| Tooltip | A control benefits from short supplementary text | Show on hover or focus, support dismissal and keep focus on the control. |
| Popover | Explanation or a small interaction belongs to a trigger | Keep placement anchored. Decide what Apply and Escape preserve and where focus goes. |
| Accordion or disclosure | Optional detail belongs in the reading flow | Change the expanded state through its trigger. Keep multiple sections open when comparison matters. |
| Modal | A bounded decision needs exclusive attention | Move focus inside, keep the background inert, and define Cancel, confirm and focus return. |
| Drawer | Edge placement preserves context or gives the task more space | An inspector keeps the background usable. A modal drawer follows dialog behavior. |
| Scrim | A dimmed background clarifies a modal layer | Couple it to the active layer and its dismissal policy. Keep a named close action available. |
| Inline editor or dedicated view | The work is one local edit or a substantial resumable task | Keep the target and draft. Provide a clear commit and return path. |

[Popovers and tooltips](popovers-and-tooltips.md) · [Accordion](accordion.md) · [Dialogs, drawers and scrims](dialogs-and-drawers.md)

## Navigation and stages

| Pattern | Choose when | Key behavior |
| --- | --- | --- |
| Breadcrumb | The current page has useful ancestors | Link to the actual ancestors, mark the current page and keep the parent reachable at narrow widths. |
| Pagination | People must revisit or jump between result pages | Mark the current page. Preserve query and sort and handle the first and last boundaries. |
| Hamburger menu | Main navigation needs a compact trigger | Open a named navigation region or drawer. Dismissal restores focus and navigation leads to the destination. |
| Tabs or segmented choice | Switch peer panels or change a value for one view | Use manual tab activation for costly panels. Segmented value choices apply according to the field or filter contract. |
| Navigation disclosure or tree | Links have groups or objects have nested operations | Link groups keep the normal Tab order. A tree supports movement, branch expansion and selection as distinct operations. |
| Wizard or checklist | Work is ordered by prerequisites or can be completed independently | Keep input on return. Expose valid destinations and completion from real checks. |

[Navigation](navigation.md) · [Tabs](tabs.md) · [Steppers](steppers.md)

## Input and collections

| Pattern | Choose when | Key behavior |
| --- | --- | --- |
| Inline validation | A person can correct a field in place | Show a specific error after meaningful interaction, associate it with the field and clear it when resolved. |
| Draft form or autosave | Edits apply together or can persist independently | Hold a draft until Save or show the actual pending and persisted versions. Preserve failed input. |
| Date, range or file picker | Calendar context, constraints or transfers matter | Offer typed or keyboard access. Show incomplete ranges and separate transfer from processing. |
| List, card or table | Scan names, preview artifacts or compare aligned fields | Keep links, selection and actions distinct. In a sortable table, keep query and actual order clear. |
| Editable grid or [reorderable list](../interactions/drag-and-drop.md) | Work requires cell navigation or intentional position changes | Separate arrow navigation from editing. Request the same move on drop or through the visible Move action. |
| Carousel or gallery | Browse media in sequence or compare items together | Give direct navigation and current position. Present comparable items together and provide a pause control for rotation. |

[Inputs](inputs.md) · [Forms](forms.md) · [Collections](collections.md) · [Carousel](carousel.md)

## Feedback and recovery

| Pattern | Choose when | Key behavior |
| --- | --- | --- |
| Toast | A brief acknowledgement can coexist with work | Always include a visible × close button. Dismiss on activation and keep task focus on arrival. |
| Inline alert | A result or problem belongs to a specific task region | Place it by the cause and keep the relevant correction or retry available. |
| Banner alert | A condition affects a page, workspace or service | Keep it visible while the condition matters and expose scope and recovery. |
| Skeleton loading | Initial content is pending and its shape is predictable | Reserve space, keep useful controls available and replace the skeleton on the real outcome. |
| Progress indicator | The person needs work progress or activity | Use a real amount for determinate progress. Show named activity when the amount is unknown. |
| Empty state | A completed read has zero items or matches | Explain the scope and offer the relevant create, reset or return action. Keep failure and access states distinct. |

[Notifications](notifications.md) · [Loading and progress](../loading-and-progress.md) · [Step status](step-status.md)

## Interactive states

Describe what receives focus, what key or pointer action changes it, what commits, what dismissal preserves and where focus goes next. Cover applicable hover, focus, pressed, selected, expanded, disabled, pending and invalid states, including combinations such as selected and focused. Verify the same useful outcome with motion reduced. [Interactive states](../interaction-contracts.md#interactive-states) · [Motion map](../motion.md)
