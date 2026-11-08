# Step Status: truthful state of a multi-stage task

For an actual stage-based task, expose Step Status so the person can identify what is done, current, waiting or blocked. It describes the workflow/job's actual state. A stepper adds navigation when stages can be visited; a progress bar measures work when a numerator/denominator exists. Neither supplies missing stage evidence.


## Prioritize what the person needs to know

Use stage labels and state words that explain actual work, blocked dependencies and available recovery. A short operational list may need compact rows; a complex failure may need vertical detail. Do not add balanced segments or a duration estimate for visual completeness.

Use current, pending, confirmed and uncertain treatments consistently. Category accents and selected navigation are separate from success/error. A progress track can accompany the status only when its measured scope is known.

Keep consequential uncertainty and recovery durable. A heartbeat, a visited stage or a finished visual transition does not supply missing completion evidence. Show only states supported by the task rather than forcing this reference's entire vocabulary into a new API.

## Anatomy and states

Use meaningful ordered stage labels, visible status text/icon, current stage when known, relevant explanation and supported recovery. Stable task and stage identity bind updates to the correct work.

| State | Meaning and presentation |
| --- | --- |
| Queued | Stage has not started; no completed check or invented duration |
| Running | Owner reports current work; optional restrained activity cue with a static equivalent |
| Completed | The relevant owner confirmed this stage's actual completion condition |
| Error / blocked | Stage cannot proceed; explain the real cause and available recovery |
| Cancelled | A cancellation outcome was acknowledged, not merely requested by the client |
| Skipped / optional | Stage was deliberately omitted under the actual workflow; distinct from completed |
| Unknown / delayed observation | State cannot currently be established; retain last confirmed state with the observation limitation |

These are UI meanings to map from the existing service, not a new required API enum. Do not silently translate an unsupported server value into completed or queued. Connection status and heartbeat are separate from stage state. A locally valid form stage is not necessarily persisted, and a confirmed stage is not necessarily a finished job.

## Adapt the representation

Use vertical detail for long Korean labels or recovery, compact rows for an operational list, or a segment/connector form when it clarifies sequence. Include text beyond color. An order tracker can show actual chronological events; an editable wizard can show local prerequisites alongside its navigation. Do not add stages just to create a balanced graphic.

An ordered list with appropriate `aria-current="step"` is often sufficient. Non-navigable status should not become dead buttons. Report meaningful state/result changes politely without announcing every animation frame or streamed token. Keep cancel/retry controls separate, scoped and truthful.

Motion can reveal an actual update or change emphasis, never create completion. Reduced motion keeps static current/state information. Handle event order, replay, changed task identity, partial failure and reconnect through the existing contract; a late update from task A cannot repaint B. A lost observer is not proof of stopped work, and retry cannot blindly replay uncertain side effects.

The approval [document](../../assets/examples/approvals/request.html) shows a static review chain, not measured execution progress. The database workbench is a design shell and does not emit execution events. The [motion lab](../../assets/examples/motion/motion-lab.html) separately labels manual presentation previews and actual local loading. Bind real service events when adapting the representation.
