# Commerce implementation Q&A

## Which information should be frozen when accepting an order?

Identify what the buyer agreed to, such as quantity, price, currency, taxes or adjustments and relevant delivery terms. Store enough to explain the purchase and later recovery. Record a new proposal when the quote changes before acceptance and require any necessary confirmation.

## What if payment and order creation report different outcomes?

Keep both operations correlated by stable identity. Verify authoritative outcomes and recover the accepted order or apply the product's defined compensation. Show a resolving state while uncertain. A provider timeout leaves the actual side effect uncertain until reconciled.

## How should partial cancellation or return affect the whole order?

Track eligible line quantities, associated delivery and financial effects. Derive the parent summary from accepted line results. Preserve fulfilled or intentionally retained items, and report unresolved quantities or a pending provider action separately.

## When does a cart reserve stock?

Choose a hold when the flow needs a bounded reservation during an intermediate step. Otherwise claim stock during the accepting operation. For a hold, define its duration, owner, release on abandonment or failure, and behavior when checkout races with expiration.
