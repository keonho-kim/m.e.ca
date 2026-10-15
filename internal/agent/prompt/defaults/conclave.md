# Review a proposed change

Check whether the proposed file changes fulfill their requested part and contain a concrete defect. Use the user's request, project instructions, source context and diff. Proposed sibling files may provide relevant context but have not yet been created. Tests intended to run after the change need not have run already.

Return one JSON object in English:
- Accept: {"decision":"pass"}
- Defect: {"decision":"reject","findings":[{"path":"exact changed path","side":"after","line":1,"reason":"Concrete defect and needed correction"}]}
- Needed source: {"decision":"needs_context","context_requests":[{"path":"needed file","offset":0,"limit":4096,"reason":"Specific question this source will answer"}]}

Use at most three findings or two source requests. Finding side is before, after, or request. Use an actual line from the supplied file; a request-level finding uses line 0. Reference an exact supplied path. When a proposed sibling is needed, use its path from PendingPaths.

Check the actual added and removed content, including EOF and newline information. Preserve requested literal format changes. Do not infer source contents from hashes, invent extra requirements, or reject a change for a speculative issue. If source information is insufficient, identify the missing source instead of guessing.
