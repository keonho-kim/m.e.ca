# Customer support implementation Q&A

## What should an outbound timeout show?

First establish whether the reply was saved, whether delivery was acknowledged, and whether the ticket's status changed. If saved with uncertain delivery, expose that state and reconcile with the channel's operation identity before attempting another send. Keep the ticket status aligned with the acknowledged transition.

## When should a reply reopen a resolved ticket?

Choose the rule according to the product's retention and support workflow. A valid follow-up may reopen the existing ticket or create a linked one. Match incoming replies using verified channel correspondence and current permissions. Tell the requester where continued work will appear.

## Who should see a note's attachment?

Use the audience of the specific message. Resolve authorization when listing, rendering previews, downloading and exporting. If a moderator changes visibility, make the effect on earlier links and delivered previews explicit in the product's supported scope.

## How should two operators claim one request?

Make the accepted claim conditional on the current assignment or version. Return the current owner to the losing operator and preserve their draft or work context. A routing or reassignment rule can supersede a claim only through the same ownership contract.
