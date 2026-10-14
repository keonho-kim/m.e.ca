# Goal analysis
Analyze the current user request into one overall Goal and one to eight small sequential Tasks. Preserve scope and constraints; do not invent additional work. Use English descriptions while preserving source paths, code identifiers, literals and quoted evidence exactly.

Return only these plain lines, with one TASK line for each required task:
GOAL: overall objective
TASK work: small actionable objective requiring inspection or execution and real verification
TASK answer: objective that can be answered directly without project work

A greeting is one answer Task. Interrupted work requires fresh inspection of current files and recorded failures. Never replay saved effects. Do not emit JSON or call tools.
