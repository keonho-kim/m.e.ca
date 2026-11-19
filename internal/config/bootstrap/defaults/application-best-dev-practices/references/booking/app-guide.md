# Booking, appointments and capacity

Use this when a person selects a service, time or shared resource and expects a confirmed allocation. Define capacity and the exact operation that can award it.

[Do & Don’t](do-and-dont.md) · [Implementation Q&A](implementation-qa.md)

## Main flow

1. Choose the service or occurrence and show eligible options in the relevant time zone. Expose terms, available quantity and any eligibility constraints the person must know.
2. Select a slot and collect needed details. If a hold is part of the flow, show its scope and actual expiry.
3. Confirm under current capacity, eligibility and terms. Give a receipt or resolving state for a known or uncertain result and preserve valid details when a slot becomes unavailable.
4. Cancel or reschedule through a supported path. Show what happens to the old allocation, the new one and notifications.

## Implementation boundaries

- Derive availability from the defined schedule and existing allocations. Recheck and claim the actual capacity at acceptance. Use the appropriate atomic constraint or equivalent race-safe operation.
- Give an availability quote, hold and confirmed reservation their own meanings. Model hold expiry and release when a multi-step flow needs a hold.
- Reconcile the same booking attempt after a lost response. If an external side effect matters, track it and correlate its outcome with the booking.
- Define who can view, book, cancel and reschedule. Keep participant details scoped and specify what calendar or notification subscribers can see.

## Borrow for this workflow

Use [Calendar](../calendar/app-guide.md) for recurrence and time-zone display. Use [Forms](../forms/app-guide.md) for participant details and [Commerce](../commerce/app-guide.md) when accepted terms and payment need separate states.

Google Calendar Help “Learn about appointment schedules in Google Calendar” describes sharing a booking page based on availability.

## Acceptance cases

- Two people request the last unit of capacity. At most one eligible reservation is accepted for that unit.
- The response disappears after confirmation. Retrying recovers the same reservation and outcome.
- A hold expires or a reschedule loses the new slot. Protect the old or new allocation according to the configured policy and give a recovery path.
