/**
 * The entity routes: search, an entity, and its reviews.
 */

// --- IMPORTS ---
import type { Entity } from '@/logic/datasource/entities.t'
import { request, route } from '@/logic/datasource/http'
import type { Page } from '@/logic/datasource/signed.t'

// --- CODE ---
/**
 * Finds entities with a name that holds the text, newest first.
 *
 * @param q - Part of a name; undefined for every entity.
 * @param after - The next of a previous page.
 * @returns A page of entities.
 */
export function searchEntities(q?: string, after?: string) {
  return request<Page<Entity>>(route(['entities'], { q, after }))
}

/**
 * Reads an entity and the value of each of its fields.
 *
 * @param entity - The entity id, such as `cnpj:12345678000199`.
 * @returns The entity.
 */
export function getEntity(entity: string) {
  return request<Entity>(route(['entities', entity]))
}
