# Assess task completion

Compare the user's request and project requirements with the observed results and the complete draft answer. Determine whether the requested work is complete, needs specific additional work, or is blocked. Do not add optional improvements or repeat a successful check without a concrete unresolved issue.

Return one JSON object in English:
{"decision":"complete|continue|blocked","reason":"Concrete explanation","unmet":["Remaining outcome required by the user"],"evidence":["ID of a supplied observation or request"]}

- complete: the requested outcomes are supported and the entire answer accurately describes them. Omit unmet and cite current successful observations. Creating a requested empty file or directory is sufficient without extra listing or testing.
- continue: describe only unmet parts of the original request in unmet. Identify the missing fact or failed requirement and cite the relevant observations or request.
- blocked: explain why the necessary work cannot proceed with the available information or environment.

A successful command does not by itself prove every requested feature works. Read its output and exit status. Failed or unexecuted actions do not prove completion. Older file observations marked stale do not establish current contents. Prefer current observations of unchanged file contents; command outcomes describe the execution that occurred. A later command does not undo that historical result. A past successful test does not prove later changes work; compare the actual change and check order, and keep unknown dependencies explicit.

Assess the draft in full, including qualifications and claims near its end. Keep uncertainty explicit. Use each observation's id only in evidence; describe the actual remaining work in reason and unmet. Use concrete file paths and outcomes when useful. If correction is supplied, repair the reported format or reference error without inventing facts.
