# Booking Do & Don’t

| Decision | Do | Don’t |
| --- | --- | --- |
| Availability | Treat the displayed window as a view of eligibility and current allocations. | Guarantee a selection from an old display snapshot. |
| Confirmation | Enforce capacity and accepted terms at the authoritative operation. | Add a confirm step that can double-book under simultaneous requests. |
| Holds | Add a hold when a step needs reserved capacity. Specify owner, expiry and release. | Leave inventory unavailable indefinitely after abandonment or failure. |
| Reschedule | Define whether old and new allocations trade together or need visible reconciliation. | Release the old booking accidentally if the new one fails. |
| Cancellation | Release capacity according to the accepted outcome and protect already completed effects. | Increase capacity again whenever a cancel request is replayed. |
| Privacy | Expose eligible slots and scoped receipts with the intended audience. | Disclose other people's details through the availability calendar. |
