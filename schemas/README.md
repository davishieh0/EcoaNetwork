# Schemas

JSON Schema (draft 2020-12) for the envelope and for the 18 event types registered in
[`docs/protocol.md`](../docs/protocol.md). These are a transcription of the specification, not a
second source of truth: where a schema and the spec disagree, the spec is right and the schema
is the bug.

```
common.schema.json     shared definitions — identifiers, text, enums
envelope.schema.json   the envelope, a closed set
payloads/*.json        one per eventType
```

## Three decisions that explain the shape of these files

**The envelope is closed, the payloads are open.** `envelope.schema.json` sets
`additionalProperties: false`; every payload sets `true`. That is not an inconsistency — it is the
versioning rule: a new optional payload field must not bump the envelope version, and must
survive a node that cannot interpret it, preserved byte-exactly for the digest. An unknown field
at the top of the envelope, by contrast, would make two nodes compute different digests for the
same event.

**Envelope field names are forbidden inside a payload.** Every payload declares
`"author": false`, `"createdAt": false` and so on. The convention "nothing in a payload duplicates
the envelope" lived only in prose; here it is checkable. It does not catch an invented synonym
such as `oldIdentity` — no schema would.

**Absent is not null.** An optional field is omitted, never `null`, and the schemas reject `null`
explicitly. The project's single exception is `deviceKey` in the envelope, which is required and
nullable, because under JCS a null-valued key and an absent key produce different digests.

## Two things a schema cannot check

**NFC normalization.** JSON Schema does not express Unicode form. The schemas check length and
the absence of control characters; normalization is the client's responsibility, applied before
signing, and a node never re-applies it — doing so would alter the signed bytes.

**Control characters, with one distinction.** A single-line field (`title`, `name`,
`displayName`, `label`) permits no control character at all. A multi-line field (`body`, `bio`,
`description`, `note`) permits `\n` and nothing else, because a review body legitimately
contains line breaks.

## RE2 compatibility

No `pattern` uses lookahead or lookbehind. The node is written in Go, and Go's `regexp` is RE2,
which supports neither. A schema that relied on them would compile under a JavaScript validator
and fail in the reference implementation.

Three more habits keep the verdict the same under other engines. Whitespace is an explicit list of
`\uXXXX` escapes, because `\s` means different sets in RE2 and Python. No pattern uses `\p` or
`\x{...}`, which Python `re` lacks. And `$` means the true end of the string: Python's `re` also
lets it match before a final newline, so a validator built on it has to anchor at the real end, or
it accepts what `vectors/rejections.json` refuses.

## Validating

The schemas compile under strict mode and are exercised by cases covering both acceptance and —
the half that matters — **rejection**. The rejection cases worth knowing about:

- an envelope with an unknown top-level field, or with `deviceKey` omitted
- `createdAt` carrying a numeric offset, or milliseconds
- `rating` outside 1–5, fractional, or as a string
- a CNPJ with punctuation, a domain in uppercase or carrying a scheme
- `review.replaced` carrying `entity` — a review's subject is immutable by construction
- an `entity:` identifier without its descriptor, or a public identifier carrying one
- a `domain` claim without `entity`, or a `human` claim with one
- a withdrawal reason given as free text instead of the enum
