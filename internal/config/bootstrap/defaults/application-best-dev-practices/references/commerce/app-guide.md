# Commerce and marketplace orders

Use this when a purchaser chooses goods or services and needs a reliable order, fulfillment and recovery path. Trace the accepted commitment through order lines, stock or availability, payment and delivery where they apply.

[Do & Don’t](do-and-dont.md) · [Implementation Q&A](implementation-qa.md)

## Main flow

1. Build a cart or quote under the correct buyer and seller scope. Show amounts, currency, availability and supported delivery terms.
2. Recheck the quote, stock and authority at checkout. Save the agreed order snapshot and show an acknowledged or still-resolving outcome.
3. Update payment and fulfillment from their respective evidence. Expose line-level results for split delivery, cancellation or return if supported.
4. Reconcile unknown writes and duplicate or out-of-order events. Give eligible buyers and operators a way to inspect the accepted result and supported recovery.

## Implementation boundaries

- Preserve the purchased description, amount, quantity and relevant terms at acceptance. Link current catalog details where useful.
- Identify each order, payment operation, shipment and return in its own scope. Choose how the parent status reflects partial results.
- Where stock or capacity is constrained, enforce its allocation at the actual write boundary and define reservation expiry or release.
- In a marketplace, enforce buyer and seller access at order and line level. Keep fulfillment, cancellation and settlement with their actual owners.

## Borrow for this workflow

Use [Booking](../booking/app-guide.md) for constrained allocations, [Customer support](../customer-support/app-guide.md) for follow-up and [Reporting](../reporting/app-guide.md) for scoped order or fulfillment views.

Shopify Help Center “Fulfilling orders individually” describes fulfilling selected items from an order.

## Acceptance cases

- A buyer retries checkout after the response is lost. The order and payment operations are recovered without duplicating commitments.
- The product changes after acceptance. Order history still describes the agreed purchase.
- Fulfill or return one line and verify quantities, summaries and seller-specific visibility.
