/**
 * Withdrawing a review: its author takes the whole chain back.
 */

// --- IMPORTS ---
import { useState } from 'react'
import type { WithdrawReason } from '@/app/reads/review/ReviewPage.t'
import type { WithdrawReviewProps } from '@/app/reads/review/WithdrawReview.t'
import { Button } from '@/components/ui/Button'
import { Select } from '@/components/ui/Select'

// --- GLOBALS ---
// the protocol's fixed list: free text here would be permanent content in
// the one event meant to take content back
const REASONS: { value: WithdrawReason | ''; label: string }[] = [
  { value: '', label: 'No reason given' },
  { value: 'mistake', label: 'It was a mistake' },
  { value: 'resolved', label: 'The problem was resolved' },
  { value: 'duplicate', label: 'It repeats another review' },
  { value: 'privacy', label: 'Privacy' },
  { value: 'other', label: 'Another reason' },
]

// --- CODE ---
/**
 * Asks the author to confirm, with an optional reason from the fixed list.
 *
 * @param props - Whether it is on its way, and the handlers.
 * @returns The confirmation.
 */
export function WithdrawReview({
  busy,
  onWithdraw,
  onCancel,
}: WithdrawReviewProps) {
  const [reason, setReason] = useState<WithdrawReason | ''>('')

  return (
    <section
      aria-label="Withdraw your review"
      className="flex flex-col gap-4 border border-danger bg-surface p-6"
    >
      <div className="flex flex-col gap-1">
        <h2 className="font-display text-h4">Withdraw your review?</h2>
        <p className="text-body-s text-secondary">
          Every version stops being shown here. The signed events stay on
          the network’s logs, as everything signed does, and anyone can
          still inspect them. This cannot be undone.
        </p>
      </div>

      <Select
        label="Reason"
        value={reason}
        options={REASONS}
        onChange={(value) => setReason(value as WithdrawReason | '')}
      />

      <div className="flex flex-wrap justify-end gap-3">
        <Button variant="ghost" type="button" onClick={onCancel}>
          Keep it
        </Button>
        <Button
          type="button"
          pending={busy}
          onClick={() => void onWithdraw(reason || null)}
        >
          {busy ? 'Signing and sending' : 'Withdraw'}
        </Button>
      </div>
    </section>
  )
}
