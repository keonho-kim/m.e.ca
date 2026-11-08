---
name: create-skill
description: Create or revise an Agent Skill with valid SKILL.md metadata, focused instructions and useful supporting resources. Use when the user asks to write, package or improve a reusable agent skill.
---

# Create an Agent Skill

Produce a reusable instruction directory for the user's actual workflow. Preserve the requested scope, destination and existing edits. A skill provides guidance; it does not grant additional permissions or start a separate process.

## Choose the scope and destination

Identify what requests should select this skill, what outcome it should produce and which task-specific facts the agent needs. Use the conversation and existing files before asking for missing details.

Inspect relevant installed skills with `list_skills` and `load_skill` before creating a duplicate. For an update, read the current SKILL.md and resources before editing them. Keep instructions that still serve the workflow.

Use the user's requested location. Otherwise create a project skill at `<project_root>/.agents/skills/<name>/SKILL.md`. MECA also discovers personal skills at `~/.meca/skills/` and shared skills at `~/.agents/skills/`. Keep writes within the current tool access and approval policy.

## Write a valid SKILL.md

The [Agent Skills specification](https://agentskills.io/specification) defines the portable format. Write UTF-8 Markdown beginning immediately with YAML frontmatter between `---` lines.

- `name` is required: 1–64 lowercase ASCII letters, digits or hyphens. Match the containing directory. Do not use underscores, leading/trailing hyphens or consecutive hyphens.
- `description` is required: a nonempty string of at most 1024 characters. State the capability and the requests that should select it; use concrete task terms.
- Optional `license` identifies licensing. Optional `compatibility` describes actual environment requirements in at most 500 characters. Optional `metadata` maps string keys to string values. Omit unused fields.
- `allowed-tools` is experimental and client-dependent. Do not rely on it to authorize execution. MECA does not interpret it as an execution policy.

For example, create `review-changes/SKILL.md`:

```markdown
---
name: review-changes
description: Review a source diff for correctness and failure handling when the user requests a code review.
---
# Review changes

Read the diff, affected callers and relevant tests. Trace changed behavior through the real execution boundary.
Report actionable defects with file locations, their observable effect and supporting evidence.
Distinguish verified behavior from assumptions and checks that could not run.
```

Quote YAML values containing special syntax, such as `: ` or `#`. Reject duplicate keys and unfinished placeholder content.

## Make the instructions useful

Keep the entrypoint concise. Explain the desired result, decision criteria, non-obvious constraints, relevant tools and how to verify the result. Give concrete input/output examples when they resolve ambiguity. Leave room for judgment when several approaches are valid.

Put selection criteria in the description: the body is read only after selection. Avoid generic advice, repeated policies, hardcoded machine paths and constraints copied from an unrelated example. Include failure handling for the workflow's actual I/O or side effects. A skill must not imply permission to publish, install dependencies or modify unrelated state.

## Add resources only when needed

Metadata supports discovery, SKILL.md supplies instructions, and supporting files are read on demand. Keep SKILL.md under 500 lines and move substantial conditional detail into linked resources.

- `references/`: task-specific documentation. Link from SKILL.md using a skill-root-relative path and explain when to read it. Prefer direct references over nested chains.
- `scripts/`: repeatable operations that benefit from deterministic execution. Document real dependencies and run meaningful tests. Reuse the project's language and tools where practical.
- `assets/`: templates or other files used to produce the deliverable.

Do not create empty directories, duplicate README files or helper scripts without a real use. Keep credentials and live application state out of the skill.

## Check it in MECA

MECA reads instructions; it does not execute bundled scripts automatically. Use `read_skill_file` for referenced text or assets, and existing execution tools when a requested operation needs a script. `!skills` previews installed skills; `#<name>` explicitly includes one in a request. These are MECA conventions, separate from the portable file format.

Load the completed skill with `load_skill` using its directory or SKILL.md `path`. Check that metadata parses and that each referenced file exists and can be read with `read_skill_file`. After editing an installed skill, list it again to check its description and source. If the optional `skills-ref` validator is already available, `skills-ref validate <skill-directory>` provides another format check; it is not a MECA runtime dependency.

Test the skill against a representative request and inspect whether its instructions lead to the intended result. Run any added scripts in a suitable test workspace. Report the final path, invocation, checks performed and remaining external requirements. Format validation alone does not prove agent behavior or task success.
