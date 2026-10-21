# Submit an event

The write path, end to end: a browser signs an event, the gateway forwards it, the
node accepts it and gossips it. This spec covers the two missing halves of that
path — a gateway that can reach a node, and a node that answers honestly about
what it did.

It does not cover validating anything. See [Out of scope](#out-of-scope).

## Problem Statement

The node already listens on gRPC and accepts an event. Nothing can reach it.

The gateway is a FastAPI service with two system routes — health and build
metadata. It has no route for publishing, no gRPC client, and no knowledge that a
node exists. So the flow stops one step after the browser signs: the event is
finished, correct, and undeliverable. A frontend developer has finished their
half of the protocol and hit a wall.

Three further problems sit behind that one, and they are what this spec is
really about:

**The node's answer is not actionable.** The existing response carries a single
boolean and a reason string. A duplicate and a first-time accept both come back
as accepted. That is correct — the protocol drops duplicates silently — and it is
also useless, because a publisher who needs to know whether to retry, whether to
show a spinner, or whether the event is already in the index cannot tell those
two apart. The gateway cannot fix it either, because it does not know which
happened.

**The gateway would break every signature it touches.** The event identifier is a
hash of the canonical form of the whole envelope. A gateway that accepts the
envelope as a JSON object and re-serializes it — which is what any framework's
model binding does by default — changes the bytes, changes the digest, and
invalidates the signature. The failure surfaces at the node, two steps from its
cause. The gateway has to be a pipe, and a pipe that re-serializes is not a pipe.

**There is no seam for validation.** Two doors lead into the node: the gRPC
service, and the Gossipsub subscription. They share no code, because there is no
code between them yet. When validation arrives, whoever writes it has to find
both and change both, and the failure mode of getting that wrong is silent — one
door validates, the other does not, and the network quietly disagrees with itself.
That is the specific failure this whole design exists to prevent.

## Solution

A publish route on the gateway, a gRPC client that reaches a node, and a node
that answers with an outcome the publisher can act on.

The gateway accepts the signed envelope as opaque bytes, checks only that it is
within the size ceiling, and forwards it verbatim. It never parses the envelope,
never re-serializes it, and never holds a key.

The node answers with one of four named outcomes — accepted, already exists,
orphan, rejected — and the gateway maps each to an HTTP status the caller can act
on without opening the body. When no node answers, the gateway says so with the
infrastructure error the service already has.

Both node doors call one submission function, so that validation has exactly one
place to land.

## User Stories

1. As a frontend developer, I want to submit a signed event over HTTP, so that the event I signed actually reaches the network.
2. As a frontend developer, I want the request body to be the exact bytes I signed, so that the node accepts an event whose signature I verified locally.
3. As a frontend developer, I want the gateway to forward my envelope without parsing it, so that no layer between me and the node can invalidate my signature.
4. As a frontend developer, I want one event per request, so that one bad event cannot poison the outcome of the others in a batch.
5. As a frontend developer, I want a distinct success response, so that I can show the user their review is published.
6. As a frontend developer, I want to be told when my event is a duplicate, so that I can show "already published" instead of a confusing second success.
7. As a frontend developer, I want to be told when my event is pending an unresolved reference, so that I can tell the user it is accepted but not yet visible in the index.
8. As a frontend developer, I want to be told why my event was rejected, so that I can show the user something better than "something went wrong".
9. As a frontend developer, I want a failed publish to return a non-2xx status, so that a client that only checks status codes does not treat a rejection as a success.
10. As a frontend developer, I want the outcome to be a named value rather than a free-text reason, so that I can branch on it instead of pattern-matching a string.
11. As a frontend developer, I want to be told the network is unreachable, so that the user can be asked to try again rather than being told their review was refused.
12. As a frontend developer, I want the request to carry no envelope in its URL or query string, so that event content never lands in an access log.
13. As an event publisher, I want my private key to stay in my browser, so that the gateway can never sign anything on my behalf.
14. As an event publisher, I want my review to reach the network from the browser, so that I can publish without running a node.
15. As an event publisher, I want publishing to be idempotent, so that a retry after a timeout does not create a second event.
16. As an event publisher, I want my review to appear in the index once it is published, so that other people can read it.
17. As a gateway operator, I want to configure the node address, so that each deployment knows which node it writes to.
18. As a gateway operator, I want a fallback list, so that a single node outage does not make the whole network unwritable.
19. As a gateway operator, I want to know that all nodes were unreachable rather than that one failed, so that the 503 I return is honest about what happened.
20. As a gateway operator, I want one long-lived gRPC channel, so that publishing does not pay a TCP and TLS handshake per request.
21. As a gateway operator, I want the channel opened at startup and closed at shutdown, so that there is no connection attempted before the app is ready or after it stops.
22. As a gateway operator, I want a per-call deadline, so that an unresponsive node cannot hold a request open indefinitely.
23. As a gateway operator, I want the gateway to check the size ceiling before spending a node round trip, so that oversized bodies are rejected for free.
24. As a gateway operator, I want the gateway to reject a body that is not JSON at all, so that obvious garbage never reaches the node.
25. As a gateway operator, I want the gateway to log the outcome and the size but never the envelope, so that logs hold no user content.
26. As a gateway operator, I want the gateway to log no source address alongside event content, so that publishing under a fresh identity is not correlated with an address by whoever holds our logs.
27. As a gateway operator, I want oversized submissions rejected at the node's transport layer too, so that the ceiling is enforced independently of the validation library.
28. As a node operator, I want the node to answer with a named outcome, so that a client can distinguish stored from dropped.
29. As a node operator, I want a duplicate to be reported as such rather than as a fresh accept, so that the outcome matches what actually happened.
30. As a node operator, I want the gRPC and Gossipsub doors to call the same submission function, so that validation has one place to land.
31. As a node operator, I want the submission function to be the highest seam available, so that a test exercises the node's behaviour without standing up a mesh.
32. As a node operator, I want both doors to store an event once accepted, so that an event submitted over gRPC and one received by gossip end up in the same place.
33. As a node operator, I want the protobuf definition in the shared artifacts directory, so that the gateway and the node generate from one source.
34. As a node operator, I want the gateway's stubs generated and committed, so that no build step needs a protobuf compiler.
35. As a node operator, I want the node-facing contract separate from the sync contract, so that neither fork forces the other to change.
36. As a node operator, I want the node to accept the connection without authentication, so that the operator's own gateway can publish without a credential to manage.
37. As a future implementer of the validation library, I want a single submission function to call, so that I implement validation once.
38. As a future implementer of the validation library, I want the event identifier the node indexes by to be isolated in one helper, so that I replace it in one place.
39. As a reader of the code, I want the gateway's lifecycle module to not be called "events", so that "event" means the protocol's event throughout the service.
40. As a reader of the decisions log, I want the gateway's publish outcome to use the name the architecture already gives it, so that one outcome has one name in the whole project.
41. As a reader of the decisions log, I want the publish verb's meaning recorded, so that nobody reads it as the gossip publish.

## Implementation Decisions

**The gateway forwards opaque bytes.** The request body is the canonical signed
envelope, read as raw bytes. No model, no schema, no re-serialization. The
gateway is a pipe, and this is the property that makes it a correct one. A
gateway that parses and re-serializes changes the canonical bytes, changes the
digest, and invalidates a signature that was valid when it left the browser.

The one exception is a syntactic JSON validity check, which is rejected. It
parses nothing semantically — no field is read, nothing is interpreted — and it
exists because a body that is not JSON at all should fail at the gateway rather
than consume a node round trip.

**The node-facing contract is its own protobuf module, separate from sync.** Two
different processes, two different compatibility timelines, two different sets of
parties. Merging them is what makes the first incompatible change to one force a
version bump on the other. The existing file is already node-facing only, so the
split costs a rename rather than a design.

**The canonical protobuf moves to the shared artifacts directory.** The gateway
is a different language in a different directory, and generating its stubs from a
file nested inside the node's module is an accident of layout rather than a
decision. The Python stubs are generated and committed, so no build or runtime
step needs a protobuf compiler.

**The node answers with a named outcome, not a boolean.** Four outcomes, one per
way an event can end: accepted, already exists, orphan, rejected. The names are
the protocol's — already-exists is the term the architecture document already
uses, and orphan is the term the protocol uses for a buffered event. Introducing
a synonym for either would give one concept two names.

Rejected carries a reason; the others carry none, because there is nothing to
explain. A publisher does not need to be told why an event was stored.

**The gateway maps outcomes to HTTP status codes.** Accepted and already-exists
are both success — they differ only in the body, and the caller who cares about
the difference reads it. Orphan is accepted-but-pending, which is what a 202
means. Rejected is a 400, so a client that checks only the status does not treat
a refusal as a success. The status alone is enough to route; the body tells you
what happened.

**One event per request.** The gossip layer carries one event per message, so a
batch would be a shape the network immediately dismantles. Worse, one invalid
event in a batch would decide the response for all of them, and the publisher
would not learn which.

**No node answer is an infrastructure error, not a rejected event.** The service
already has an exception hierarchy and a registered handler; a rejected publish
and an unreachable network are different failures and must not share a status.
When the primary node and every fallback have failed, the publisher is being told
the network is not accepting writes right now — which is worth distinguishing
from "your review was refused".

**The size ceiling is enforced at the gRPC transport, and nowhere else yet.** The
protocol ceiling is 16 KB per envelope; the gateway's default receive limit is
several megabytes. Lowering the transport limit rejects oversized bodies before
they are parsed, which is where cheap rejections belong. The gossip door has no
equivalent transport limit and no check of its own, and that gap is real and
recorded rather than treated as an oversight — it closes when the validation
library lands, since size is one of its steps.

**Both doors call one submission function.** Today the gRPC door stores and
gossips; the gossip door logs and returns. Neither shares code with the other.
The submission function is the highest seam the node has: a test drives it
directly against a real store, and it is the single call the validation library
will sit behind. Testing the two doors separately would reproduce exactly the
duplication the seam exists to remove.

**The event identifier is extracted by one isolated helper.** It is the only
parse on the submission path, it exists only to index by, and it is temporary. It
stays until the validation library replaces it, in one place, with a comment
saying so. Anticipating the abstraction would mean inventing a type the library
does not have yet.

**No validation is implemented, deliberately.** The node stores and gossips what
it is given. This is a known, accepted intermediate state and it is the reason
the submission function exists — so that the library lands in one place rather
than being threaded through two doors. The library is written in the node's
language and imported there. Validation does not belong in the gateway: a
gateway that validated before forwarding could be bypassed by anyone running
their own node, which would move integrity control from the node to an
intermediary the protocol treats as optional.

**The gateway is unauthenticated to the node.** The two run inside one operator's
trust boundary. Anti-abuse belongs to node admission policy and to gossip peer
scoring, not to the transport between two processes the same person runs. The
cost is recorded: anything on the gateway's network can make it issue node
requests.

**The gateway keeps a fallback list of nodes.** Publishing goes to the primary;
if it does not answer, to the next. Without a list the gateway is a proxy with
latency rather than a gateway. Publishing stays safe to retry across nodes because
the event identifier is a content hash, which is what makes the fallback list
free of ordering hazards.

**The gRPC channel is opened at startup and closed at shutdown.** One channel per
deployment, not one per request — a channel per request pays a connection setup
on every publish. A deadline applies per call so that an unresponsive node cannot
hold a request open. The lifecycle module is the natural home for both, which is
also why it is the module being renamed.

**The gateway's lifecycle module is renamed.** It is currently called "events",
which in this codebase means a signed envelope and nothing else. The name goes to
the protocol's meaning; the module that talks to the node is named for the node.
This follows the same reasoning that stopped "disputed" from naming both a
moderation label and a vote: one vocabulary, one meaning.

**The gateway logs the outcome and the size, never the envelope and never a
source address.** The design requires gateways not to log source addresses
alongside event content, because publishing reveals an address and correlating it
with an event undoes the value of a fresh identity. Since the gateway does not
parse the envelope, it cannot log an event identifier either — which makes the
compliant behaviour the natural one rather than a discipline to maintain.

**The publish verb keeps its name on the gRPC service.** It is not the gossip
publish, which means "send to the topic", but renaming a contract and its
generated stubs to settle a vocabulary point is a poor trade. The meaning is
recorded in the decision log instead: on the gRPC boundary, publish means submit
for acceptance.

## Testing Decisions

A good test here asserts what a caller observes — the outcome name, the status
code, whether the bytes reached the store unchanged — and never which internal
function produced them. In particular, no test asserts that the gateway invoked
the gRPC client, or that the node called the store: those are the
implementation details the seams exist to let us change.

**The node is tested at one seam, not two.** The submission function, driven
directly against a real store in a temporary directory, is the whole of the node
side. Writing tests against the gRPC service and against the gossip subscription
would test the same behaviour twice and would re-create, in the test suite, the
duplication the seam removes. One test that a submission lands in the store and
is gossiped; one that the same identifier submitted twice reports already-exists;
one that the second submission does not create a second stored entry.

**The gateway is tested at its HTTP route.** The route is the highest seam the
gateway has, and testing below it would test the framework. The node client is
doubled, so the tests assert what a caller of the API sees — the status code, the
outcome name, the body — for each of the four outcomes, plus the unreachable
case. They do not assert that gRPC was called.

**Byte preservation is tested directly, because it is the property everything
else rests on.** A test that submits an envelope through the route and reads the
identical bytes back out of the doubled client is what proves the gateway did not
re-serialize. This is the test that would have caught a model-binding mistake, and
it is worth more than any number of tests that only check status codes.

**Oversized bodies are tested at the gateway.** A body over the ceiling is
rejected without reaching the node, asserted by the doubled client never being
called.

**Fallback is tested as behaviour, not as structure.** With the node client
doubled, the first node refusing and the second succeeding is a test of the
published outcome. Asserting that a loop was iterated is not.

Prior art, both in-repo: the node's existing tests drive a real libp2p host
against temporary state and assert externally visible behaviour, which is the same
discipline at a lower seam. The gateway's existing tests follow a naming
convention and a shared fixture module, which new tests follow.

## Out of Scope

**Validation of any kind.** All thirteen steps: size, network and version,
envelope shape, duplicates, payload schema, canonicalization, digest
recomputation, signature verification, reference resolution, admission policy,
storage, indexer notification, rebroadcast. This is the library's work, and the
library is specified separately. Until it lands the node accepts anything with an
envelope shape it can read, and that is stated plainly rather than softened.

**The read path.** No index, no indexer, no search, no fetching an event by
identifier for display. Reading does not touch a node, so there is nothing on
the write path to build toward it.

**The frontend.** No integration, no client library, no change to the signing
code. The gateway has to be ready for the contract; it does not have to be
consumed by anything in this spec.

**Authentication**, on the gateway's HTTP surface or on the gRPC channel.

**Node-to-node sync**, and its protobuf definition. The gRPC contract is
node-facing only.

**The orphan buffer**, and any buffer at all. The orphan outcome is reported so
the contract has room for it; nothing buffers.

**Admission policy.** Per-author rate limits, per-identity thresholds. Node-local,
and a decision the operator makes.

**Changes to the event store** beyond what the submission path needs.

## Further Notes

The node is not safe to expose to an untrusted network once this lands, and the
spec says so rather than leaving it to be discovered. The transport ceiling and a
syntactic JSON check are not validation — they reject the obvious, not the
forged. Anything with an envelope shape the node can read is stored and gossips
out onto the network. The gap closes when the validation library lands, and it is
the reason the submission function exists.

The gateway's role on the write path is transport and discovery, and it is worth
not overselling it. It does not validate content, it cannot — it holds no key and
verifies no signature. Every integrity guarantee lives at the node. The gateway's
real reasons to exist are that a browser cannot run the peer-to-peer stack, that
a client should not need to know which node to write to, and that reads are
served from an index that does not live in the node at all.

Publishing is not anonymous and the gateway is where that becomes visible, since
it is the component that sees a source address next to a signed event. This is
recorded in the design's privacy section and is the reason the logging decision
is written the way it is.

Two entries belong in the decision log as part of this work: the publish verb's
meaning on the gRPC boundary, and the outcome naming matching the architecture
document's existing term rather than introducing a synonym. The repository layout
change — the protobuf moving to the shared artifacts directory — is also
awaiting a decision-log entry.