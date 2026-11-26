# Booking implementation Q&A

## When is a selected time actually reserved?

A selection is an intent. Accept a reservation only after the capacity check and claim succeed. A separately modeled hold may reserve capacity before final commitment. Give it a bounded lifetime and let the confirmation operate on the same intended allocation.

## Do all booking flows need temporary holds?

A short atomic confirm can serve a flow that needs only a small form. Add holds when a payment, review or long checkout needs a bounded reservation. Define clock authority and what happens when confirmation and expiry race.

## What if payment succeeds after a hold expired?

Reconcile the admission and provider results separately. The acknowledged payment alone cannot create capacity that is gone. Apply the product's supported recovery, including an eligible replacement or compensation where justified, and keep the person informed of both outcomes.

## How should reschedule deal with the previous reservation?

If the product promises to keep the current booking until a new one succeeds, enforce that invariant at the accepting operation or implement equivalent recovery across the relevant authority. Report which allocation is current and when notifications follow.
