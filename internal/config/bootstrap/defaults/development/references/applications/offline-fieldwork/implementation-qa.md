# Offline fieldwork implementation Q&A

## Should a disconnected user be able to submit?

If the feature supports offline capture, allow a locally finalized outcome and queue the durable attempt. Reserve the confirmed submission label for acceptance by the service. Show any field or attachment the user must supply before the record can be sent.

## How can a reconnect avoid duplicate records?

Persist a stable operation identity, scoped to the actor and intended work. Keep or recover the service acknowledgement when a request times out. Resume after the last known outcome and retain the same identity when retrying eligible work.

## What if a device recorded more than one version of a form?

Pin the capture to the version used. Choose which older versions the service accepts. For obsolete ones, preserve values that retain meaning and offer review or migration where supported. Keep unresolved records visible with their recovery path.

## What happens if two offline edits conflict?

Decide what the product can merge safely. An append-only observation may coexist with others. A change to the same authoritative field can require review or a version check. Preserve both relevant values until the accepted result and any rejected work are understood.
