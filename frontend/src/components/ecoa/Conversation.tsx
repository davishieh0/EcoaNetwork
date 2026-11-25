/**
 * The conversation on a review: responses, and responses to them.
 */

// --- IMPORTS ---
import { motion } from 'motion/react'
import type { ConversationProps } from '@/components/ecoa/Conversation.t'
import { Badge } from '@/components/ui/Badge'
import { cn } from '@/components/ui/cn'
import { useReveal } from '@/components/ui/useReveal'

// --- GLOBALS ---
// a response to the review sits at depth 1, flush; one to a response is
// indented once, and deeper ones stay there, saying what they answer
const INDENT = ['', '', 'ml-8']

// --- CODE ---
/**
 * Lists the responses in order, indented by what they answer.
 *
 * @param props - The responses, an optional inspect handler, extra classes.
 * @returns The conversation.
 */
export function Conversation({
  replies,
  onInspect,
  onReply,
  onLabel,
  className,
}: ConversationProps) {
  const reveal = useReveal()

  return (
    <ol className={cn('flex flex-col gap-3', className)}>
      {replies.map((reply) => (
        <motion.li
          {...reveal()}
          key={reply.eventId}
          className={cn(
            'border bg-surface px-5 py-4',
            reply.official ? 'border-valid' : 'border-structural',
            INDENT[Math.min(reply.depth, 2)],
          )}
        >
          <div className="flex flex-wrap items-center gap-2">
            {reply.official ? (
              <Badge tone="valid" glyph="◆">
                Official response
              </Badge>
            ) : null}
            <span className="meta text-secondary">{reply.author}</span>
            {reply.human && !reply.official ? (
              <Badge tone="info" glyph="◆">
                Human
              </Badge>
            ) : null}
            <span
              title={reply.authorId}
              className="font-mono text-mono-s break-all text-muted"
            >
              {reply.authorId}
            </span>
            <span className="meta ml-auto text-muted">{reply.publishedAt}</span>
          </div>

          {reply.replyTo ? (
            <p className="meta mt-2 text-muted">
              ↳ In reply to {reply.replyTo}
            </p>
          ) : null}

          <p
            className={cn(
              'reading-measure mt-2 whitespace-pre-line text-secondary',
              'text-pretty',
            )}
          >
            {reply.body}
          </p>

          <div className="mt-2 flex flex-wrap items-center gap-2">
            {reply.textChanged ? (
              <Badge tone="neutral" glyph="■">
                Answered an earlier version
              </Badge>
            ) : null}
            {reply.labels?.map((label) => (
              <Badge key={label} tone="warning" glyph="●">
                Labeled · {label}
              </Badge>
            ))}
            {onLabel ? (
              <button
                type="button"
                onClick={() => onLabel(reply.eventId)}
                className={cn(
                  'meta cursor-pointer text-muted transition-colors',
                  'hover:text-warning',
                  onReply ? '' : 'ml-auto',
                )}
              >
                Label…
              </button>
            ) : null}
            {onReply ? (
              <button
                type="button"
                onClick={() => onReply(reply.eventId)}
                className={cn(
                  'meta ml-auto cursor-pointer text-muted transition-colors',
                  'hover:text-primary',
                )}
              >
                Reply
              </button>
            ) : null}
            {onInspect ? (
              <button
                type="button"
                onClick={() => onInspect(reply.eventId)}
                className={cn(
                  'meta cursor-pointer border-b border-signal text-primary',
                  'hover:text-secondary',
                  onReply ? '' : 'ml-auto',
                )}
              >
                Inspect ↗
              </button>
            ) : null}
          </div>
        </motion.li>
      ))}
    </ol>
  )
}
