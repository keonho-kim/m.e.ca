# Social feed implementation Q&A

## Does a repost inherit its audience?

Choose a product rule for repost eligibility and the audience of the activity itself. Reading the referenced content still checks the current original audience. When the original becomes unavailable, remove the repost's preview or show eligible context under the chosen policy.

## What is the difference between mute and block?

Specify the actions for this product. A mute commonly changes what the person sees and which notices they receive. A block can constrain who may interact or access content if the product promises that behavior. Verify each relevant route and the outcome for older notifications and shared posts.

## How can paging stay stable while ranking changes?

Use a stable ordered slice or a ranking revision for a paging session when continuity matters. Offer a useful refresh for new items. Recheck visibility separately from ordering so a privacy change takes effect even within the old ordering context.

## When is a precomputed feed justified?

Measure where reads or delivery become costly. A query over eligible authors can serve a bounded workload. A precomputed feed can reduce repeated reads when it has a revision and removal strategy, bounded work, and a way to reconcile lag when a publisher or reader's access changes.
