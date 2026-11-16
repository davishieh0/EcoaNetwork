/** The review form and its local preview. */

// --- IMPORTS ---
import { useState } from 'react'

import { Rating } from '@/components/ecoa/Rating'
import { Button } from '@/components/ui/Button'
import { Container } from '@/components/ui/Container'

// --- GLOBALS ---
const INPUT = 'w-full border border-structural bg-surface p-3'

// --- CODE ---
/** Lets a reviewer compose and preview an evaluation. */
export function WriteReview() {
  const [preview, setPreview] = useState<FormData | null>(null)

  return (
    <Container className="max-w-3xl pt-28 pb-16">
      <h1 className="font-display text-h1">Write a review</h1>
      <p className="text-secondary">Publishing is currently unavailable.</p>

      <form
        className="mt-8 grid gap-4"
        onChange={() => setPreview(null)}
        onSubmit={(event) => {
          event.preventDefault()
          setPreview(new FormData(event.currentTarget))
        }}
      >
        <label className="grid gap-2">
          <span>Business, product, or service</span>
          <input name="entity" className={INPUT} required />
        </label>
        <label className="grid gap-2">
          <span>Rating</span>
          <select name="rating" className={INPUT} defaultValue="" required>
            <option value="" disabled>Choose a rating</option>
            {[1, 2, 3, 4, 5].map((rating) => (
              <option key={rating} value={rating}>{rating} / 5</option>
            ))}
          </select>
        </label>
        <label className="grid gap-2">
          <span>Title</span>
          <input name="title" className={INPUT} required />
        </label>
        <label className="grid gap-2">
          <span>Your experience</span>
          <textarea name="body" className={INPUT} rows={6} required />
        </label>
        <Button type="submit" className="justify-self-start">
          Preview review
        </Button>
      </form>

      <div aria-live="polite">
        {preview && (
          <section className="mt-8 grid gap-3 border border-structural p-5">
            <h2 className="meta text-muted">Preview</h2>
            <p className="font-display text-h4 break-words">
              {String(preview.get('entity'))}
            </p>
            <Rating value={Number(preview.get('rating'))} />
            <h3 className="font-semibold break-words">
              {String(preview.get('title'))}
            </h3>
            <p className="text-secondary break-words whitespace-pre-wrap">
              {String(preview.get('body'))}
            </p>
          </section>
        )}
      </div>
    </Container>
  )
}
