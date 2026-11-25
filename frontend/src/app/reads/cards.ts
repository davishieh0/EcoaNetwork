/**
 * What the index answers, as the review components show it.
 */

// --- IMPORTS ---
import { dateOf, shortOf } from '@/app/reads/words'
import type { Reply } from '@/components/ecoa/Conversation.t'
import type { Review } from '@/components/ecoa/ReviewCard.t'
import type { Version } from '@/components/ecoa/VersionHistory.t'
import { i18n } from '@/i18n/i18n'
import type { ReviewItem } from '@/logic/datasource/entities.t'
import type { ResponseItem } from '@/logic/datasource/reviews.t'
import type { Recognized } from '@/logic/datasource/signed.t'
import { readEnvelope, reviewTextOf, textOf } from '@/logic/protocol/signed'

// --- CODE ---
/**
 * The mark a card shows beside its author: the entity's verified
 * representative first, a human claim otherwise.
 *
 * @param claims - The claims that count for the author.
 * @param entity - The entity the review is about, when known.
 * @returns The mark, or undefined.
 */
export function claimMarkOf(
  claims: Recognized[] | undefined,
  entity?: string,
): Review['authorClaim'] {
  if (claims?.some((c) => c.claimType === 'domain' && c.entity === entity)) {
    return { label: i18n.t('Verified representative'), tone: 'valid' }
  }

  if (claims?.some((c) => c.claimType === 'human')) {
    return { label: i18n.t('Human'), tone: 'info' }
  }

  return undefined
}

/**
 * Turns a signed version of a review into what the card shows.
 *
 * @param item - The version, as the index answered it.
 * @param names - The display names found, by did:key.
 * @param entity - The entity it is about, for the representative's mark.
 * @returns The card's review.
 */
export function cardOf(
  item: ReviewItem,
  names: Record<string, string>,
  entity?: string,
): Review {
  const envelope = readEnvelope(item.envelope)
  const text = reviewTextOf(envelope)

  return {
    authorClaim: claimMarkOf(item.observed.claims, entity ?? text.entity),
    rating: text.rating,
    title: text.title,
    body: text.body,
    author: names[envelope.author] ?? i18n.t('No display name'),
    authorId: envelope.author,
    publishedAt: dateOf(envelope.createdAt),
    eventId: envelope.eventId,
    observedBy: item.observed.nodes?.length,
    edited: item.derived.edited || envelope.eventType === 'review.replaced',
    endorsements: item.derived.endorsements,
    disputes: item.derived.disputes,
    labels: item.observed.labels?.map((applied) => applied.label),
  }
}

/**
 * Numbers a review's versions, oldest first, and marks the current one and
 * the one the link pointed at.
 *
 * @param versions - The chain, as the index answered it, oldest first.
 * @param asked - The eventId in the link.
 * @returns The versions, as the history shows them.
 */
export function versionsOf(versions: ReviewItem[], asked: string): Version[] {
  return versions.map((item, index) => {
    const envelope = readEnvelope(item.envelope)
    const text = reviewTextOf(envelope)

    return {
      eventId: envelope.eventId,
      number: index + 1,
      publishedAt: dateOf(envelope.createdAt),
      rating: text.rating,
      title: text.title,
      body: text.body,
      endorsements: item.derived.endorsements,
      disputes: item.derived.disputes,
      current: item.derived.head,
      asked: envelope.eventId === asked,
    }
  })
}

/**
 * Turns a conversation's signed responses into what it shows, each reply to
 * a response saying whose it answers.
 *
 * @param items - The responses so far, as the index answered them.
 * @param names - The display names found, by did:key.
 * @returns The conversation's replies.
 */
export function repliesOf(
  items: ResponseItem[],
  names: Record<string, string>,
): Reply[] {
  const authors = new Map(
    items.map((item) => {
      const envelope = readEnvelope(item.envelope)
      return [envelope.eventId, envelope.author]
    }),
  )

  const whose = (eventId: string) => {
    const author = authors.get(eventId)
    if (!author) return i18n.t('response {{id}}', { id: shortOf(eventId) })
    return names[author] ?? shortOf(author)
  }

  return items.map((item) => replyOf(item, names, whose))
}

/**
 * Turns one signed response into what the conversation shows.
 *
 * @param item - The response, as the index answered it.
 * @param names - The display names found, by did:key.
 * @param whose - Names who wrote the response it answers.
 * @returns The conversation's reply.
 */
function replyOf(
  item: ResponseItem,
  names: Record<string, string>,
  whose: (eventId: string) => string,
): Reply {
  const envelope = readEnvelope(item.envelope)

  return {
    eventId: envelope.eventId,
    author: names[envelope.author] ?? i18n.t('No display name'),
    authorId: envelope.author,
    body: textOf(envelope, 'body') ?? '',
    publishedAt: dateOf(envelope.createdAt),
    depth: item.derived.depth,
    replyTo:
      item.derived.replyTo === item.derived.respondedVersion
        ? undefined
        : whose(item.derived.replyTo),
    textChanged: item.derived.textChanged,
    labels: item.observed.labels?.map((applied) => applied.label),
    official: item.observed.official ?? false,
    human:
      item.observed.claims?.some((c) => c.claimType === 'human') ?? false,
  }
}
