# Calendars and events

Use this when people create, find or attend events and need their time, location or attendance state to stay understandable across changes.

[Do & Don’t](do-and-dont.md) · [Implementation Q&A](implementation-qa.md)

## Main flow

1. Open the relevant calendar or event list and resolve the intended time range in its display zone. Keep all-day events identifiable.
2. Create a single event or a recurring series. Show the organizer, the event's intended time basis and those invited where appropriate.
3. Open an occurrence to change attendance, time or cancellation. Dragging or resizing an occurrence can propose a new time. Use the same change operation as the editor and confirm occurrence or series scope when it applies. Notify affected people from the accepted change.
4. Reconcile displayed events, invitations and calendar integrations by identity and revision after a change, retry or reconnect.

## Implementation boundaries

- Represent date-only events, fixed instants and recurring local times according to their meaning. Preserve the organizer's time zone when a series follows local civil time.
- Identify an occurrence by its series and original occurrence key. A moved instance can retain identity when its displayed start changes.
- Give organizer changes, per-person attendance and external invitation delivery their respective outcomes. Bound expansion of open-ended series to the requested range.
- Set rules for edit scope, conflicts, reminders and event visibility. Restrict calendar data at read, export and invitation boundaries.

## Borrow for this workflow

Use [Booking](../booking/app-guide.md) when a slot or resource must be allocated exclusively. Use [Forum](../forum/app-guide.md) for an event discussion and [Messenger](../messenger/app-guide.md) for notification delivery and retries.

Google Calendar Help “Create a recurring event” describes changing one occurrence in a repeating event.

## Acceptance cases

- Move one occurrence and verify the series, direct links and attendance follow the chosen scope.
- View an all-day event in different zones and a recurring local-time event across a clock change.
- Cancel an occurrence while invitation delivery is delayed. The current event and later notices reflect the accepted cancellation.
- Propose a move into an unavailable time. Keep the confirmed occurrence and explain the conflict without sending an update.
