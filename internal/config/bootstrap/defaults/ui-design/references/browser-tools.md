# Browser verification and optional diagnostics

Use Playwright CLI for rendered UI checks. Inspect `playwright-cli --help` or the configured project-local CLI for available commands. Do not fetch a CLI implicitly when checking whether it is installed.

If no usable Playwright CLI is installed, ask the user in their language to install it **because it is needed to run the UI layout and motion verification tools**. Link the [official installation instructions](https://github.com/microsoft/playwright-cli#installation); `npm install -g @playwright/cli@latest` is an example the user can run. Do not install or upgrade it automatically. Honor an explicit user request to perform installation under the project's existing policies. Continue independent work, but mark browser and diagnostic checks as not run until the tool is available.

Start the target project's dev/static server and confirm its actual HTTP URL. Choose a task-unique session; the following name, URL and dimensions are illustrative:

```sh
playwright-cli --help
playwright-cli -s=ui-review open http://127.0.0.1:3000
playwright-cli -s=ui-review resize 1440 900
playwright-cli -s=ui-review snapshot
# Exercise the relevant flow using current snapshot references before capturing.
playwright-cli -s=ui-review screenshot
playwright-cli -s=ui-review console error
playwright-cli -s=ui-review requests
playwright-cli -s=ui-review close
```

Use current snapshot references with `click`, `fill` and `press`; refresh references after view changes. Inspect screenshots at relevant sizes and investigate console/network failures. Wait for observable content and fonts to be ready, not arbitrary sleeps. Keep captures and logs in project-approved, task-owned locations; do not persist credentials or unrelated browser state. Close only the session and server owned by this task, including on failure. Existing Playwright Test suites supplement this inspection. CLI and browser availability are development prerequisites, not runtime dependencies of MECA or the target app.

## Run the relevant diagnostic

The optional [layout check](../scripts/layout-check.js) and [motion check](../scripts/motion-check.js) are single Playwright function expressions. Read the selected skill's resource with `read_skill_file` before execution. For the bundled `meca:ui-design` source, its installed root is `~/.meca/skills/ui-design/`; do not assume that path for a project or shared skill. An opaque skill ID is not a filesystem path. If the selected resource's actual path cannot be established, do not substitute another skill's script; report that diagnostic as not run.

The examples below use the bundled source and the already-open session. Check that the installed CLI supports `run-code --filename` and `--json`; if it does not, explain the missing capability and request a compatible installation. Use the project's configured executable in place of `playwright-cli` when appropriate.

```sh
playwright-cli -s=ui-review --json run-code \
  --filename="$HOME/.meca/skills/ui-design/scripts/layout-check.js"
```

The layout check samples visible buttons and button roles in the current viewport for content-height and horizontal-overflow candidates. It reports `checked`, `findings` and `uninspected`; it cannot establish accessibility, visual balance or whether truncation was intentional. It does not inspect iframe contents, unsupported text or controls outside the viewport. Moving or unsupported controls, bounded scans and omitted findings make the result inconclusive. It checks at most 200 controls and 3,000 DOM elements, bounds the text sampled in each control and returns at most 20 findings.

For a relevant motion, start the observer, perform the actual authorized action with the current snapshot, then stop it on the same page:

```sh
playwright-cli -s=ui-review --json run-code \
  --filename="$HOME/.meca/skills/ui-design/scripts/motion-check.js"
playwright-cli -s=ui-review click e42
playwright-cli -s=ui-review --json run-code \
  'async page => page.evaluate(() => window.__mecaUIDesignMotion?.stop() ?? {status: "inconclusive", reason: "recording was lost"})'
```

`e42` is an example: use the current snapshot's actual reference. The script observes visible buttons, links, form controls and elements with an ID, class or role. It records no text, does not trigger the action, and limits observation to 15 seconds, 900 frames, 120 visible elements and 3,000 scanned DOM elements. It stops and releases recorded geometry when finished; an automatically stopped report expires. A page change loses the observer and is inconclusive. Gaps, sparse frames, limits, document scrolling, resizing, iframes or no observed movement also make the result inconclusive. Nested scrolling can produce candidates requiring manual review. It cannot inspect painted pixels, unchanged boxes with changing content, or elements never sampled.

Each diagnostic returns `issues`, `no-issues-detected` or `inconclusive`; motion start returns `recording`. CLI process success does not mean the UI passed. Parse the CLI result and then its reported status; current `--json` versions may wrap the returned object as a JSON string in `result`. Findings are candidates, not automatic defects, and an empty list or a screenshot does not certify the design. Inspect relevant frames at actual speed and at useful widths, including the start, action, result, interruption, reduced motion and any loop boundary. Recheck after corrections and report what was not observed.
