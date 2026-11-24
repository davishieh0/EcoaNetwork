/**
 * The entity screen's data: the entity, its reviews, and their authors.
 */

// --- IMPORTS ---
import { useCallback } from 'react'
import { useParams, useSearchParams } from 'react-router'
import { useAuthors } from '@/app/reads/useAuthors'
import { usePaged } from '@/app/reads/usePaged'
import { useRead } from '@/app/reads/useRead'
import { getEntity, listEntityReviews } from '@/logic/datasource/entities'
import { useModeration } from '@/logic/moderation/moderation'
import { readEnvelope } from '@/logic/protocol/signed'

// --- CODE ---
/**
 * Reads the entity in the address and its reviews, showing what moderation
 * hides when `?labeled=show` asks for it.
 *
 * @returns The entity, its reviews, their authors' names, and the toggle.
 */
export function useEntityData() {
  const entityId = useParams().entity ?? ''
  const [params, setParams] = useSearchParams()
  const labeled = params.get('labeled') === 'show' ? 'show' : undefined

  const { param } = useModeration()

  const readEntity = useCallback(
    () => getEntity(entityId, param),
    [entityId, param],
  )
  const entity = useRead(readEntity)

  const loadReviews = useCallback(
    (after?: string) => listEntityReviews(entityId, after, labeled, param),
    [entityId, labeled, param],
  )
  const reviews = usePaged(loadReviews)

  const names = useAuthors(
    reviews.items.map((item) => readEnvelope(item.envelope).author),
    param,
  )

  const showLabeled = useCallback(
    (show: boolean) => setParams(show ? { labeled: 'show' } : {}),
    [setParams],
  )

  return { entityId, entity, reviews, names, labeled: !!labeled, showLabeled }
}
