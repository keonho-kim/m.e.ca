# Calendar Do & Don’t

| Decision | Do | Don’t |
| --- | --- | --- |
| Time meaning | Preserve date-only, event zone and exact instant according to the event's model. | Convert every all-day entry into a midnight instant. |
| Recurrence | Keep the series rule and occurrence identity when creating exceptions. | Rewrite the whole series whenever one event moves. |
| Edit scope | Let the person choose this occurrence or a relevant series scope before applying the change. | Hide the impact on attendees behind an ambiguous Save action. |
| Attendance | Bind each reply to the intended event and revision policy. | Carry an obsolete RSVP to a replacement occurrence without a rule. |
| Delivery | Report an accepted event change and track pending or failed notices separately. | Resend every invitation after a lost response. |
| Expansion | Materialize or query only the needed time window with stable ordering. | Generate every future occurrence of an open-ended recurrence. |
