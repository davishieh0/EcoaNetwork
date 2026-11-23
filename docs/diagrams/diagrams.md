# How It Fits Together

Diagrams for the parts that are hardest to hold in your head from prose. Each one points at the document that actually specifies the behavior — where a diagram and a spec disagree, the spec is right.

These are deliberately not one-to-one with the documents. A diagram earns its place by showing a *mechanism* that a list can't, so the flows with branches and the rules with asymmetries are here, and things that are already a clean list are not.

---

## 1. The shape of the system

Two paths that never cross. Writes go through a node and out to the network; reads never touch a node at all.

```mermaid
flowchart TB
    User(["User"])
    Client["Client<br/>signs locally, holds the key"]
    GW["Gateway API<br/>HTTP, data only, never serves app code"]
    Node["Node<br/>validates, stores, gossips"]
    Indexer["Indexer<br/>projects events into a query layer"]
    Index[("Index")]
    Net{{"P2P network<br/>other nodes, other operators"}}

    User --> Client
    Client -->|"W1. signed envelope"| GW
    GW -->|"W2. publish"| Node
    Node <-->|"W3. Gossipsub + sync"| Net
    Indexer -->|"W4. GetEventsSince cursor"| Node
    Indexer -->|"W5. project"| Index

    Client -->|"R1. query"| GW
    GW -->|"R2. read"| Index
```

What to notice: the indexer **pulls** from the node rather than being pushed to, so it can be restarted, rebuilt, or run twice with different configuration. And the gateway serves data only — the client is a static bundle hosted independently, because whoever ships the code can ship code that steals the key. See [architecture.md](./architecture.md#substitutability-only-counts-if-the-client-isnt-served-by-the-gateway).

---

## 2. Publishing one review, end to end

```mermaid
sequenceDiagram
    autonumber
    participant U as User
    participant C as Client
    participant G as Gateway
    participant N as Node A
    participant P as Nodes B and C
    participant I as Indexer

    U->>C: writes a review, picks a rating 1 to 5
    C->>C: build envelope, JCS canonicalize
    C->>C: sign with the local key
    Note over C: the private key never leaves this box
    C->>G: POST the signed envelope
    G->>G: size, shape, rate limit
    G->>N: forward, unchanged
    N->>N: validation checklist
    N->>N: store, assign local sequence
    N-->>G: accepted, eventId
    G-->>C: accepted
    N->>P: Gossipsub broadcast
    P->>P: the same checklist, independently
    I->>N: GetEventsSince cursor
    N-->>I: everything after that position
    I->>I: project
```

What to notice: the gateway never signs and never modifies. It checks shape and forwards bytes. Nodes B and C re-run the entire validation themselves — nothing is trusted for having arrived from a peer, including from the node that first accepted it.

---

## 3. What a node does when an event arrives

The numbered list in [protocol.md](./protocol.md#validation-checklist-every-node-runs-this-on-receipt) reads like a straight line. It isn't — the branches are where the design lives.

```mermaid
flowchart TD
    A["event arrives"] --> B{"1. within the size limit"}
    B -->|no| X1["reject"]
    B -->|yes| C{"2. protocol and version supported"}
    C -->|no| X2["ignore, do not relay,<br/>upgrade on your own schedule"]
    C -->|yes| D{"3. envelope shape valid<br/>no unknown top-level field<br/>createdAt not future-dated"}
    D -->|no| X1
    D -->|yes| E{"4. eventId already stored"}
    E -->|yes| X3["drop silently, stop"]
    E -->|no| F{"eventType recognized"}
    F -->|yes| G["5. payload schema<br/>identifier normalization"]
    F -->|no| H["skip the type-specific steps"]
    G --> I{"6 and 7. digest matches eventId"}
    H --> I
    I -->|no| X1
    I -->|yes| J{"8. signature valid, and the<br/>signing key not already revoked here"}
    J -->|no| X1
    J -->|yes| K{"eventType recognized"}
    K -->|no| M
    K -->|yes| L{"9. references resolve<br/>and authorship holds"}
    L -->|"target missing"| O["orphan buffer<br/>not stored, not indexed, not relayed"]
    L -->|"wrong author"| X1
    L -->|ok| M["10 and 11. admission policy,<br/>store, assign local sequence"]
    M --> N["12. notify the indexer"]
    N --> P["13. rebroadcast"]
    O -.->|"the target arrives later"| L
```

Three branches worth reading twice:

- **Unknown `eventType` keeps flowing.** It skips the type-specific steps and is still stored and relayed, so one un-upgraded node can't stall a new event type across the network.
- **A missing reference is not a rejection.** It's a delay. Rejecting would mean no rebroadcast, which would drop valid events out of the network for the crime of arriving a few hundred milliseconds early.
- **Duplicate detection sits before the expensive work**, so replaying an event a node already has costs a hash lookup rather than a signature verification.

---

## 4. Out-of-order arrival

Gossip doesn't deliver causally. A vote arriving before the review it targets is routine, not an attack.

```mermaid
sequenceDiagram
    autonumber
    participant P as Peer
    participant N as Node
    participant B as Orphan buffer

    P->>N: review.endorsed, targets evt_123
    N->>N: signature valid, but evt_123 is unknown here
    N->>B: hold it
    Note over N,B: not stored, not indexed, not relayed
    P->>N: review.created, evt_123
    N->>N: validate and store
    N->>B: anything waiting on evt_123
    B-->>N: yes, the endorsement
    N->>N: re-run from step 9, store
    N->>P: rebroadcast both
```

The buffer is bounded in size and age, and bounded **per peer** — otherwise flooding it with references that will never resolve would be a cheap way to evict everyone else's legitimate orphans. Eviction is safe: a genuine orphan comes back through ordinary sync, because whoever has the target has the child too.

---

## 5. Catching up after downtime

```mermaid
sequenceDiagram
    autonumber
    participant A as Node A, was offline
    participant B as Peer B

    Note over A: back online. Last cursor with B was 1840
    A->>B: GetEventsSince 1840
    B-->>A: events 1841 through 2310
    A->>A: the full checklist on each one
    Note over A: cursor with B is now 2310
    A->>B: GetEvents for evt_x and evt_y
    Note over A,B: filling targets its orphans are waiting on
    B-->>A: per-id status: found, pruned, or unknown
```

Two calls, and deliberately no more. The cursor is a position in **B's local receive sequence**, never a timestamp — an event stamped three years ago and published today would fall behind a timestamp cursor and never reach anyone syncing incrementally. That also means cursors aren't portable: A tracks one per peer, and starts from zero with a peer it has never met. See [protocol.md](./protocol.md#node-to-node-sync).

---

## 6. A review over time — and what attaches to what

The subtlest rule in the project, and the one most worth getting right in an implementation.

```mermaid
flowchart LR
    V1["v1 - evt_a<br/>review.created"]
    V2["v2 - evt_b<br/>review.replaced"]
    V3["v3 - evt_c<br/>review.replaced<br/>current head"]
    V1 --> V2 --> V3

    E1["review.endorsed<br/>target evt_a"]
    E2["review.endorsed<br/>target evt_c"]
    LB["moderation.label_created<br/>target evt_a"]
    RS["review.response_created<br/>target evt_a"]

    E1 -.->|"counts for v1 only,<br/>never for the head"| V1
    E2 -.->|"counts for the head"| V3
    LB ==>|"follows the chain"| V3
    RS ==>|"follows the chain"| V3
```

**Credit binds to the version that earned it; warnings and conversation bind to the thing.**

Votes stop at the version they were cast on, so an author can't collect endorsements and then swap the text underneath them. Labels and responses carry forward, so a spam label can't be cleared by publishing a one-character edit. Anything an author could *gain* by editing must not survive the edit; anything they could *escape* by editing must. See [protocol.md](./protocol.md#votes-do-not-follow-an-edit).

`replaces` names the immediately preceding version, not the original. Two edits naming the same predecessor is a fork, resolved by higher `createdAt` with ties on lower `eventId` — every link is by the same author, so that ordering is self-scoped and every implementation resolves the same head.

---

## 7. Identity: one key, or a cold root with hot devices

```mermaid
flowchart TD
    Root["did:key:root<br/>the identity, cold key"]
    D1["device key - laptop"]
    D2["device key - phone"]
    Ev1["review.created<br/>author = root<br/>deviceKey = laptop"]
    Ev2["review.endorsed<br/>author = root<br/>deviceKey = phone"]

    Root -->|"identity.device_authorized"| D1
    Root -->|"identity.device_authorized"| D2
    D1 -->|"signs"| Ev1
    D2 -->|"signs"| Ev2
    Root -.->|"identity.device_revoked<br/>stolen phone, history survives"| D2
```

`author` is always the identity; `deviceKey` names whichever key produced the signature, and is `null` for a single-key identity. Every "same author" comparison in the protocol is against `author`, so swapping devices never fragments a history.

The `deviceKey` field ships in v1 even though v1 may never set it. The envelope is a closed set, so adding a field later is a breaking version bump the whole network must adopt in lockstep — a `null` per event now buys the option without a migration.

### Rotation: the first one observed is the one that holds

```mermaid
flowchart LR
    K1["did:key:old"]
    K2["did:key:new<br/>observed first - canonical"]
    K3["did:key:mallory<br/>arrives later - rejected<br/>whatever its eventId"]
    H[("everything published<br/>under did:key:old")]

    K1 -->|"identity.key_rotated<br/>signed by the old key"| K2
    K1 -.->|"a later, conflicting rotation<br/>from a leaked old key"| K3
    K2 -->|"can still edit and withdraw"| H
```

An identity has at most one successor, ever, and a node that has already accepted a rotation **rejects any later conflicting one outright** — no `eventId` comparison, no reconsideration.

An earlier version of this resolved conflicts by taking the lower `eventId` globally, which turned out to be a retroactive identity hijack: someone recovering a leaked old key months later could grind an `eventId` below the accepted rotation's, in about two attempts on average, and take the identity along with everything published since. A rule with no clock in it also had no history in it.

A node back-filling history sees both rotations at once and has no "first observed". It then **fails closed** — the identity is contested, neither successor is canonical, and history under the original key stays readable. A stolen key can therefore still freeze an identity, but never transfer it. See [identity.md](./identity.md#key-rotation).

---

## 8. From a claim to actual authority

Publishing a claim grants nothing. **Recognition** is where authority lives, and recognition is the consumer's decision.

```mermaid
flowchart TD
    C["verification.claim_created"] --> T{"claim type"}
    T -->|"domain, self-verifiable"| S{"does the DNS TXT record<br/>still name this identity"}
    T -->|"human, email, cnpj - delegated"| V{"is the issuer on this<br/>indexer's verifier list"}
    S -->|no| W0["no weight, no authority<br/>downgraded, not revoked"]
    V -->|no| W0
    S -->|yes| REC["recognized"]
    V -->|yes| REC
    REC --> A1["vote weight<br/>human 0.3, email 0.6, domain and cnpj 1.0"]
    REC --> A2{"does the claim name the<br/>exact entity being acted on"}
    A2 -->|yes| A3["tier 1 - edits canonical immediately<br/>official response badge"]
    A2 -->|no| A4["tier 2 - a weighted proposal like anyone else's"]
```

A `domain` claim never verifies once and stays true: it's **live state**, re-checked on a TTL, and a lapsed DNS record downgrades it without any revocation event existing. A claim also grants authority over the identifier it names and nothing else — a `domain` claim doesn't reach a `cnpj:` entity, because nothing in a DNS record says which CNPJ is behind it.

---

## 9. Who wins an entity metadata edit

```mermaid
flowchart TD
    E["entity.updated arrives"] --> Q{"does the author hold a recognized claim<br/>for this exact identifier"}
    Q -->|yes| T1["tier 1 - canonical immediately<br/>every previous value retained<br/>still subject to moderation labels"]
    Q -->|no| T2["tier 2 - a proposal, not a fact"]
    T2 --> F["resolve each field independently"]
    F --> V{"any votes on this proposal"}
    V -->|yes| W1["rank by vote weight"]
    V -->|no| W2["rank by the author's own identity weight"]
    W1 --> HY{"does it beat the incumbent by 20 percent"}
    W2 --> HY
    HY -->|yes| CAN["becomes canonical for that field"]
    HY -->|no| KEEP["the incumbent holds"]
```

The fallback to author weight is the load-bearing part: nobody votes on a company's category, so ranking purely by votes means ranking zero against zero, and whatever tiebreak the implementation accidentally has becomes the real rule — almost always last-write-wins, which hands every entity to whoever edited most recently.

Note the consequence, spelled out in [entities.md](./entities.md#2-unverified-edit--a-proposal-not-a-fact): the *rule* is deterministic but the *outcome* is not, because weight is a local observation. Two honest indexers can land on different canonical values.

---

## 10. What is replayed, and what is re-derived

The project's central claim is that the index is a projection rebuildable from the log. That's true of one half of the index and not the other, and being precise about which is the difference between a testable property and a slogan.

```mermaid
flowchart TB
    LOG[("the signed event log<br/>the only thing that is shared truth")]
    LOG --> DET["deterministic projection<br/>every honest indexer agrees<br/>a rebuild reproduces it exactly"]
    LOG --> OBS["local observation<br/>re-derived, not replayed<br/>honest indexers may differ"]
    DET --> VIEW["what a reader actually sees"]
    OBS --> VIEW

    DET -.- DL["which events exist, content, authorship<br/>structural validity<br/>chain heads and withdrawal state<br/>which vote binds to which version<br/>which entity proposals exist<br/>all within-author ordering"]
    OBS -.- OL["receivedAt and identity age<br/>whether a domain claim still verifies<br/>reputation and vote weight<br/>which entity proposal is canonical<br/>the revocation cut<br/>ranking, and an entity's rating"]
```

The test for which side something belongs on isn't "is the rule shared" — the rules are written down and identical either way. It's **"can it be recomputed from the events and nothing else."**

A disagreement between two indexers on the left is a bug in one of them. A disagreement on the right is the design working. Moving anything from right to left would require electing an authority over clocks, DNS, or ranking, which is precisely what this architecture exists to avoid. See [architecture.md](./architecture.md#what-must-be-identical-across-indexers-and-what-must-not-be).
