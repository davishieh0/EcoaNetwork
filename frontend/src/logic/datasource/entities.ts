/**
 * The entity routes: search, an entity, and its reviews.
 */

// --- IMPORTS ---
import type { Entity, ReviewItem } from '@/logic/datasource/entities.t'
import { request, route } from '@/logic/datasource/http'
import type { Labeled, Page } from '@/logic/datasource/signed.t'

// --- CODE ---
/**
 * Finds entities with a name that holds the text, newest first.
 *
 * @param q - Part of a name; undefined for every entity.
 * @param after - The next of a previous page.
 * @returns A page of entities.
 */
export function searchEntities(q?: string, after?: string, moderators?: string) {
  return request<Page<Entity>>(route(['entities'], { q, after, moderators }))
}

/**
 * Reads an entity and the value of each of its fields.
 *
 * @param entity - The entity id, such as `cnpj:12345678000199`.
 * @returns The entity.
 */
export function getEntity(entity: string, moderators?: string) {
  return request<Entity>(route(['entities', entity], { moderators }))
}

/**
 * Lists the current version of each of an entity's reviews, newest first.
 *
 * @param entity - The entity id.
 * @param after - The next of a previous page.
 * @param labeled - 'show', to see what moderation hides, marked.
 * @returns A page of reviews.
 */
export function listEntityReviews(
  entity: string,
  after?: string,
  labeled?: Labeled,
  moderators?: string,
) {
  return request<Page<ReviewItem>>(
    route(['entities', entity, 'reviews'], { after, labeled, moderators }),
  )
}
