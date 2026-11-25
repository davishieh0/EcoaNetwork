# Concepts

This document explains the *ideas* behind Ecoa Network in plain terms. For how things are actually built, see [architecture.md](./architecture.md), [protocol.md](./protocol.md), and [identity.md](./identity.md).

## What "decentralized" means here

Not a blockchain. Not "every phone is a node." It means: **no single server is the definitive source of truth**, and every meaningful piece — client, gateway, storage, moderation — can be run by a different, independent party while still speaking the same language and reading the same data. Decentralization is about *who can run what*, not about how many machines are involved.

## The protocol

A protocol is a set of rules, not a program — the same way HTTP isn't software, it's an agreement that lets unrelated browsers and servers understand each other. The Ecoa Network protocol defines what a review looks like as data, how it's proven authentic, and how nodes exchange it. Anyone could build a new client, node, or indexer from the spec alone, with no shared code, and it would still work with everything else.

## Events: the atomic unit of truth

The system never "updates a row." It only ever produces new, signed, timestamped facts: *"identity X published review Y about entity Z, at time T."* A correction isn't an edit — it's a new fact that says *"this replaces Y."* This is the whole reason nothing can be silently rewritten: the original fact still exists, byte-for-byte, even after the app stops showing it by default.

## Identity vs. account

**Identity** is a cryptographic keypair the user controls — it's what actually authors and proves ownership of events. **Account** is an optional convenience (login, saved preferences, notifications) that any given website can layer on top. Lose the account, and you can still publish from another client with the same identity. Lose the identity's private key without a backup, and the account can't save you.

The official web client asks for an account to write — it lets the gateway refuse abuse — and keeps the identity apart from it: the key is made in the browser, sealed into a backup file only its owner holds, and never sent anywhere, the account's server included.

## Node

A node is a program that stores events, checks their signatures, and talks to other nodes. It is *not* a server in the traditional sense — no node has special authority, and nothing requires a specific one to be up. Nodes can be run at different scales: **personal** (light, just for one user), **community** (always-on, helps others catch up), **full** (tries to mirror everything, backs the network up). In v1 no node prunes anything, so the distinction is about availability and reach rather than about how much history each one keeps — the tiers matter later, when retention becomes a choice someone actually has to make.

## Network vs. protocol

Two layers people tend to conflate. The **network** (libp2p) is plumbing: it moves bytes from node A to node B securely and reliably. The **protocol** is meaning: it defines what those bytes represent and what a node should do with them. The plumbing could theoretically be swapped for something else entirely without changing what an "event" or a "review" *is*.

## Indexer

A raw stream of signed events is a terrible thing to search directly. The indexer is a translator: it reads that stream and builds normal, fast, queryable tables. The important part is that it's **disposable** — anyone can run their own indexer with their own rules for what to include, and rebuild it at any time from the same underlying events. No indexer is "the real one."

## Client

Any program a person uses to reach the network: the official website today, a future mobile app, a future desktop app, even a script. All of them are equally legitimate — none is privileged over another by the protocol. The web frontend is just the first client that happens to exist.

## Moderation without censorship

Traditional moderation deletes or hides content at the source, so nobody can even check what the "real" record was. Here, moderation is a *second, separate stream of events* layered on top of the first — labels like "possible spam" or "off-policy" — that any client or indexer can choose to respect or ignore. The underlying content isn't erased from the network; different communities can simply choose to present it differently.

## Reputation

There's no single global trust score. The protocol only stores raw, verifiable facts — how many events an identity published, which claims it holds, how many disputes it drew, how many endorsements it received. Each indexer turns those facts into its own reputation score, using whatever formula it wants. That means there's no one "official" score to game or centrally manipulate.

One input is deliberately *not* a protocol fact: **how old an identity is.** Timestamps are written by the author who signs them, so age is something each node observes locally rather than something the network can attest to. It's used, but it's an observation, and the weight of the defense sits on claims instead. See [protocol.md](./protocol.md#what-createdat-can-and-cannot-be-used-for).

## Progressive decentralization

The project won't launch with a thousand independent operators — day one, most infrastructure runs on servers controlled by the project itself. What matters is that the *architecture* already allows nodes, indexers, and APIs to be handed off to communities and third parties over time, without anyone losing their identity or history when that happens. Decentralization here is a direction the system is built to move in, not a launch-day requirement.
