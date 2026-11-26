# Algorithm techniques

Start with a simple reference computation or a small hand-checkable example. State what a candidate must preserve, including tie-breaking, ordering, numeric precision and any permitted approximation. Express costs using the dimensions that vary independently, such as records, edges, distinct keys, requested results and input bytes.

For concrete construction steps, use [dependencies and allocation](dependency-and-allocation.md) for graph scheduling, incremental evaluation, matching and budget selection. Use [ranges and edits](ranges-and-edits.md) for sweeps and incremental parsing, and [scheduling and streams](scheduling-and-streams.md) for bounded selection and merge processing.

## Choose the structure of the computation

| Situation | Technique and decision |
| --- | --- |
| The next choice is safe for every remaining input | Use a greedy method after showing the invariant or exchange argument. Earliest finishing time maximizes the count of nonoverlapping unweighted intervals. Weights change the problem and can justify dynamic programming. |
| Different paths revisit the same subproblem | Define a sufficient DP state, base cases and transitions. Count reachable states and transitions per state. Use top-down memoization for sparse reachability or bottom-up order for dense predictable states. Keep only the previous frontier when reconstruction and transitions allow it. |
| An exact subset search is exponential but the input has a moderate number of elements | Consider meet-in-the-middle by enumerating each half and matching compatible summaries. Count the roughly `2^(n/2)` entries per half, the join/search cost and memory. Summaries must retain everything needed for constraints, duplicate handling and any result reconstruction. |
| Subproblems are independent until their results combine | Use divide and conquer when splitting and merging reduce total work. Include split/merge costs, recursion depth, skew and the size of intermediate results. Use a straightforward base case for small inputs. |
| Resources must be assigned across eligibility edges and capacities | Model a matching or flow problem when local greedy choices can block a valid global allocation. Distinguish maximum count, minimum total cost and required fairness. Check whether constraints fit the selected model and budget the graph size and solver cost. |
| Feasibility changes once over an ordered domain | Search the boundary with a monotone predicate. Prove the direction, feasible/infeasible sentinels and termination. Total work includes each predicate evaluation. Use bounds and midpoint arithmetic that fit the domain. |
| Queries need counts or sums over many static ranges | Build prefix aggregates where the operation supports subtraction or another inverse. For non-invertible queries or updates, use the data structure that supports the required combine and update operations. |
| A window can be adjusted monotonically | Use two pointers when each endpoint only moves forward. For example, a sum limit with nonnegative elements supports a shrinking left edge. Negative elements can invalidate that choice. For exact subarray sums, prefix-sum frequencies work with negative elements too. |
| Intervals or events share a coordinate or time order | Use a sweep with sorted start/end events and an active set. Define inclusive/exclusive boundaries and ties before sorting. With half-open intervals, process endings before starts at the same coordinate when tracking simultaneous activity. |
| One text is matched against many fixed patterns | Consider Aho–Corasick when repeated independent searches dominate. Budget the automaton, preprocessing and reported matches. Define the input units, normalization, overlapping matches and duplicate patterns. Bound output when the caller only needs a summary or a prefix. |

## Choose graph traversal by the edge contract

- Use breadth-first search for unit-cost shortest paths. For weights 0 or 1, a deque supports 0–1 BFS by putting zero-cost updates at the front and unit-cost updates at the back.
- Use topological processing for an acyclic dependency graph. Detect cycles when the input can violate the DAG requirement. One edge pass can propagate constant-cost per-edge state, including shortest paths with signed weights in a DAG.
- Use Dijkstra for nonnegative weights. With a binary heap and one updatable entry per vertex the bound is `O((V + E) log V)`. For a lazy heap, skip stale distances and budget queued entries and heap operations in terms of relaxations. [MIT 6.006 priority queue analysis](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/ec187276db0faaa047d655979a44de7f_MIT6_006S20_r13.pdf)
- If negative edges are valid and cycles are possible, account for reachable negative cycles and undefined finite shortest paths on affected routes. Bellman–Ford is a simple baseline with `O(VE)` time. Use a cheaper algorithm only when the edge/graph constraints support it.
- Distinguish reachability, shortest paths and undirected connectivity. Disjoint sets track components under additions and serve Kruskal's minimum spanning tree algorithm. They cannot answer directed reachability or remove arbitrary edges.

## Compare using the actual output

For a read-heavy dependency graph, determine if callers need one reachable path, all downstream nodes, or repeated reachability queries from a small set of roots. A per-root walk uses `O(V + E)` time with an adjacency list and avoids storing all pairs. Reusing a rooted result can save walks when the graph version stays stable and roots recur. A full reachability table has quadratic space in the worst case, even when the input graph is sparse.

Verify optimized results against the simple implementation on small generated and boundary inputs. Include empty inputs, ties, disconnected nodes, cycles, numeric limits and worst-case shapes when they apply. Measure output-sensitive algorithms at realistic result sizes. For approximate results, specify allowed error, deterministic expectations and how accuracy will be measured against an exact reference.
