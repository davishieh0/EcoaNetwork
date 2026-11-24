/**
 * The display names of the identities on a page, read in one call.
 */

// --- IMPORTS ---
import { useCallback, useMemo } from 'react'
import { useRead } from '@/app/reads/useRead'
import { getProfiles } from '@/logic/datasource/profiles'
import type { Profiles } from '@/logic/datasource/profiles.t'
import { readEnvelope, textOf } from '@/logic/protocol/signed'

// --- GLOBALS ---
// the most the index answers in one call
const MOST = 100

// --- CODE ---
/**
 * Reads the profiles of some identities, a hundred at a time.
 *
 * @param ids - The did:key, in any order, repeats allowed.
 * @returns Every profile found.
 */
async function profilesOf(
  ids: string[],
  moderators?: string,
): Promise<Profiles> {
  const chunks: string[][] = []
  for (let start = 0; start < ids.length; start += MOST) {
    chunks.push(ids.slice(start, start + MOST))
  }

  const answers = await Promise.all(
    chunks.map((chunk) => getProfiles(chunk, moderators)),
  )
  return { items: answers.flatMap((answer) => answer.items) }
}

/**
 * Maps each identity to the name its profile gives. Nothing verifies a
 * display name: the screens show it beside the did:key, never instead.
 * A profile a moderator labeled gives no name.
 *
 * @param ids - The did:key of the authors on the page.
 * @param moderators - The reader's moderators, as reads send them.
 * @returns The names found, by did:key; a failure leaves it empty.
 */
export function useAuthors(
  ids: string[],
  moderators?: string,
): Record<string, string> {
  // the same set of authors, in any order, is the same read
  const key = [...new Set(ids)].sort().join(',')

  const read = useCallback(
    () =>
      key
        ? profilesOf(key.split(','), moderators)
        : Promise.resolve({ items: [] }),
    [key, moderators],
  )
  const { data } = useRead(read)

  return useMemo(() => {
    const names: Record<string, string> = {}

    for (const item of data?.items ?? []) {
      if (item.observed.labels?.length) continue

      const envelope = readEnvelope(item.envelope)
      const name = textOf(envelope, 'displayName')
      if (!name) continue

      names[envelope.author] = name
      names[item.derived.identity] = name
    }

    return names
  }, [data])
}
