/**
 * Types for the Conversation.
 */

// --- GLOBALS ---
/** One response, as the conversation shows it. */
export type Reply = {
  eventId: string
  author: string
  /** The identity that signed it, shown beside the name. */
  authorId: string
  body: string
  publishedAt: string
  /** 1 answers the review; deeper answers a response. Indented up to 2. */
  depth: number
  /** Who it answers, when it answers a response, as a reader names them. */
  replyTo?: string
  /** It answered a version the author has since replaced. */
  textChanged: boolean
  labels?: string[]
}

export type ConversationProps = {
  /** In the order of the conversation. */
  replies: Reply[]
  onInspect?: (eventId: string) => void
  className?: string
}
