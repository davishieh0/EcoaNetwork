/** A public directory of example reviews. */

// --- IMPORTS ---
import { useState } from 'react'
import { Link } from 'react-router'

import { Rating } from '@/components/ecoa/Rating'
import { Button } from '@/components/ui/Button'
import { Container } from '@/components/ui/Container'
import { ROUTES } from '@/routes'
import { HUMAN_REVIEWS } from '@/site/landing/content'

// --- CODE ---
/** Renders the review directory and filters it by the search term. */
export function Explore() {
  const [query, setQuery] = useState('')
  const term = query.trim().toLowerCase()
  const reviews = HUMAN_REVIEWS.filter(({ entity, kind, body, author }) =>
    [entity, kind, body, author].some((value) =>
      value.toLowerCase().includes(term),
    ),
  )

  return (
    <Container className="max-w-4xl pt-28 pb-16">
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="font-display text-h1">Explore</h1>
          <p className="text-secondary">Browse example reviews from Ecoa.</p>
        </div>
        <Button as={Link} to={ROUTES.write} variant="secondary">
          Write a review
        </Button>
      </header>

      <label className="mt-8 flex flex-col gap-2">
        <span>Search reviews</span>
        <input
          type="search"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="Business, product, or service"
          className="w-full border border-structural bg-surface p-3"
        />
      </label>

      <p role="status" className="my-4 text-secondary">
        {reviews.length === 0
          ? 'No matching reviews.'
          : `${reviews.length} of ${HUMAN_REVIEWS.length} example reviews`}
      </p>

      <ul className="grid gap-4">
        {reviews.map(({ entity, kind, rating, body, author }) => (
          <li key={entity} className="border border-structural bg-surface p-5">
            <article className="grid gap-3">
              <p className="meta text-muted">{kind}</p>
              <h2 className="font-display text-h4">{entity}</h2>
              <Rating value={rating} />
              <p className="text-secondary">{body}</p>
              <p className="meta text-muted">@{author}</p>
            </article>
          </li>
        ))}
      </ul>
    </Container>
  )
}
