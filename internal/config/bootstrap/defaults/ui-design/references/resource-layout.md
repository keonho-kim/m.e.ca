# Read guidance and studies for different reasons

The target project's requirements, established design system and current request govern its design. This skill is a reference system for making decisions. Its example collection is a separate fictional context, not the target project or a required starter architecture.

## Roles and authority

| Resource | Purpose | How to use it |
| --- | --- | --- |
| `SKILL.md` | Shared purpose, authority, essential boundaries and routing | Load once when the skill applies; choose what changes the current decision |
| `references/` maps | Locate a focused design question | Select a topic internally; no user-facing recipe command |
| `references/` guides | Criteria, relationships, tradeoffs, state and verification | Apply the reasoning to actual content; adapt or omit patterns |
| Source-study maps | Locate one implemented case and its direct dependencies | Inspect why it works and what it costs before copying any source |
| `assets/examples/` | Concrete HTML/CSS/JS, media and painted records | Optional material to inspect/adapt, not universal requirements |
| Study `DESIGN.md` / `PRODUCT.md` | Explain the fictional collection's scope and local choices | Keep separate from the actual project's design/product authority |
| Provenance/notices | Record authorship, illustrative media, licenses and source limits | Preserve relevant notices when material is reused |

A file can describe a successful choice without prescribing it. A dark data tool does not mandate dark mode; a minimal/editorial library does not assign its palette to all content apps. A local controller demonstrates behavior but does not choose another project's framework, state management or API.

## Keep discovery selective

Use entry → relevant map → selected guide/source. This bundle keeps three reading levels and every resource reachable within two file-link hops. Folder depth is organizational, not an extra reading stage. Peer references are optional topics available from their maps, not an instruction to recursively follow the whole tree.

Choose one useful screen, its relevant style/behavior and a painted record when needed. Do not load all apps, every theme or every shared file. Copy relative dependencies only when actually adapting source. Reuse the target project's components, tokens and service mechanisms where they already own the behavior.

No overview gallery, per-folder browsing app, theme selector or extra invocation syntax is needed. Markdown maps own the source locations. Add a reference only for a distinct decision; keep stable guidance at one owner rather than duplicating the same checklist in every folder.

## Distinguish bundle authoring limits

| Limit/convention | Applies to |
| --- | --- |
| Explanatory Markdown≤8KiB per file | Instructions maintained in this bundle |
| Code/media≤1MiB, preferably≤512KiB | Bundled study assets |
| Repository code headers and 350 physical lines | Authored study code in MECA's source tree |
| Reader text paging | Delivery through MECA's existing reader, not app source-file size |
| Local offline HTML/CSS/Vanilla JS studies | Their runtime portability, not a requirement that every target app use this stack |

Applications built with the skill follow their own architecture, tooling, budgets and supported platforms. These local limits do not authorize splitting their code mechanically, forbidding an established framework or adding MECA-style file headers.

## Standard and inspected precedent

The [Agent Skills specification](https://agentskills.io/specification) requires a frontmatter-bearing SKILL.md and permits supporting files. `references`, `assets` and `scripts` are recommended roles for documentation, static resources and executable automation, not universal names or these byte limits. Keep supporting material focused and load it as needed.

The pinned [Hallmark tree](https://github.com/Nutlope/hallmark/tree/407fa689f4a276f2fd389208d271d0fef08f5058) was inspected as a precedent: theme notes/images and code snippets in references, tooling/fixtures in scripts and public completed examples outside the installed skill. That observed layout is not an instruction to mirror it. Here complete studies stay inside assets so installation preserves their portability. [Provenance](example-provenance.md) owns source and distribution facts.
