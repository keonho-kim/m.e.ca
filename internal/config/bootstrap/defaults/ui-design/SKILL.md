---
name: ui-design
description: Design or improve desktop WebApp/SaaS through task-based layouts, visual direction, Korean typography, usable controls and purposeful motion. Use for frontend composition, app flows and design review; supporting code is illustrative rather than a required template.
license: Apache-2.0; bundled materials retain their notices; see LICENSE.txt and NOTICE.md
---

# UI DESIGN

Design the work people need to understand, compare and complete. Derive the screen and its substantial regions before choosing controls or effects. Default to desktop WebApp/SaaS while preserving resized-window, zoom and keyboard usability.

## Use the right authority

The user's request, actual requirements and the target project's system govern the work. Inspect its content, routes, components and behavior. Preserve settled facts; replace an identity only within an authorized redesign.

References supply decision criteria; `assets/examples/` supplies fictional studies. Their palettes, names, page counts, data and modules are local choices. Their `DESIGN.md` and `PRODUCT.md` describe those studies, not the target project's brand or requirements. Transfer the design reason and derive a fitting result, rather than installing the collection.

Match the deliverable: an honest design shell or a functional frontend using the services in scope. Do not add a backend, document engine or unrelated page suite to make an example feel complete.

## One entry, selective reading

Invoke only `#ui-design`, without subcommands or required arguments. Infer the task from conversation and project; ask about a consequential unresolved purpose or constraint, not a component recipe. Respect planning-only and review-only requests.

Read **entry → relevant map → selected resource**. Keep three reading levels/two file-link hops and load only material needed for the decision. The [resource guide](references/resource-layout.md) explains how guidance, case studies and assets differ. This bundle's 8KiB instruction and 1MiB asset budgets are authoring limits, not constraints on applications built with it.

## Establish the design problem

For uncertain goals use the [brief](references/brief.md); for a new product use [app scenarios](references/app-scenarios.md). Follow [usability reasoning](references/usability.md) to decide what stays visible, what must be compared together, what is sequential and what must survive interruption. Assumptions are not research findings.

Before substantial implementation, record a concise outcome and scope, one complete path, necessary regions, visual direction and important state/recovery checks. A useful path includes entry, the task, its result/detail and a return or recovery. Mark conditional regions instead of adding dashboards, billing, chat or onboarding by habit. Reuse settled decisions rather than requiring a new planning ceremony for every control.

| Need | Read |
| --- | --- |
| Large regions and their relationships | [Layout map](references/layout-map.md) |
| Global/project/object/view navigation | [Application shell](references/layouts/app-shell.md) |
| Search, comparison, records and analytical evidence | [Data workspaces](references/layouts/data-workspaces.md) |
| Detail, forms, settings and staged commitments | [Task flows](references/layouts/task-flows.md) |
| Queues, collaboration, AI input/result and authoring context | [Collaboration](references/layouts/collaboration.md) |
| Reading, publishing, authoring and technical tools | [Content pages](references/layouts/content-pages.md) |
| Discovery, transactions and scheduling | [Commerce and booking](references/layouts/commerce-and-booking.md) |
| Public explanation and activation | [Public pages](references/layouts/public-pages.md) |

Use [Korean desktop guidance](references/korean-desktop.md) for actual Hangul/mixed-script text, IME, readable density and local formats. Language and usage context guide the choices; there is no mandatory national visual style.

## Derive identity beyond color

Use the [visual-system map](references/visual-system.md) for composition, typography, density, geometry, content, material and motion. Professional clarity comes from hierarchy, consistency and truthful operation, not a pale-blue palette, an all-white canvas or a sidebar. Editorial expression can live in a detail or authored artifact while its surrounding tools stay compact.

The [eight color principles](references/color-systems.md) explain neutral foundations, role-based accents, category/data colors, state and light/dark relationships. Distribute useful accents across the decisions that need recognition; define their meanings before choosing swatches. A requested expressive palette still needs readable pairs and clear selection/focus.

Read only a relevant material: [glass](references/themes/glassmorphism.md), [minimalism](references/themes/minimalism.md) or [futurism](references/themes/futurism.md). These are adaptable materials, not app-category assignments. Region-specific intensity can vary within a coherent system. A reference brand supplies evidence of a useful relationship, not permission to copy its whole identity; use [service craft](references/service-craft.md).

## Compose controls, feedback and motion

| Decision | Read |
| --- | --- |
| Control meaning, visual grouping and context-specific variations | [Component map](references/components/index.md) |
| First load, refresh, media, execution and real streaming | [Loading map](references/loading-and-progress.md) |
| State feedback, transitions, continuity and expressive movement | [Motion map](references/motion.md) |
| Authoritative state, scope, interruption and recovery | [Interaction contracts](references/interaction-contracts.md) |

Choose semantics and information visibility before styling. Tabs, filters, destinations, stage navigation and processing progress have different jobs even when they share a shape. Use components from the target system or reliable native primitives; code in the studies does not require a particular framework or global controller.

For known first-load structure, consider Skeleton Pulse or Shimmer. Meaningful content imagery defaults to genuine Blur-up when the pipeline can supply a matching derivative; retain geometry and failure alternatives. Stage-based work uses actual Step Status. Cached/ready results are not delayed to showcase an effect.

Motion follows a change in relationship or state: selection, a revealed record, a replaced artifact, a meaningful amount or a contextual layer. Choose effects for that reason, with interruption, focus and reduced-motion behavior. Do not replay the same entrance across every working region or create success with an animation timer.

## Use examples as evidence

Use [reasoning examples](references/examples.md) for contrasting situations. Read a concrete study only when its composition helps: [application source map](references/code-examples.md), [research/office studies](references/product-workspaces.md), [same-task comparisons](references/visual-studies.md). Study a tradeoff, not an optimum or universal business model.

Apply [anti-slop](references/anti-slop.md) throughout: real content, meaningful hierarchy and truthful state. Distinguish a demonstrable defect, an unchosen default and intentional expression. Requested material and familiar controls remain valid; fabricated evidence, inaccessible operation and unreadable text do not.

## Verify and refine the cause

Use [browser tools](references/browser-tools.md) and [quality checks](references/quality.md). If Playwright CLI is missing, ask the user to install it for UI verification; do not install it automatically. Compare rendered views with the brief at relevant sizes; exercise paths, keyboard/IME, failure and interruption. Diagnose layout, type, content, state or material; recheck corrections.

Report actual evidence and remaining limits in the user's language. A screenshot proves a painted state; a build, detector score or agent walkthrough does not certify usability, accessibility, service integration or production readiness.
