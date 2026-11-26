# Application Best Dev Practices

Start with one person completing one task. Trace their action through the interface, the operation that accepts it, the data that establishes its outcome, and the feedback or recovery path they see. Keep the requested scope and the project's existing stack, policies and state owners.

## Combine guides by feature

1. **Choose the main workflow.** Pick the app guide closest to what the person is trying to complete. A feature such as attachments or scheduling can start directly at its guide.
2. **Read the relevant contracts.** Each app folder has an app guide for the workflow and implementation boundaries, Do & Don’t for common decisions, and implementation Q&A for difficult alternatives and edge cases. Read the parts that affect the current work.
3. **Borrow a specific capability.** Describe the behavior needed, the record and actor it belongs to, its authority, and what its outcome means to the main workflow. Apply that interface, state or recovery pattern where it fits. Use the main workflow's access and retention rules for the combined feature.
4. **Check the join.** Walk through success and the applicable failure cases across the combined workflow. Check target identity, permission changes, retries or conflicts, and what the user sees when one part succeeds and another is still pending or fails.

Treat the app guides as decision aids. When correctness depends on a condition, state the condition with the rule. Add optional stages, automation or integrations when the task calls for them. Use bundle-relative references and record service examples by name and documented feature.

When drag and drop, a context menu and a visible control request the same change, route them through one operation with the same target, permissions and outcomes. The [UI guide](ui.md) covers invoking, previewing and completing those interactions. Use [backend design](backend.md) to make their state, authorization and retry guarantees authoritative. Use [algorithms and data structures](algorithms.md) when computation or representation drives the cost.

## Use cases and representative services

| User task | Observable service element to study | App guide |
| --- | --- | --- |
| Exchange messages and reply in context | Slack groups replies into threads around messages | [Messenger](applications/messenger/app-guide.md) |
| Publish a topic, discuss it or answer a question | GitHub Discussions offers Q&A categories with accepted answers | [Forum and Q&A](applications/forum/app-guide.md) |
| Follow people and publish to an audience | Mastodon lets authors choose a post's visibility | [Social feed](applications/social-feed/app-guide.md) |
| Track, assign and move work through stages | Trello cards can move within and between lists | [Project and Kanban](applications/project-management/app-guide.md) |
| Receive, resolve and follow up on a support request | Zendesk tickets expose public replies and internal notes as distinct actions | [Customer support](applications/customer-support/app-guide.md) |
| Edit, review and recover a shared document | Google Docs exposes version history and restoration | [Collaborative documents](applications/collaborative-docs/app-guide.md) |
| Organize files and grant access to other people | Google Drive exposes file access roles such as Viewer and Editor | [File sharing](applications/file-sharing/app-guide.md) |
| Collect structured responses with conditional questions | Google Forms can route respondents to a section based on an answer | [Forms and intake](applications/forms/app-guide.md) |
| Arrange repeated events and invite participants | Google Calendar supports updating one occurrence of a repeating event | [Calendar and events](applications/calendar/app-guide.md) |
| Study material, submit attempts and view progress | Moodle Quiz exposes attempt limits and grading choices | [Learning and quizzes](applications/learning/app-guide.md) |
| Buy, fulfill, cancel or return specific goods | Shopify supports partial fulfillment of an order | [Commerce and marketplace](applications/commerce/app-guide.md) |
| Explore indicators and export a scoped result | Metabase dashboard filters connect to selected cards | [Reporting and dashboards](applications/reporting/app-guide.md) |
| Upload, process, publish and play media | YouTube exposes processing and visibility during the upload workflow | [Media](applications/media/app-guide.md) |
| Capture observations and send them after reconnection | ODK Collect distinguishes drafts, finalized forms and sent submissions | [Offline fieldwork](applications/offline-fieldwork/app-guide.md) |
| Converse with an assistant and supervise work | MECA presents streamed responses, tool execution and request cancellation | [AI conversation and tasks](applications/ai-workspace/app-guide.md) |
| Operate records, member roles and approvals within an organization | GitHub organizations and repositories expose different access roles | [Administrative workspace](applications/admin-workspace/app-guide.md) |
| Select and confirm a scarce appointment | Google Calendar appointment schedules expose a shareable booking page | [Booking](applications/booking/app-guide.md) |
| Prepare drafts, publish revisions and restore earlier content | WordPress exposes post revision comparison and restoration | [Content publishing](applications/content-publishing/app-guide.md) |

## Combinations to study

- **Slack connects conversation and work tracking.** Messages can become Lists items, and each item can have its own thread. Give the linked work item its own completion state and permissions.
- **GitHub Discussions can create an issue from a discussion.** Track the issue independently and retain the connection for users with access to both records.
- **YouTube combines media processing with publication controls.** Model playable renditions and publication visibility explicitly. Preserve both decisions through retries.
- **ODK Collect combines structured intake with offline submission.** Bind answers to the form version and track which records were finalized locally or sent.

## Worked use cases

**Community events.** Use the [Forum guide](applications/forum/app-guide.md) for the discussion, [Calendar](applications/calendar/app-guide.md) for occurrence times, and [Booking](applications/booking/app-guide.md) for capacity. Point the discussion at the occurrence and keep the reservation as the authority for a seat. Verify simultaneous requests for the last seat and cancellation of the shared occurrence.

**Support portal with attachments.** Use [Customer support](applications/customer-support/app-guide.md) for ticket visibility, [Messenger](applications/messenger/app-guide.md) for pending and retried delivery where relevant, and [File sharing](applications/file-sharing/app-guide.md) for protected files. Bind each attachment to the reply or note it belongs to. After permission changes, verify both direct downloads and previously generated previews.

**Offline inspection and reporting.** Use [Offline fieldwork](applications/offline-fieldwork/app-guide.md) for the local queue, [Forms](applications/forms/app-guide.md) for validation, and [Reporting](applications/reporting/app-guide.md) for the accepted dataset. Show queued capture on the device. Count the accepted revision once when a reconnect or retry repeats the submission.

## Verify the requested workflow

Record the relevant actors, affected records, accepted outcome, visible feedback and recovery. Derive acceptance cases from the applicable boundaries and run the checks available in the target project. For design or review work, describe the proposed verification and keep the findings within the requested scope.
