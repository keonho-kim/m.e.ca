# Progressive delivery, media and optimistic continuity

Show useful independent regions when ready and retain correct identity. Perceived-speed techniques cannot establish actual completion, permissions or persistence.

## Preserve the content hierarchy while it arrives

Choose which regions can be useful independently. Keep established context and geometry so progressive delivery does not become repeated page rearrangement. A media-led detail may reserve a large image and caption; a work queue may reveal rows while a secondary inspector is still pending. Neither implies waiting for the whole page to become ready.

Keep photograph, document and data roles clear. Genuine image color can supply variation on a neutral frame; action/selection accents should remain stable rather than change to match every newly loaded picture. Place decisive metadata where it remains available if media fails.

## Default content-image pipeline: Blur-up Image

Use Blur-up for content imagery by default: prepare a genuine small preview from the same original/crop, reserve its intrinsic aspect ratio, and reveal the full image when its load/decode is ready. Keep the meaningful alt/caption on the full content; the decorative preview is hidden from assistive reading. Do not use an unrelated blurred scene or stretched gradient as a fabricated preview.

When a preview is unavailable, derive it in the actual asset pipeline or state the fallback. A cached ready image, icon or already-available logo need not be delayed just to display a loading effect. Preserve critical first-screen priority; lazy loading is for appropriate secondary content, not a requirement of Blur-up.

Handle full-image error with a stable fallback and useful alternative text. A rejected decode must not leave an invisible image forever. Bind late load/decode results to the current image identity and release owned listeners/preview resources on teardown. Reduced motion removes the fade while retaining the same complete image/failure states.

An optional [media controller study](../../assets/examples/shared/media.js) and [loading lab](../../assets/examples/motion/motion-lab.html) pair a real derived preview with its original. Its local derivative and original demonstrate the relationship. Adapt to the target project’s asset pipeline; the study does not prescribe local-only hosting, a new CDN or its controller architecture.

## Progressive delivery, perceived speed and ownership

- Render available shell/text and independent regions without waiting for unrelated media. Reserve boundaries so incremental reveal does not scatter the page or move its main action.
- Lazy-load below-the-fold or secondary material where useful. Do not lazily defer the critical first-screen image by default. A placeholder cannot compensate for an unnecessary network waterfall.
- Use existing cached data with explicit scope and freshness; background revalidation is a data policy, not permission to show another user's content. Name cache ownership, invalidation and release before adding a cache or prefetcher.
- Use purposeful, bounded prefetching for a likely next step only when access, cost and freshness allow it. Do not fetch every record or hidden panel to make all skeletons disappear.
- Choose paging, load-more or infinite continuation by navigation needs. Virtualization addresses measured rendering volume, not request ordering; preserve row identity, selection, focus and accessible context.
- Debounce input frequency when appropriate, but still guard request identity. Bound subscriptions, preview URLs, retained collections and concurrent requests; release them with their actual owner.
- Restrict optimistic treatment to an operation with a credible pending/confirmation/reconciliation path. Financial commitment, destructive work and permission changes need their actual contract, not a decorative immediate checkmark.

## Visual choices

| Recipe | Useful situation | Guardrail |
| --- | --- | --- |
| Aspect-ratio Placeholder | Real media is not ready | Reserve actual geometry and provide a useful fallback; no fake product image |
| Blur-up Image (default) | Content imagery has its genuine small preview | Reveal the decoded original with one coherent alt/caption; provide an explicit fallback otherwise |
| Region Reveal | Independent content arrives | Local fade or immediate display within stable bounds; avoid replay on each refetch |
| Boundary Loader | Another result page is pending | Keep existing rows, focus and actual continuation scope |
| Retained-content Refresh | The same authorized resource is revalidated | Label actual applied scope/freshness; never relabel account A as B |
| Optimistic Local Change | A reversible operation has a real reconciliation path | Pending styling, targeted rollback and truthful confirmed/failed/unknown result |

A preview/image replacement must not restart layout, duplicate spoken content or force an animation for cached ready media. Failed images retain useful alternatives and task controls. Optimistic failure rolls back only the relevant change, preserving unrelated newer updates.

Reduced motion uses immediate media/reveal changes. Do not lazy-load critical first-screen media by habit, create a cache without invalidation or retain every preview to avoid a placeholder. Release image observers, object URLs, requests and animation handles on teardown. Measure loading waterfalls, layout shift and memory before adding a new prefetch or cache layer.

Source: [MDN lazy loading](https://developer.mozilla.org/en-US/docs/Web/Performance/Guides/Lazy_loading). These are delivery/composition decisions, not a requirement for a new data-fetching dependency.
