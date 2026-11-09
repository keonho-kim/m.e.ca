# Choose controls by meaning and information need

Read this level-2 map after identifying the region's task. Its families and variations are open-ended examples, not commands, an exhaustive catalog or a checklist to demonstrate on every screen. Adapt reliable semantics and choose a composition before an effect.

| Need | Read | Principal design decision |
| --- | --- | --- |
| Bounded interruption or work beside a selection | [Dialogs/drawers](dialogs-and-drawers.md) | Modal, inline, inspector or route; how much context stays available? |
| Anchored explanation, quick detail or supplementary help | [Popovers/tooltips](popovers-and-tooltips.md) | Essential visible information versus genuine optional help |
| Acknowledgement, failure, Undo or result | [Notifications](notifications.md) | Local/transient versus durable recovery |
| Commands or one/many value choices | [Dropdowns](dropdowns.md) | Menu, select, combobox or a richer comparison |
| Peer panels or a compact choice | [Tabs](tabs.md) | Panel selection, navigation or filtering; is comparison simultaneous? |
| A meaningful visual set | [Carousel](carousel.md) | Serial presentation versus visible collection/direct choice |
| Occasional related detail | [Accordion](accordion.md) | What stays visible, what can collapse and which groups compare? |
| Independent/exclusive/binary/continuous choice | [Selection controls](selection-controls.md) | Actual choice semantics and explicit selected/focus state |
| Text, exact values, prose or search | [Inputs](inputs.md) | Value meaning, labels, width, editing and correction |
| Related decisions and commitment | [Forms](forms.md) | Grouping, summary, dependency and recoverable input |
| Local, primary or persistent action | [Buttons/FAB](buttons-and-floating-actions.md) | Priority, target identity, placement and real consequence |
| Dependent input or read-only tracking | [Steppers](steppers.md) | Sequence, optional work and current stage |
| Actual multi-stage execution | [Step Status](step-status.md) | Confirmed, pending, blocked and unknown work |
| Location, hierarchy and continuation | [Navigation](navigation.md) | Global/project/object/view scope and return context |
| Repeated records or independent artifacts | [Collections](collections.md) | Comparison, exploration, selection and item/action separation |
| Secondary commands, category/filter/status | [Actions/status](actions-and-status.md) | Distinct meanings despite a shared compact shape |
| Repeated commands across substantial work | [Command palette](command-palette.md) | Discoverability, scoped candidates and explicit execution |

## Design and semantics stay connected

Define applicable normal, hover, focus, press/selection, disabled, pending and failure states. Do not manufacture unsupported states or give every component all possible effects. Real Korean labels, content length, input methods and geometry determine a usable result.

Distinguish behavior, composition and motion. Disclosure can explain prices, a process or evidence; its animation does not change the facts. A pill can be a filter, tab or category; appearance does not choose its role. A tree/filmstrip navigates objects, not necessarily global areas. Shared shapes need consistent type/boundary roles without erasing semantic differences.

Use a neutral foundation and purposeful accents where the decision needs recognition. Preserve state without color, shadow, transparency or movement. Keep ordinary controls familiar while adapting their information density and visual character to the actual app.

The peer [motion map](../motion.md) and [loading map](../loading-and-progress.md) explain relationships and waiting. Read them only when needed. The code studies provide implementation examples; real projects should reuse their own accessible primitives and state/lifecycle mechanisms rather than adopt a sample's module structure.
