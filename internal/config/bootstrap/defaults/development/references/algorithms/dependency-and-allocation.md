# Dependencies and constrained choices

State the required result before selecting an optimizer. Model precedence, capacity and eligibility explicitly, and identify which facts can change between planning and execution. Use the [technique guide](techniques.md) for general algorithm conditions.

## Running work with prerequisites

- **Use when** builds, imports, reports or processing jobs have several prerequisite steps and independent branches can proceed concurrently.
- **Build** a graph with prerequisite-to-dependent edges, a successor list and each node's unresolved prerequisite count. Seed a ready queue with zero-count nodes. Release successors after a successful prerequisite completes, enqueueing each successor when its count reaches zero. Limit active workers separately from the graph. A topological scheduler's ready/done interface demonstrates this arrangement. [Python graphlib](https://docs.python.org/3/library/graphlib.html)
- **Cost and ownership.** Graph construction and readiness accounting use `O(V + E)` work with adjacency lists. Give graph transitions one owner, and account for each completion exactly once. Detect cycles before side effects where the workflow requires a fully valid plan. Define whether unrelated branches continue after a failure and mark descendants blocked or cancelled as appropriate.
- **Verify** a diamond dependency, duplicate edges, cycles, a failed prerequisite and repeated completion delivery. For `A → B`, `A → C`, and `B,C → D`, start D only after both B and C succeed. Use the [durable workflow guide](../backend/compositions/workflow.md) when work must resume after process loss.

## Recomputing only what a change affects

- **Use when** spreadsheets, dependency reports, derived configuration or document analysis repeat expensive computation after a small input change.
- **Build** reverse edges from an input to its consumers. Mark the changed input and reachable consumers dirty, then recompute in dependency order. Key each reusable result by all relevant inputs and options. Publish results for the graph/source revision used to compute them and discard a late result for an obsolete revision.
- **Cost and ownership.** Work can follow the affected subgraph, plus the cost of maintaining dependencies and versions. Stop propagating through an unchanged result only when its equality captures everything downstream observes. Record dynamically discovered dependencies, including those removed when a branch changes. Rebuild a sufficiently small graph when incremental maintenance costs more than its saved work.
- **Verify** a shared dependency, a changed branch, dependency deletion, cycles and edits during computation. Compare the incremental result with a fresh full evaluation. Give retained snapshots and derived results a release point when the owning document or operation ends.

## Assigning jobs to eligible resources

- **Use when** jobs, bookings or requests compete for resources with eligibility restrictions and a local first-fit choice can block a valid assignment.
- **Build** a bipartite graph of jobs and resources with edges for valid assignments. Use matching for one-to-one assignment. Use capacities when a resource can accept several jobs, and introduce costs only when the optimization objective is defined. For example, A can use X or Y while B can only use X. Assigning A to Y allows both requests to succeed.
- **Cost and ownership.** Count vertices and eligibility edges before choosing a maintained solver. Model the full constraints. Jobs spanning several time slots or consuming several independent budgets can need a richer optimization model. Distinguish required feasibility, maximum accepted jobs, minimum cost and fairness, and decide how competing objectives are ordered.
- **Verify** scarce resources, disconnected jobs, conflicting objectives and eligibility changes after planning. Revalidate and commit assignments at the authoritative boundary. A computed plan needs a concurrency-safe reservation step through the [transaction guide](../backend/storage/transactions.md).

## Choosing valuable work under a budget

- **Use when** a batch must select work with known costs and benefits under a capacity or time budget, and a simple sort cannot establish the requested optimum.
- **Build** the state from the actual constraints. For independent optional items with integer costs and one capacity, a 0/1 knapsack DP can track the best value at each reachable capacity. Process capacities in descending order for each item to prevent using that item repeatedly. For weighted nonoverlapping time intervals, sort by finishing time, find each interval's last compatible predecessor and compare including it with skipping it.
- **Cost and ownership.** Knapsack's cost depends on the numeric capacity as well as item count. Choose units carefully and include reconstruction memory if callers need the selected items. Weighted interval scheduling can use sorting and predecessor searches in `O(n log n)` followed by a linear DP. For a small item count and a huge numeric budget, compare sparse states or meet-in-the-middle with dense DP.
- **Verify** zero-cost items, ties, incompatible intervals and negative or missing estimates according to the input contract. Compare small cases with exhaustive enumeration. If estimates change or a solver budget ends, report the feasible result and its established quality. Recheck execution-time constraints before applying the plan.
