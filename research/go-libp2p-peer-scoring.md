# go-libp2p peer scoring, connection manager and resource manager

Research for Linear ECO-176. Question: what does go-libp2p already give the
node against misbehaving peers, at the versions this repo pins, and does any
of it punish a peer for orphans that never resolve? Two decisions depend on
the answer: the per-peer orphan-buffer quota (`docs/protocol.md` →
*Unresolved references*) and node admission policy (step 10).

Versions, from `node/go.mod`:

- `github.com/libp2p/go-libp2p v0.50.0`
- `github.com/libp2p/go-libp2p-pubsub v0.17.0`

Source paths below are relative to the module roots at those exact versions
(read from the local module cache,
`~/go/pkg/mod/github.com/libp2p/<module>@<version>/`). Production-network
sources were read at the HEAD of their default branch on 2026-10-21; the
commit is given next to each.

## TL;DR

1. **Peer scoring is off unless the node opts in.** `pubsub.NewGossipSub(ctx, h)`
   — what `node/internal/p2p/gossip.go` calls today — runs with no score at
   all. It needs `pubsub.WithPeerScore(params, thresholds)`, and there are **no
   library defaults** for any weight or threshold: every production network
   writes its own.
2. **`ValidationReject` feeds P4; `ValidationIgnore` does not.** This is both
   the spec (gossipsub v1.1, *Extended Validators*) and the code
   (`score.go`, `peerScore.RejectMessage`). Ignore leaves no trace in the
   score — not on the sender, not on the peers who forwarded the same message.
3. **Nothing built in penalises orphans that never resolve.** An orphan must
   leave the gossipsub validator as `ValidationIgnore` (it is neither valid
   nor invalid yet, and must not be forwarded) — so P4 never sees it, and once
   the validator has returned there is no API to change the verdict
   retroactively. The claim in `docs/protocol.md` → *Unresolved references*
   that this "is exactly the behavior Gossipsub peer scoring already
   penalizes" is **not true of go-libp2p-pubsub as shipped**.
4. **The hook that does exist is P5, `AppSpecificScore func(peer.ID) float64`.**
   It is evaluated live on every score computation, so a node can keep its own
   per-peer counter of expired orphans and return a negative value. That is a
   built-in *slot*, but the counting is Ecoa's own work.
5. **The peer gater (`WithPeerGater`) does count Ignores**, per peer, as
   lowered "goodput" — but it only acts while the validation queue is being
   throttled, and only as a random early drop, never as a penalty.
6. **Gossipsub scoring only sees gossip.** The sync door (a separate stream
   protocol) is invisible to it. Orphans arriving through sync can be
   attributed to a peer only by Ecoa's own code.
7. **Connection limits are on by default** through `libp2p.New`: a basic
   connection manager at low/high water 160/192 and a resource manager with
   auto-scaled limits, 8 connections per peer, 8 concurrent connections per
   IPv4 /32, and a connection-rate limiter per subnet.

## Gossipsub v1.1 peer scoring (go-libp2p-pubsub v0.17.0)

Enabled with `WithPeerScore(*PeerScoreParams, *PeerScoreThresholds)`
(`gossipsub.go:368`). Both structs are in `score_params.go`. The spec is
`libp2p/specs` `pubsub/gossipsub/gossipsub-v1.1.md` (commit `c750236de4`),
which says reasonable defaults "are not shown here" because they are
application-specific.

### Score formula

`score.go:271` (`peerScore.score`):

```
score = min(TopicScoreCap, Σ_topics TopicWeight · (P1·w1 + P2·w2 + P3·w3 + P3b·w3b + P4·w4))
      + P5 · AppSpecificWeight
      + P6 · IPColocationFactorWeight
      + P7 · BehaviourPenaltyWeight
```

### Per-topic parameters (`TopicScoreParams`, `score_params.go:117`)

| Param | Fields | Value | Weight sign |
|---|---|---|---|
| — | `TopicWeight` | multiplier for the topic | — |
| P1 time in mesh | `TimeInMeshWeight`, `TimeInMeshQuantum`, `TimeInMeshCap` | `meshTime/Quantum`, capped | positive |
| P2 first deliveries | `FirstMessageDeliveriesWeight`, `…Decay`, `…Cap` | decaying counter, capped | positive |
| P3 mesh delivery rate | `MeshMessageDeliveriesWeight`, `…Decay`, `…Cap`, `…Threshold`, `…Window`, `…Activation` | `(threshold − counter)²` when below threshold, after `Activation` | negative |
| P3b mesh failure | `MeshFailurePenaltyWeight`, `…Decay` | sticky deficit when pruned with an active P3 penalty | negative |
| P4 invalid messages | `InvalidMessageDeliveriesWeight`, `…Decay` | `counter²` | negative |

Topic params can be replaced at runtime with `Topic.SetScoreParams`
(`topic.go:52`).

### Global parameters (`PeerScoreParams`, `score_params.go:66`)

- `Topics map[string]*TopicScoreParams`, `TopicScoreCap` (caps the *positive*
  topic contribution; 0 = no cap).
- **P5** `AppSpecificScore func(peer.ID) float64`, `AppSpecificWeight`.
  Mandatory unless `SkipAtomicValidation` (`score_params.go` `validate`).
- **P6** `IPColocationFactorWeight`, `IPColocationFactorThreshold`,
  `IPColocationFactorWhitelist`: `(peersOnSameIP − threshold)²`.
- **P7** `BehaviourPenaltyWeight`, `BehaviourPenaltyThreshold`,
  `BehaviourPenaltyDecay`: `(counter − threshold)²`. The counter is bumped
  only by the router, through the unexported `peerScore.AddPenalty`:
  GRAFT during prune backoff (+1, +1 more inside the flood cutoff,
  `gossipsub.go:1104,1111`), IHAVE promises not followed by the message
  within `IWantFollowupTime` (3 s default, `gossipsub.go:1951`), and an
  extensions-protocol violation (+10, `gossipsub.go:321`). **The application
  cannot add to P7.**
- Decay: `DecayInterval` (default constant 1 s), `DecayToZero` (0.01),
  `RetainScore` (how long counters survive a disconnect), `SeenMsgTTL`.
  `ScoreParameterDecay(d)` turns "decays to zero in `d`" into the per-tick
  factor (`score_params.go:407`).

### Thresholds (`PeerScoreThresholds`, `score_params.go:12`)

Validated in `score_params.go:37`; behaviour from the spec, *Score
Thresholds*, and `gossipsub.go`:

| Threshold | Constraint | Below it |
|---|---|---|
| `0` (implicit) | — | pruned from the mesh at heartbeat, not grafted, no PX either way |
| `GossipThreshold` | ≤ 0 | no IHAVE/IWANT to or from the peer |
| `PublishThreshold` | ≤ `GossipThreshold` | own publications not flood-published to it |
| `GraylistThreshold` | ≤ `PublishThreshold` | **every RPC from it is dropped** (`AcceptFrom`, `gossipsub.go:860`) |
| `AcceptPXThreshold` | ≥ 0 | its PX records are ignored |
| `OpportunisticGraftThreshold` | ≥ 0 | (mesh median) triggers opportunistic grafting |

Gossipsub never closes a connection on score; graylisting is the strongest
built-in action. Disconnecting is left to the application (Prysm and
Lighthouse do it in their own peer managers). Two related levers:
`PubSub.BlacklistPeer(peer.ID)` (`pubsub.go:1998`, drop everything from a
peer, permanently for the process) and `WithPeerScoreInspect` (`score.go:157`,
periodic score snapshots for the app to act on).

### How validation outcomes reach the score

Validators are registered with `RegisterTopicValidator` and return
`ValidationAccept | ValidationReject | ValidationIgnore`. The validator
receives `src peer.ID`, which is the **forwarding** peer
(`Message.ReceivedFrom`), not the event's author.

`validation.go:340–425` maps the result to a trace:

- `ValidationReject` → `RejectValidationFailed`
- `ValidationIgnore` → `RejectValidationIgnored`
- concurrency throttle hit → `RejectValidationThrottled`

`peerScore.RejectMessage` (`score.go:734`) then:

- signature/self-origin failures → P4 on the sender immediately;
- `RejectValidationQueueFull`, `RejectValidationThrottled`,
  `RejectValidationIgnored` → **no penalty**; delivery record set to
  throttled/ignored and its peer list discarded;
- everything else, including `RejectValidationFailed` → delivery marked
  invalid, **P4 on the sender and on every peer that delivered the same
  message during validation**; later duplicates of that message also earn P4
  (`DuplicateMessage`, `deliveryInvalid` case).

This matches the spec's *Extended Validators*: "Ignore message; … the router
does not trigger the P₄ penalty."

A validator-level timeout exists (`WithValidatorTimeout`, `validation.go:584`)
but it only cancels the context passed to the validator; whatever the
validator returns is what counts. An unexpected return value becomes Ignore
(`validation.go:512`).

### The peer gater (`peer_gater.go`)

`WithPeerGater(*PeerGaterParams)` is Random Early Drop in front of the
validation queue. Per peer (and per IP) it keeps decaying counters of
`deliver`, `duplicate`, `ignore` and `reject`; defaults weigh them 1 / 0.125 /
1 / 16 (`DefaultPeerGater*`). It only engages when
`throttled/validated ≥ Threshold` (0.33) and turns off after `Quiet`
(1 min) without throttling; while engaged, a peer is admitted with
probability `(1 + deliver) / (1 + weighted total)` (`AcceptFrom`,
`peer_gater.go:325`). So **Ignores do count against a peer here** — but only
under load, only probabilistically, and never as a score penalty.

### Other relevant knobs

Validation pipeline: `WithValidateQueueSize` (default 32),
`WithValidateThrottle` (global concurrent async validations, 8192),
`WithValidatorConcurrency` (per topic, 1024), `WithValidatorInline`
(`validation.go:15–17, 547–608`). `WithMaxMessageSize` (default 1 MiB,
`pubsub.go:31`). Mesh defaults: D=6, Dlo=5, Dhi=12, Dscore=4, Dout=2,
heartbeat 1 s, prune backoff 1 min (`gossipsub.go:53–67`).

### Connection-manager integration

Gossipsub registers a tag tracer automatically (`gossipsub.go:314`): mesh
peers are `Protect`ed under `pubsub:<topic>`, direct peers under
`pubsub:<direct>`, and first deliverers get a decaying
`pubsub-deliveries:<topic>` tag (+1 per first delivery, cap 15, −1 every
10 min; `tag_tracer.go:14–32`). This is what makes the connection manager
prefer useful gossip peers when it trims — independent of whether scoring is
enabled.

## Connection manager (go-libp2p v0.50.0, `p2p/net/connmgr`)

`connmgr.NewConnManager(low, hi int, opts...)` (`connmgr.go:111`). When
the peer count exceeds `hi`, it trims down to `low`, closing the
lowest-value peers by tag sum and skipping protected ones. Options
(`options.go`): `WithGracePeriod` (default 1 min — new connections are
immune), `WithSilencePeriod` (default 10 s between trims), `DecayerConfig`.
Also `Protect`/`Unprotect`, `TagPeer`/`UpsertTag`, decaying tags,
`TrimOpenConns`, `ForceTrim`.

**Default when the node passes nothing:** `libp2p.New` applies
`DefaultConnectionManager` = `NewConnManager(160, 192)` (`defaults.go:122`).
This is what `node/internal/p2p/host.go` gets today.

The connection manager is passive: it trims, it does not refuse. Refusing is
`core/connmgr.ConnectionGater` (`InterceptPeerDial`, `InterceptAccept`,
`InterceptSecured`, …), which is how a peer would actually be banned at the
transport layer.

## Resource manager (go-libp2p v0.50.0, `p2p/host/resource-manager`)

Hard limits on conns, streams, memory and FDs in nested scopes: system,
transient, service, service-peer, protocol, protocol-peer, peer, conn,
stream. `libp2p.New` installs one by default:
`NewResourceManager(NewFixedLimiter(DefaultLimits.AutoScale()))`
(`defaults.go:110`). `AutoScale` scales `DefaultLimits` against 1/8 of system
memory and 1/2 of the FD limit (`limit_defaults.go:629`).

`DefaultLimits` base values (`limit_defaults.go:663`), before scaling:

- System: 64 inbound / 128 outbound / 128 conns, 128 MiB, 256 FD; +64/+128/+128
  conns and +1 GiB memory per GiB of scaling memory.
- Transient: 32 / 64 / 64 conns, 32 MiB.
- **Per peer: 8 inbound, 8 outbound, 8 conns**, 256/512/512 streams, 64 MiB.
- Per conn: 1, 32 MiB; per stream 16 MiB.

Plus two subnet guards that apply without configuration:

- Concurrent conns per subnet (`conn_limiter.go:33–46`): 8 per IPv4 /32,
  8 per IPv6 /56, 64 per IPv6 /48 (`WithLimitPerSubnet`,
  `WithNetworkPrefixLimit` to change).
- Connection *rate* per subnet (`conn_rate_limiter.go:10–58`): 0.2 conn/s
  burst 16 per IPv4 /32; 0.2/s burst 16 per IPv6 /56; 0.5/s burst 80 per
  /48; localhost unlimited (`WithConnRateLimiters`).

An allowlist (`WithAllowlistedMultiaddrs`) gives trusted endpoints separate
system/transient budgets.

## What production networks set

None of them leaves scoring at library defaults, because there are none.

### Filecoin / Lotus (go-libp2p-pubsub) — `node/modules/lp2p/pubsub.go`, `ce14c9c6ee`

- Thresholds: gossip −500, publish −1000, graylist −2500, acceptPX 1000,
  opportunisticGraft 3.5.
- P4 on every topic: weight −1000, decay `ScoreParameterDecay(1h)`; with
  TopicWeight 0.1, one invalid message is −100 (their comment).
- P3/P3b (mesh delivery) **commented out** — not enabled.
- P5: +2500 for bootstrappers, +1500 for drand bootstrappers, 0 otherwise,
  with the comment "TODO: we want to plug the application specific score to
  the node itself in order to provide feedback to the pubsub system based on
  observed behaviour".
- P6 threshold 5, weight −100. P7 threshold 6, weight −10, decay 1h.
  `RetainScore` 6h. `WithValidateQueueSize(256)`.
- Peer gater on (threshold 0.33, global decay 2 min, source decay 1h — 10 min
  on bootstrappers).
- Connection manager: low 150, high 180, grace 20 s
  (`node/config/def.go:59–61`).
- Validators (`chain/sub/incoming.go`) return Ignore for "can't judge now"
  cases and Reject for provably bad messages.

### Ethereum consensus — Prysm (go-libp2p-pubsub) — `beacon-chain/p2p/gossip_scoring_params.go`, `59d56353d3`

- Thresholds: gossip −4000, publish −8000, graylist −16000, acceptPX 100,
  opportunisticGraft 5.
- `TopicScoreCap` 32.72, P5 always 0, P6 threshold 10 weight −35.11,
  P7 threshold 6 weight −15.92 decaying over 10 epochs, `DecayInterval` one
  slot, `RetainScore` 100 epochs.
- Per-topic weights derived from expected message rates; beacon block P4
  weight −140.4475. Score snapshots read every minute through
  `WithPeerScoreInspect` (`beacon-chain/p2p/pubsub.go:170`).
- **The orphan analogue:** a block whose parent is unknown has its signature
  checked, is put in a pending queue, its parent is requested, and the
  validator returns **`ValidationIgnore`**
  (`beacon-chain/sync/validate_beacon_blocks.go:179–214`). That is exactly
  what the consensus spec mandates: "[IGNORE] The block's parent has been
  seen … (MAY be queued until parent is retrieved)", and "clients SHOULD
  maintain maximum queue sizes to avoid DoS vectors"
  (`ethereum/consensus-specs` `specs/phase0/p2p-interface.md`, `2344668a85`).

### Ethereum consensus — Lighthouse (rust-libp2p, for comparison) — `beacon_node/lighthouse_network/src/service/gossipsub_scoring_parameters.rs`, `014ba06b56`

Same thresholds as Prysm (−4000 / −8000 / −16000 / 100 / 5) — the Ethereum
clients converged on one parameter set. P6 threshold 8.

### IPFS / Kubo — `core/node/libp2p/pubsub.go`, `833e04a240`

- **No peer scoring.** `NewGossipSub` with discovery, flood-publish and a
  seqno validator only.
- Connection manager: low 32, high 96, grace 20 s, silence 10 s
  (`config/init.go:81–96`).
- Resource manager: custom limits sized from `MaxMemory` (default half of
  system RAM), one inbound connection per MiB, and at least twice the
  connection manager's high water (`core/node/libp2p/rcmgr_defaults.go`).

## What this means for the two decisions

**Per-peer orphan-buffer quota.** Required, and Ecoa's own. Nothing in
go-libp2p-pubsub counts unresolved orphans against a peer: they leave the
validator as Ignore, which P4 skips by design; the verdict cannot be revised
after eviction; P7 is not reachable from the application. Ethereum's
answer to the same problem (unknown-parent blocks) is the same shape: Ignore,
queue, bound the queue, and request the missing parent. The sync door is not
covered by gossipsub at all. The one built-in place to make expired orphans
*cost* a peer is P5: keep a per-peer count of orphans evicted unresolved
(and of buffered events that later fail step 9), decay it, and return it
negated from `AppSpecificScore`. That function is called under the score
lock on every score read — including on every incoming RPC through
`AcceptFrom` — so it must be a cheap, non-blocking lookup that never calls
back into pubsub. The peer gater, if enabled, adds a load-only throttle on
peers with many Ignores for free.

`docs/protocol.md` → *Unresolved references* should be corrected: peer
scoring penalises orphans only if the node feeds them in through P5.

**Node admission policy (step 10).** Scoring sees peers (the forwarder),
never authors, so per-author and per-identity limits are not covered by
anything in libp2p. What libp2p does cover, by default and without code:
connection counts (160/192 trimming), 8 conns per peer, 8 concurrent conns
and a connection-rate limit per IPv4 address, memory/stream caps. What it
covers only once configured: P4 for events that fail steps 1–8 (if the
validator returns Reject for them), P6 IP colocation, P7 protocol
misbehaviour, graylisting. Step-10 rejections are local policy, so they
should be Ignore, not Reject: rejecting would make an honest relaying peer
pay P4 for a limit only this operator applies.

## Implementation notes picked up on the way

- `node/internal/p2p/gossip.go` registers no topic validator today; the
  single validation path will have to be the gossipsub validator for the
  gossip door, mapping step outcomes to Accept / Reject / Ignore.
- A resolved orphan must be re-published by the node itself. With the
  default message ID (`from` + `seqno`) that re-publication would be a new
  gossipsub message; `WithMessageIdFn` keyed on `eventId` makes the network's
  seen-cache match Ecoa's dedup.
