/**
 * A review: its current version, every version before it, and the
 * conversation on it.
 */

// --- IMPORTS ---
import { AnimatePresence, motion, useReducedMotion } from 'motion/react'
import { Link } from 'react-router'
import { messageOf } from '@/app/forms/messages'
import { cardOf, repliesOf, versionsOf } from '@/app/reads/cards'
import { LoadMore } from '@/app/reads/LoadMore'
import { Placeholder } from '@/app/reads/Placeholder'
import { ShowLabeled } from '@/app/reads/ShowLabeled'
import { Status } from '@/app/reads/Status'
import { EditReview } from '@/app/reads/review/EditReview'
import { LabelForm } from '@/app/reads/review/LabelForm'
import { Respond } from '@/app/reads/review/Respond'
import { ReviewActions } from '@/app/reads/review/ReviewActions'
import { useReviewActions } from '@/app/reads/review/useReviewActions'
import { useReviewData } from '@/app/reads/review/useReviewData'
import { WithdrawReview } from '@/app/reads/review/WithdrawReview'
import { useMe } from '@/app/reads/useMe'
import { useInspection } from '@/app/reads/useInspection'
import { identifierOf, shortOf } from '@/app/reads/words'
import { Conversation } from '@/components/ecoa/Conversation'
import { EventInspector } from '@/components/ecoa/EventInspector'
import { ReviewCard } from '@/components/ecoa/ReviewCard'
import { VersionHistory } from '@/components/ecoa/VersionHistory'
import { Container } from '@/components/ui/Container'
import { useReveal } from '@/components/ui/useReveal'
import { useRise } from '@/components/ui/useRise'
import { HttpError } from '@/logic/errors/httpError'
import { useModeration } from '@/logic/moderation/moderation'
import { readEnvelope, reviewTextOf } from '@/logic/protocol/signed'
import { ROUTES } from '@/routes'

// --- GLOBALS ---
// the failures that are the review's state, not a fault
const GONE: Record<number, { title: string; body: string }> = {
  410: {
    title: 'Withdrawn by its author',
    body:
      'Its author took this review back. What it said is no longer shown ' +
      'here, though the signed event stays on the network’s logs.',
  },
  451: {
    title: 'Removed by moderation',
    body:
      'A moderator this index follows labeled it illegal or personal data, ' +
      'so nothing of it is shown. Another index, following others, may ' +
      'show it.',
  },
}

// --- CODE ---
/**
 * Renders the review screen.
 *
 * @returns The review, its history and its conversation.
 */
export function ReviewPage() {
  const {
    eventId,
    review,
    head,
    pending,
    entity,
    responses,
    names,
    labeled,
    showLabeled,
    expect,
    following,
  } = useReviewData()
  const { inspect, inspector } = useInspection()
  const me = useMe()
  const { choice } = useModeration()
  const actions = useReviewActions(pending ? null : head, expect, me.known)
  const moderating = choice.moderating && me.canAct && !pending
  const headId = head ? readEnvelope(head.envelope).eventId : null
  const ownLabels = (head?.observed.labels ?? []).filter(
    (applied) => applied.moderator === me.did,
  )
  const replies = repliesOf(responses.items, names)

  // the author edits and withdraws; anyone else votes
  const author = head ? readEnvelope(head.envelope).author : null
  const mine = author !== null && me.dids.includes(author)
  // the response being answered, named as the conversation names it
  const answered = replies.find((reply) => reply.eventId === actions.replyTo)
  const answering = answered
    ? (names[answered.authorId] ?? shortOf(answered.authorId))
    : null
  const rise = useRise()
  const reveal = useReveal()
  const reduceMotion = useReducedMotion()

  const status =
    review.error instanceof HttpError ? review.error.status : null
  const gone = status ? GONE[status] : undefined
  const versions = review.data ? versionsOf(review.data.versions, eventId) : []
  const old = versions.some((version) => version.asked && !version.current)

  // a note that comes or goes folds open or shut
  const fold = {
    initial: reduceMotion ? false : { opacity: 0, height: 0 },
    animate: { opacity: 1, height: 'auto' },
    exit: reduceMotion ? undefined : { opacity: 0, height: 0 },
    transition: { duration: reduceMotion ? 0 : 0.25 },
  } as const

  const conversation = pending
    ? 'Nobody has responded yet.'
    : responses.error
      ? messageOf(responses.error)
      : responses.loading
        ? 'Reading the conversation…'
        : responses.items.length === 0
          ? 'Nobody has responded yet.'
          : 'In the order it happened'

  return (
    <Container className="max-w-3xl pt-28 pb-16">
      <motion.p {...rise()} className="meta">
        {entity ? (
          <Link
            to={ROUTES.entity(entity)}
            className="text-muted transition-colors hover:text-primary"
          >
            ← {identifierOf(entity)}
          </Link>
        ) : (
          <Link
            to={ROUTES.explore}
            className="text-muted transition-colors hover:text-primary"
          >
            ← Explore
          </Link>
        )}
      </motion.p>

      <motion.h1 {...rise(0.06)} className="mt-4 font-display text-h2">
        Review
      </motion.h1>

      {review.loading && !pending ? (
        <div className="mt-6">
          <Placeholder count={1} lines={3} />
        </div>
      ) : null}

      {gone ? (
        <motion.section
          {...rise(0.1)}
          className="mt-6 border border-structural bg-surface p-6"
        >
          <h2 className="font-display text-h4">{gone.title}</h2>
          <p className="reading-measure mt-2 text-secondary">{gone.body}</p>
          <p className="meta mt-4 break-all text-muted">
            Event /{' '}
            <span className="font-mono normal-case">{eventId}</span>
          </p>
          {status === 410 ? (
            <button
              type="button"
              onClick={() => inspect(eventId)}
              className={
                'meta mt-4 cursor-pointer border-b-2 border-signal ' +
                'text-primary transition-colors hover:text-secondary'
              }
            >
              Inspect the signed event ↗
            </button>
          ) : null}
        </motion.section>
      ) : review.error && !pending ? (
        <motion.p {...rise(0.1)} role="alert" className="mt-6 text-danger">
          {messageOf(review.error)}
        </motion.p>
      ) : null}

      {head ? (
        <>
          <AnimatePresence initial={false}>
            {pending ? (
              <motion.div key="pending" {...fold} className="overflow-hidden">
                <p
                  role="note"
                  className={
                    'mt-6 border-l-2 border-info bg-info-wash px-4 py-3'
                  }
                >
                  Published. The index has not caught up with it yet, so this
                  is the review as your browser signed it; it is read again in
                  a moment.
                </p>
              </motion.div>
            ) : null}
          </AnimatePresence>

          {old ? (
            <motion.p
              {...rise(0.1)}
              role="note"
              className="mt-6 border-l-2 border-info bg-info-wash px-4 py-3"
            >
              The link points to an earlier version. Below is the current
              one; the version linked is marked in the history.
            </motion.p>
          ) : null}

          <motion.div {...rise(0.12)} className="mt-6">
            <ReviewCard
              review={cardOf(head, names, entity ?? undefined)}
              onInspect={
                pending
                  ? undefined
                  : () => inspect(readEnvelope(head.envelope).eventId)
              }
              actions={
                pending ? null : (
                  <ReviewActions
                    canAct={me.canAct}
                    sessionKnown={me.sessionKnown}
                    mine={mine}
                    busy={actions.busy}
                    onEdit={() => actions.open('editing')}
                    onWithdraw={() => actions.open('withdrawing')}
                    onVote={(kind) => void actions.vote(kind)}
                    onLabel={
                      moderating && headId
                        ? () => actions.setLabeling(headId)
                        : undefined
                    }
                  />
                )
              }
            />
          </motion.div>

          <AnimatePresence initial={false} mode="wait">
            {actions.mode === 'editing' ? (
              <motion.div key="editing" {...fold} className="overflow-hidden">
                <div className="mt-4">
                  <EditReview
                    current={reviewTextOf(readEnvelope(head.envelope))}
                    busy={actions.busy}
                    onSave={actions.edit}
                    onCancel={() => actions.open('idle')}
                  />
                </div>
              </motion.div>
            ) : actions.mode === 'withdrawing' ? (
              <motion.div
                key="withdrawing"
                {...fold}
                className="overflow-hidden"
              >
                <div className="mt-4">
                  <WithdrawReview
                    busy={actions.busy}
                    onWithdraw={actions.withdraw}
                    onCancel={() => actions.open('idle')}
                  />
                </div>
              </motion.div>
            ) : null}
          </AnimatePresence>

          <AnimatePresence initial={false}>
            {actions.labeling ? (
              <motion.div key="labeling" {...fold} className="overflow-hidden">
                <div className="mt-4">
                  <LabelForm
                    what={
                      actions.labeling === headId
                        ? 'this review'
                        : 'this response'
                    }
                    busy={actions.busy}
                    onLabel={actions.label}
                    onCancel={() => actions.setLabeling(null)}
                  />
                </div>
              </motion.div>
            ) : null}
          </AnimatePresence>

          {moderating && ownLabels.length > 0 ? (
            <ul className="mt-3 flex flex-col gap-2">
              {ownLabels.map((applied) => (
                <li
                  key={applied.eventId}
                  className="flex flex-wrap items-center gap-3"
                >
                  <span className="meta text-muted">
                    Your label · {applied.label}
                  </span>
                  <button
                    type="button"
                    disabled={actions.busy}
                    onClick={() => void actions.revoke(applied.eventId)}
                    className={
                      'meta cursor-pointer text-secondary hover:text-primary'
                    }
                  >
                    Revoke
                  </button>
                </li>
              ))}
            </ul>
          ) : null}

          <Status
            text={
              actions.error ??
              (following
                ? `${actions.notice ?? 'Published.'} Waiting for the index…`
                : (actions.notice ?? ''))
            }
            failed={!!actions.error}
            className="mt-3"
          />
          {actions.needsIdentity ? (
            <Link
              to={ROUTES.identity}
              className="meta mt-1 inline-block border-b border-signal"
            >
              Create or restore your identity →
            </Link>
          ) : null}

          {versions.length > 1 ? (
            <motion.section {...reveal()} className="mt-12">
              <h2 className="font-display text-h4">
                History · {versions.length} versions
              </h2>
              <p className="meta mt-1 text-muted">
                An edit adds a version; none is erased.
              </p>
              <VersionHistory
                versions={versions}
                onInspect={inspect}
                className="mt-4"
              />
            </motion.section>
          ) : null}

          <motion.section {...reveal()} className="mt-12">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <h2 className="font-display text-h4">Conversation</h2>
              <ShowLabeled checked={labeled} onChange={showLabeled} />
            </div>
            <Status
              text={conversation}
              failed={!pending && !!responses.error}
              className="mt-1"
            />
            {responses.loading && !pending ? (
              <div className="mt-4">
                <Placeholder count={2} lines={1} />
              </div>
            ) : null}
            <Conversation
              replies={replies}
              onInspect={inspect}
              onReply={me.canAct && !pending ? actions.setReplyTo : undefined}
              onLabel={moderating ? actions.setLabeling : undefined}
              className="mt-4"
            />
            <LoadMore
              hasMore={responses.hasMore}
              more={responses.more}
              onLoad={responses.loadMore}
            />
            {me.canAct && !pending ? (
              <Respond
                answering={answering}
                busy={actions.busy}
                onRespond={actions.respond}
                onCancel={() => actions.setReplyTo(null)}
              />
            ) : null}
          </motion.section>
        </>
      ) : null}

      <EventInspector {...inspector} />
    </Container>
  )
}
