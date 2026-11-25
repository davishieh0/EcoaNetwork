/**
 * Landing copy.
 */

// --- IMPORTS ---
import type { ClaimGlyphKind } from '@/components/ecoa/ClaimGlyph.t'
import type { ProblemGlyphKind } from '@/site/landing/motifs/ProblemGlyph.t'

// --- GLOBALS ---
/** Landing copy lives here rather than inside the components. */
export const PROBLEMS = [
  {
    index: '01',
    glyph: 'captive',
    title: 'The review is not yours',
    body:
      'You write it, and it becomes a row in a database you cannot read, ' +
      'export or take with you.',
  },
  {
    index: '02',
    glyph: 'overwritten',
    title: 'History can change quietly',
    body:
      'An edited review can replace the original with no trace that an ' +
      'earlier version ever existed.',
  },
  {
    index: '03',
    glyph: 'unverifiable',
    title: 'Origin cannot be checked',
    body:
      'There is no way for a reader to establish who actually wrote ' +
      'something, or whether it was altered afterwards.',
  },
  {
    index: '04',
    glyph: 'central',
    title: 'One company decides',
    body:
      'The same organisation controls ranking, presentation and moderation, ' +
      'and answers to no one outside itself.',
  },
] as const satisfies ReadonlyArray<{
  index: string
  glyph: ProblemGlyphKind
  title: string
  body: string
}>

/** The comparison, as dimensions rather than two disconnected lists. */
export const COMPARISON = [
  {
    aspect: 'Where it lives',
    traditional: 'One private database',
    ecoa: 'Replicated across independent nodes',
  },
  {
    aspect: 'When you edit',
    traditional: 'The original is overwritten',
    ecoa: 'A new version publishes, the old one stays',
  },
  {
    aspect: 'Proof of origin',
    traditional: 'Nothing a reader can check',
    ecoa: 'Every event is signed and checkable',
  },
  {
    aspect: 'How ranking works',
    traditional: 'Computed internally, rules unpublished',
    ecoa: 'Computed from events anyone can read',
  },
  {
    aspect: 'If you leave',
    traditional: 'The record stays behind',
    ecoa: 'The record travels with your identity',
  },
] as const

export const STEPS = [
  {
    index: '01',
    title: 'Experience',
    body:
      'Something happens. A meal, a repair, a delivery, a service that went ' +
      'well or badly.',
  },
  {
    index: '02',
    title: 'Sign',
    body:
      'You write it and sign it with a key only you hold. The signature ' +
      'travels with the text.',
  },
  {
    index: '03',
    title: 'Publish',
    body:
      'The signed review becomes an event: a small, self-contained object ' +
      'that carries its own proof.',
  },
  {
    index: '04',
    title: 'Echo',
    body:
      'Independent nodes take a copy. Anyone holding one can confirm the ' +
      'event is exactly what you signed.',
  },
] as const

/** Claims, for the Trust section. */
export const CLAIMS = [
  {
    label: 'Domain verified',
    glyph: 'domain',
    asserts: 'This account controls marés-bakery.pt',
    checkedBy: 'A signed record published at that domain',
    doesNotMean: 'That the business is honest or well run',
    tone: 'valid',
  },
  {
    label: 'Human claim',
    glyph: 'human',
    asserts: 'A person vouched that this account is a real individual',
    checkedBy: 'Another identity, which staked its own reputation',
    doesNotMean: 'That their legal name is known to anyone',
    tone: 'neutral',
  },
  {
    label: 'Official representative',
    glyph: 'delegate',
    asserts: 'This account may answer on behalf of the entity',
    checkedBy: 'The entity, in a signed delegation it can revoke',
    doesNotMean: 'That their answers are neutral',
    tone: 'info',
  },
  {
    label: 'Purchase attested',
    glyph: 'purchase',
    asserts: 'A transaction matching this review existed',
    checkedBy: 'A receipt issuer who signed the attestation',
    doesNotMean: 'That the review is fair, only that the visit happened',
    tone: 'valid',
  },
] as const satisfies ReadonlyArray<{
  label: string
  glyph: ClaimGlyphKind
  asserts: string
  checkedBy: string
  doesNotMean: string
  tone: 'valid' | 'neutral' | 'info'
}>

/** The version chain behind one review. */
export const HISTORY = [
  {
    title: 'Joana wrote it',
    at: '14 August',
    note:
      'She published the first version and signed it, the way you sign a ' +
      'letter.',
    event: 'review.created',
    state: 'past',
  },
  {
    title: 'She corrected a date',
    at: '16 August',
    note:
      'Two days later she fixed a detail. The first version is still there ' +
      'to ' +
      'read, and the change is visible to anyone.',
    event: 'review.replaced',
    state: 'past',
  },
  {
    title: 'Someone agreed',
    at: '19 August',
    note: 'A different person put their name behind this version.',
    event: 'review.endorsed',
    state: 'past',
  },
  {
    title: 'What you see today',
    at: 'now',
    note: 'The latest version, with every step above still reachable.',
    event: 'current head',
    state: 'current',
  },
] as const

export const NETWORK_FACTS = [
  { value: '05', label: 'Independent nodes' },
  { value: '01', label: 'Protocol' },
  { value: '00', label: 'Definitive servers' },
] as const

export const NETWORK_POINTS = [
  'No server is the authority. Nodes hold copies, and copies agree because ' +
    'the events are signed.',
  'A node can disappear without taking the record with it.',
  'Anyone holding a copy can verify an event without asking us.',
  'The infrastructure is replaceable. The protocol is the thing that persists.',
] as const

/** A company, a product and a service, not a complaints board. */
export const HUMAN_REVIEWS = [
  {
    entity: 'Alfama Bike Works',
    kind: 'Local business',
    rating: 4,
    body:
      'The bill came in forty euros over the estimate. I asked about it ' +
      'once, ' +
      'they walked me through the part that caused it, and then took it off. ' +
      'Nobody made me push for that.',
    author: 'p.aguiar',
  },
  {
    entity: 'Halcyon 2 headphones',
    kind: 'Product',
    rating: 3,
    body:
      'They sound better than anything near the price and the hinge broke in ' +
      'four months. Both things are true and most reviews only tell you one ' +
      'of them.',
    author: 'r.mendes',
  },
  {
    entity: 'Northbound Movers',
    kind: 'Service',
    rating: 5,
    body:
      'They quoted a price, hit a wall of traffic, arrived two hours late, ' +
      'and charged the quote anyway. I did not have to argue for that, which ' +
      'is the whole review.',
    author: 'tamsin',
  },
] as const

/** The FAQ. */
export const FAQ = [
  {
    question: 'Do I need an account to read reviews?',
    answer:
      'No. Reading is public and always will be. An account exists so you ' +
      'can ' +
      'sign what you write, not so you can see what others wrote.',
  },
  {
    question: 'Does it cost anything?',
    answer:
      'Reading and writing are free. Ecoa is a protocol rather than a ' +
      'company ' +
      'sitting between you and the review, so there is no placement to sell ' +
      'and no ranking to auction.',
  },
  {
    question: 'Do I need to use my real name?',
    answer:
      'No. An identity here is a key, not a legal name. You can stay ' +
      'pseudonymous and still be accountable, because everything you publish ' +
      'is provably by the same person.',
  },
  {
    question: 'Can a business delete a bad review about it?',
    answer:
      'No. A business never held the review in the first place — the author ' +
      'signed it and independent nodes hold copies. It can reply, and it can ' +
      'dispute, and both of those are public too.',
  },
  {
    question: 'Can a business reply to me?',
    answer:
      'Yes, and its reply is signed the same way yours is. If an account ' +
      'claims to speak for a business, you can see what backs that claim and ' +
      'who granted it.',
  },
  {
    question: 'What stops people writing fake reviews?',
    answer:
      'Nothing stops the writing. What changes is that a fresh identity with ' +
      'no history looks like exactly that, and a thousand of them created ' +
      'the ' +
      'same afternoon are visible as a pattern rather than as a score.',
  },
  {
    question: 'Can I delete something I wrote?',
    answer:
      'You can withdraw it, which marks it as withdrawn everywhere and stops ' +
      'it being shown. Copies already published cannot be reached into and ' +
      'erased — that is the cost of nobody being able to erase yours either.',
  },
  {
    question: 'Can a review be edited?',
    answer:
      'Yes, and the edit is published as a new version rather than written ' +
      'over the old one. The earlier version stays part of the record, so an ' +
      'edit can never be silent.',
  },
  {
    question: 'What happens if I lose access to my identity?',
    answer:
      'This is the genuinely hard part. A lost key cannot be recovered by ' +
      'us, ' +
      'because we never had it. You can authorise a second device in advance ' +
      'and name a successor identity, and doing that early is worth the five ' +
      'minutes.',
  },
  {
    question: 'What does "verified" mean here?',
    answer:
      'On its own, nothing. Ecoa never issues a single badge. It shows which ' +
      'specific claim was checked, who checked it, and what that claim does ' +
      'not cover.',
  },
  {
    question: 'Who controls moderation?',
    answer:
      'No one globally. Moderation labels are themselves signed events ' +
      'published by whoever applied them, so you can see who made a call and ' +
      'decide whether to honour it.',
  },
  {
    question: 'Is Ecoa a blockchain?',
    answer:
      'No. There is no chain, no consensus round and no token. Reviews are ' +
      'signed events that independent nodes copy, and a signature is what ' +
      'makes one checkable — not agreement between miners.',
  },
] as const
