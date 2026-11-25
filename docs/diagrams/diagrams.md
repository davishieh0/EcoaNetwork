# How It Fits Together

Diagrams for the parts that are hardest to hold in your head from prose. Each one points at the document that actually specifies the behavior — where a diagram and a spec disagree, the spec is right.

These are deliberately not one-to-one with the documents. A diagram earns its place by showing a *mechanism* that a list can't, so the flows with branches and the rules with asymmetries are here, and things that are already a clean list are not.

Sections 1 to 10 draw the design. Sections 11 to 17 draw the system as it is built in this repository — the services, their data, and the flows the MVP added — and say where v1 builds less than the design. The architecture picture itself is [architecture.drawio](./architecture.drawio).

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
    Indexer -->|"W4. SubscribeEvents from its cursor"| Node
    Indexer -->|"W5. project"| Index

    Client -->|"R1. query"| GW
    GW -->|"R2. read API"| Indexer
    Indexer -->|"R3. query"| Index
```

What to notice: the indexer **asks for** the node's log from a cursor it keeps itself, rather than the node knowing about it, so it can be restarted, rebuilt, or run twice with different configuration. The stream it opens (`SubscribeEvents`) sends the history after the cursor, then each new event as it is stored: the node keeps nothing about who reads it. And the gateway serves data only — the client is a static bundle hosted independently, because whoever ships the code can ship code that steals the key. See [architecture.md](../architecture.md#substitutability-only-counts-if-the-client-isnt-served-by-the-gateway).

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
    G->>G: size and shape
    G->>N: forward, unchanged (the next node if this one is down)
    N->>N: validation checklist
    N->>N: store, assign local sequence
    N-->>G: accepted, eventId
    G-->>C: accepted
    N->>P: Gossipsub broadcast
    P->>P: the same checklist, independently
    N-->>I: the open SubscribeEvents stream sends it
    P-->>I: and so do theirs, to the same indexer
    I->>I: store once, project, note which nodes hold it
```

What to notice: the gateway never signs and never modifies. It checks shape and forwards bytes, trying its nodes in order. Rate limiting at the gateway is designed ([identity.md](../identity.md#where-these-controls-actually-sit)) and not built in v1; each node's admission policy caps events per author per hour. Nodes B and C re-run the entire validation themselves — nothing is trusted for having arrived from a peer, including from the node that first accepted it.

---

## 3. What a node does when an event arrives

The numbered list in [protocol.md](../protocol.md#validation-checklist-every-node-runs-this-on-receipt) reads like a straight line. It isn't — the branches are where the design lives.

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
    M --> N["12. wake the open<br/>SubscribeEvents streams"]
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

Two calls, and deliberately no more. The cursor is a position in **B's local receive sequence**, never a timestamp — an event stamped three years ago and published today would fall behind a timestamp cursor and never reach anyone syncing incrementally. That also means cursors aren't portable: A tracks one per peer, and starts from zero with a peer it has never met. See [protocol.md](../protocol.md#node-to-node-sync).

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

Votes stop at the version they were cast on, so an author can't collect endorsements and then swap the text underneath them. Labels and responses carry forward, so a spam label can't be cleared by publishing a one-character edit. Anything an author could *gain* by editing must not survive the edit; anything they could *escape* by editing must. See [protocol.md](../protocol.md#votes-do-not-follow-an-edit).

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

A node back-filling history sees both rotations at once and has no "first observed". It then **fails closed** — the identity is contested, neither successor is canonical, and history under the original key stays readable. A stolen key can therefore still freeze an identity, but never transfer it. See [identity.md](../identity.md#key-rotation).

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
    REC --> A1["vote weight<br/>human 0.3, email 0.6, domain and cnpj 1.0<br/>(designed; v1 weighs nothing yet)"]
    REC --> A2{"does the claim name the<br/>exact entity being acted on"}
    A2 -->|yes| A3["tier 1 - edits canonical immediately<br/>official response badge"]
    A2 -->|no| A4["tier 2 - a weighted proposal like anyone else's"]
```

A `domain` claim never verifies once and stays true: it's **live state**, re-checked on a TTL, and a lapsed DNS record downgrades it without any revocation event existing. **v1 builds** `domain` (the indexer's own lookup) and `human` (issued by the gateway, counted when the indexer follows it); weights, `email` and `cnpj` wait. Section 15 draws both as built. A claim also grants authority over the identifier it names and nothing else — a `domain` claim doesn't reach a `cnpj:` entity, because nothing in a DNS record says which CNPJ is behind it.

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

Note the consequence, spelled out in [entities.md](../entities.md#2-unverified-edit--a-proposal-not-a-fact): the *rule* is deterministic but the *outcome* is not, because weight is a local observation. Two honest indexers can land on different canonical values.

**v1 builds** tier 1 (a verified `domain` representative's latest value holds) and the incumbency rule with every weight at zero, so the first proposal of each field holds until weights exist; nobody votes on proposals yet. The rule is `projection.ResolveFields` in `protocol/go`, and the indexer already hands it the claims it recognizes.

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

A disagreement between two indexers on the left is a bug in one of them. A disagreement on the right is the design working. Moving anything from right to left would require electing an authority over clocks, DNS, or ranking, which is precisely what this architecture exists to avoid. See [architecture.md](../architecture.md#what-must-be-identical-across-indexers-and-what-must-not-be).

---

## 11. The services, as they run

What `infra/local` starts, and who talks to whom. Every arrow is the only way that pair talks.

```mermaid
flowchart LR
    B(["Browser<br/>React client<br/>keys in IndexedDB"])
    subgraph site["frontend · nginx :8080"]
        S["static bundle"]
        P["/api proxy<br/>BACKEND_ORIGIN"]
    end
    BE["backend · FastAPI :8000<br/>accounts (Cognito)<br/>publish · reads · human verifier"]
    subgraph nodes["three nodes · Go"]
        N1["node-1<br/>gRPC :50051 · libp2p :4001"]
        N2["node-2"]
        N3["node-3"]
    end
    subgraph idx["indexer · Go"]
        PR["indexer project<br/>follows every node<br/>checks domain claims"]
        SV["indexer serve<br/>read API :8081"]
    end
    PG[("Postgres<br/>ecoa_backend · ecoa_indexer")]
    DNS["dns · CoreDNS<br/>the .test zone"]
    COG[["AWS Cognito"]]

    B --> S
    B --> P --> BE
    BE -->|"Publish, in order, failover"| N1
    BE -.->|"if node-1 is down"| N2
    BE -->|"HTTP reads"| SV
    BE --> COG
    BE -->|"users"| PG
    N1 <-->|"Gossipsub + sync"| N2
    N2 <-->|"Gossipsub + sync"| N3
    N1 <-->|"Gossipsub + sync"| N3
    PR -->|"SubscribeEvents"| N1
    PR -->|"SubscribeEvents"| N2
    PR -->|"SubscribeEvents"| N3
    PR -->|"projections"| PG
    PR -->|"TXT lookups"| DNS
    SV -->|"queries"| PG
```

`project` and `serve` are one binary run twice: the writer follows the logs, the reader answers.

---

## 12. Reading a review, end to end

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant G as Gateway
    participant I as Indexer API
    participant D as Postgres

    C->>G: GET /api/reviews/{id}?moderators=...
    G->>I: GET /reviews/{id}, only the parameters the route takes
    I->>D: the version, its chain, its votes, the labels and claims that count
    D-->>I: rows, the envelope as stored bytes
    I-->>G: {review, versions}: envelope as a string, derived, observed
    G-->>C: the same status and the same bytes, never parsed
    C->>C: read the envelope's fields to render it
    C->>C: inspect: recompute the eventId, verify the signature (WebCrypto)
```

The envelope is a string end to end: no JSON parser between the index and the client rewrites a number, an escape or the order of keys, so the client checks the signature over the bytes the author signed. `derived` is what every honest indexer agrees on; `observed` is this indexer's own view (receivedAt, the nodes holding it, the labels and claims that count).

---

## 13. The indexer's data model

Eighteen tables, in two halves ([architecture.md](../architecture.md#what-must-be-identical-across-indexers-and-what-must-not-be)). The **deterministic** ones are a projection of the logs: a rebuild reproduces them exactly, and two indexers agree on them. The **observed** ones are this indexer's own: re-derived, never replayed. Every id is an `eventId` or a `did:key`, as text.

```mermaid
erDiagram
    events ||--o| event_observations : "earliest receipt"
    events ||--o{ event_holders : "held by nodes"
    events ||--o| reviews : "a version"
    reviews ||--o| review_chains : "root of"
    review_chains }o--|| reviews : "head"
    reviews ||--o{ reviews : "replaces"
    review_chains ||--o{ review_withdrawals : "withdrawn by"
    reviews ||--o{ votes : "target"
    reviews ||--o| version_votes : "tally"
    review_chains ||--o{ responses : "conversation"
    reviews ||--o{ responses : "responded version"
    review_chains ||--o{ labels : "labeled chain"
    responses ||--o{ labels : "labeled response"
    entity_proposals ||--o{ labels : "labeled proposal"
    entities ||--o{ entity_proposals : "proposals"
    events ||--o| profiles : "current profile"
    events ||--o| key_rotations : ""
    events ||--o| key_revocations : ""
    events ||--o| claims : ""
    claims ||--o| claim_checks : "DNS lookup"

    events {
        text event_id PK
        text event_type
        text author
        timestamptz created_at
        bytea envelope "as signed, never jsonb"
    }
    reviews {
        text event_id PK
        text chain_root
        text replaces FK
        text author
        text entity "always the root's"
        smallint rating "1 to 5"
        text title
        text body
        text lang
        timestamptz created_at
    }
    review_chains {
        text chain_root PK
        text entity
        text head FK "null when truncated"
        boolean withdrawn
        boolean truncated
        timestamptz created_at "the original's: the listing order"
    }
    review_withdrawals {
        text event_id PK
        text chain_root FK
        text target FK
    }
    votes {
        text event_id PK
        text target FK "the exact version"
        text voter
        text kind "endorsed or disputed"
        timestamptz created_at
    }
    version_votes {
        text version PK
        int endorsements "latest vote per identity"
        int disputes
    }
    responses {
        text event_id PK
        text chain_root FK
        text target "a version or a response"
        text responded_version FK
        int depth
        text author
        text body
        timestamptz sort_at
    }
    labels {
        text event_id PK
        text target
        text chain_root FK
        text response FK
        text proposal FK
        text profile FK
        text moderator
        text label
        text note
        boolean revoked
    }
    entities {
        text entity PK
        timestamptz first_seen
    }
    entity_proposals {
        text event_id PK
        text entity FK
        text author
        text name
        text category
        text description
        jsonb descriptor
        text search_name "folded, for search"
    }
    profiles {
        text identity PK
        text event_id FK
        text display_name
        text bio
    }
    key_rotations {
        text event_id PK
        text key
        text successor
    }
    key_revocations {
        text event_id PK
        text key
        text reason
    }
    identities {
        text key PK
        text identity
        int position
        boolean contested
    }
    claims {
        text event_id PK
        text claim_type
        text issuer
        text subject
        text entity
        text evidence_ref
        timestamptz expires_at
        boolean revoked
    }
    event_observations {
        text event_id PK
        timestamptz received_at "earliest any node reported"
    }
    event_holders {
        text event_id PK
        text node PK
        bigint sequence
        timestamptz received_at
    }
    claim_checks {
        text event_id PK
        boolean verified
        boolean dnssec
        timestamptz checked_at
        timestamptz verified_since
        text detail
    }
    projector_cursors {
        text node PK
        text log_id
        bigint sequence
    }
```

| Half | Tables |
|---|---|
| Deterministic | `events`, `reviews`, `review_chains`, `review_withdrawals`, `votes`, `version_votes`, `responses`, `labels`, `entities`, `entity_proposals`, `profiles`, `key_rotations`, `key_revocations`, `identities`, `claims` |
| Observed | `event_observations`, `event_holders`, `claim_checks` |
| Bookkeeping | `projector_cursors`: how far each node's log was read, saved in the same transaction as what the event changed |

What is **not** a table, on purpose: which labels count (the moderators a request follows, decided per query), which claims count (the indexer's lookups and verifier list, decided per query), the canonical entity metadata (`ResolveFields` at answer time), the summary of ratings and the ranking. None of them is replayed, so none is stored as if it were. The migrations are `indexer/migrations/`.

---

## 14. The gateway's and the node's data

The gateway keeps accounts and nothing that was signed: a review never touches its database. Cognito holds the credentials; `users` ties the Cognito subject to the public handle.

```mermaid
erDiagram
    users {
        uuid id PK "uuidv7"
        text sub UK "Cognito's subject"
        text handle UK "3 to 30, lowercase"
        timestamptz created_at
    }
```

A node keeps its log in one bbolt file, in seven buckets:

| Bucket | Key → value | For |
|---|---|---|
| `events` | eventId → the raw envelope | every stored event, as signed |
| `log` | local sequence → eventId | reading the log in the order it was received (`SubscribeEvents`, sync) |
| `meta` | eventId → sequence and local receive time | the receive time a stream reports |
| `tombstones` | eventId → pruning time and author | the shape only: no v1 node prunes |
| `cursors` | peer ID → the last sync cursor from it | catching up with each peer |
| `peers` | peer ID → its multiaddrs | redialing known peers after a restart |
| `info` | `logId` → a random id made with the store | telling a new log from an old one, so a cursor never counts in the wrong one |

The browser keeps its identities in IndexedDB (`ecoa` / `identity`): the main key pair under `browser`, what it knows of it under `browser:meta` (backed up or not), and each fresh identity under `fresh:<did:key>`. The private keys are non-extractable.

---

## 15. Claims, as v1 issues and recognizes them

```mermaid
sequenceDiagram
    autonumber
    participant B as Browser
    participant G as Gateway
    participant N as Node
    participant P as Indexer project
    participant DNS as DNS
    participant S as Indexer serve

    Note over B,G: human: a proof of work, no third party
    B->>G: GET /api/claims/human/challenge
    G-->>B: an ALTCHA challenge, signed by the gateway
    B->>B: solve it
    B->>G: POST /api/claims/human {subject, solved}
    G->>G: verify, once per challenge
    G->>G: sign claim_created {human, subject} with the verifier key
    G->>N: publish
    Note over S: counts when the issuer is in VERIFIERS_FILE

    Note over B,DNS: domain: self-verifiable, no verifier
    B->>B: show the TXT record to publish
    B->>N: claim_created {domain, subject = author, entity}, via the gateway
    N-->>P: stream
    P->>P: store the claim
    loop every 10 s: claims never checked, failed 30 s ago, passed 30 days ago
        P->>DNS: TXT _ecoa-verify.domain
        DNS-->>P: ecoa-identity=did:key:...
        P->>P: claim_checks: verified, since when
    end
    S->>S: the sole verified holder represents the entity:<br/>tier 1, official responses, out of the summary
```

---

## 16. The identity in the browser

```mermaid
flowchart TD
    A{"this browser has<br/>an identity"} -->|yes| SIGN["sign: the key never leaves"]
    A -->|no| C{"create or restore"}
    C -->|"create, with a password"| G1["generate Ed25519, exportable<br/>for an instant"]
    G1 --> SEAL["seal the PKCS#8 copy:<br/>PBKDF2-SHA-256 600k + AES-GCM,<br/>bound to the did:key"]
    SEAL --> FILE[/"backup file, downloaded"/]
    G1 --> KEEP["re-import non-extractable,<br/>keep in IndexedDB"]
    C -->|"create, no backup"| G2["generate non-extractable<br/>lost with this browser's data"]
    G2 --> SIGN
    KEEP --> SIGN
    C -->|"restore"| R["file + password<br/>decrypt, sign a probe,<br/>check it with the did:key"]
    R --> KEEP
    SIGN --> F{"publish under a<br/>fresh identity"}
    F -->|no| MAIN["signed by the main identity"]
    F -->|yes| FR["a new key for this review alone<br/>kept to edit it, never backed up"]
```

An edit or a withdrawal is signed by the key that wrote the review, whichever of the browser's identities it was: the authorship rule ([protocol.md](../protocol.md#who-may-sign-what)) would refuse any other.

---

## 17. Moderation, per reader

```mermaid
flowchart LR
    M["moderator M<br/>label_created spam"] --> LOG[("the log:<br/>the label and the review,<br/>both kept")]
    LOG --> IDX["indexer: every label stored"]
    R1["reader following M<br/>?moderators=M"] --> IDX
    R2["reader following nobody<br/>?moderators="] --> IDX
    R3["reader with no choice<br/>the index's MODERATORS_FILE"] --> IDX
    IDX -->|"the review is left out"| V1["R1's list"]
    IDX -->|"the review shows"| V2["R2's list"]
    IDX -->|"as the index follows"| V3["R3's list"]
```

A label is a projection, not a deletion: the event stays on every node, and which labels count is the reader's choice, applied in the query. `illegal` and `personal-data` remove (451 by id), `spam`, `harassment` and `impersonation` hide unless the reader asks to see them, `off-topic` and `other` mark.
