/**
 * How the read screens word what the network holds: entity ids, dates.
 */

// --- IMPORTS ---
import { dateLocale } from '@/i18n/format'
import type { Entity } from '@/logic/datasource/entities.t'

// --- GLOBALS ---
const KINDS: Record<string, string> = {
  cnpj: 'Business · CNPJ',
  domain: 'Website',
  gtin: 'Product · GTIN',
  entity: 'Entity',
}

const DATE: Intl.DateTimeFormatOptions = {
  day: 'numeric',
  month: 'short',
  year: 'numeric',
}

const DATE_TIME: Intl.DateTimeFormatOptions = {
  day: 'numeric',
  month: 'short',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  timeZoneName: 'short',
}

// --- CODE ---
/**
 * Says what kind of thing an entity id names.
 *
 * @param entity - The entity id, such as `cnpj:12345678000199`.
 * @returns The kind, in words.
 */
export function kindOf(entity: string): string {
  return KINDS[entity.slice(0, entity.indexOf(':'))] ?? 'Entity'
}

/**
 * Writes an entity id the way people read it: a CNPJ with its dots.
 *
 * @param entity - The entity id.
 * @returns The identifier, without its kind.
 */
export function identifierOf(entity: string): string {
  const [kind, value] = [
    entity.slice(0, entity.indexOf(':')),
    entity.slice(entity.indexOf(':') + 1),
  ]

  if (kind === 'cnpj' && /^\d{14}$/.test(value)) {
    return value.replace(/^(\d{2})(\d{3})(\d{3})(\d{4})(\d{2})$/,
      '$1.$2.$3/$4-$5')
  }

  return value
}

/**
 * The name that holds for an entity, or its identifier when nobody named it.
 *
 * @param entity - The entity, as the index answered it.
 * @returns The name to show.
 */
export function nameOf(entity: Entity): string {
  return entity.observed.name?.value ?? identifierOf(entity.entity)
}

/**
 * Words a category of the protocol's fixed list.
 *
 * @param category - Such as `food-and-drink`.
 * @returns Such as "Food and drink".
 */
export function categoryOf(category: string): string {
  const words = category.replace(/-/g, ' ')
  return words.charAt(0).toUpperCase() + words.slice(1)
}

/**
 * A date, for a reader.
 *
 * @param iso - An RFC 3339 time.
 * @returns The day, such as "23 Nov 2026".
 */
export function dateOf(iso: string): string {
  return new Intl.DateTimeFormat(dateLocale(), DATE).format(new Date(iso))
}

/**
 * A moment, to the second, for inspection.
 *
 * @param iso - An RFC 3339 time.
 * @returns The date and time, with the zone.
 */
export function momentOf(iso: string): string {
  return new Intl.DateTimeFormat(dateLocale(), DATE_TIME).format(new Date(iso))
}

/**
 * Shortens an id for a line of metadata, keeping its start and end.
 *
 * @param id - An eventId or a did:key.
 * @returns The id, shortened.
 */
export function shortOf(id: string): string {
  return id.length > 24 ? `${id.slice(0, 14)}…${id.slice(-6)}` : id
}
