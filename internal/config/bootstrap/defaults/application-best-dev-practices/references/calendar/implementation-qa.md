# Calendar implementation Q&A

## Which time values need to be stored?

Store a date for all-day semantics, an instant for an absolute event, and a local recurrence rule plus time zone when a series follows a wall clock. Display the relevant time zone and choose an explicit policy for skipped or repeated local times around clock changes.

## What is the identity of a moved occurrence?

Keep a key based on its series and original occurrence. Save new display times as an exception. That identity keeps cancellation, RSVP, direct routes and integrations attached to the same occurrence after a move.

## Should a calendar forbid overlap?

Allow overlap where the calendar is for planning or recording events. When accepting a reservation claims limited availability, enforce the invariant at the booking authority. Define what tentative calendar entries and confirmed allocations promise.

## How should synchronization handle a late update?

Apply a source's changes within its ownership boundary. Compare revisions or reconcile with authoritative state before applying ambiguous updates or deletes. Preserve a useful visible state if an external source has delayed or stopped syncing.
