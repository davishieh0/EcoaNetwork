/**
 * The public profile's data: this browser's identity, its current profile,
 * and publishing a new one.
 */

// --- IMPORTS ---
import { useCallback, useEffect, useState } from 'react'
import type { ProfileText } from '@/app/account/PublicProfile.t'
import { failureOf, publishSigned } from '@/app/reads/publish'
import { useMe } from '@/app/reads/useMe'
import { getProfile } from '@/logic/datasource/profiles'
import { HttpError } from '@/logic/errors/httpError'
import { readEnvelope, textOf } from '@/logic/protocol/signed'

// --- GLOBALS ---
// how long to read again for the index to show a profile just published
const FOLLOW = { tries: 10, every: 1000 }

// --- CODE ---
/**
 * Reads the current profile of the identity, null when it has none.
 *
 * @param did - The did:key.
 * @returns What the profile says, or null.
 */
async function profileOf(did: string): Promise<ProfileText | null> {
  try {
    const item = await getProfile(did)
    const envelope = readEnvelope(item.envelope)
    return {
      displayName: textOf(envelope, 'displayName'),
      bio: textOf(envelope, 'bio'),
    }
  } catch (error) {
    if (error instanceof HttpError && error.status === 404) return null
    throw error
  }
}

/**
 * Reads the profile of this browser's identity, and publishes a new one.
 *
 * @returns The did:key, the current profile, and publish.
 */
export function usePublicProfile() {
  const { did, known } = useMe()
  const [current, setCurrent] = useState<ProfileText | null>(null)
  const [loaded, setLoaded] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)

  useEffect(() => {
    if (!did) return
    let live = true

    profileOf(did).then(
      (profile) => {
        if (!live) return
        setCurrent(profile)
        setLoaded(did)
      },
      () => live && setLoaded(did),
    )

    return () => {
      live = false
    }
  }, [did])

  const publish = useCallback(
    async (payload: Record<string, string>) => {
      setBusy(true)
      setError(null)
      setNotice(null)

      try {
        await publishSigned('profile.updated', payload)
        known()
        setNotice('Published. Readers see it beside your key.')

        // read it back once the index has it
        for (let i = 0; i < FOLLOW.tries && did; i++) {
          await new Promise((done) => setTimeout(done, FOLLOW.every))
          const profile = await profileOf(did).catch(() => null)
          if (
            profile &&
            profile.displayName === payload.displayName &&
            profile.bio === payload.bio
          ) {
            setCurrent(profile)
            break
          }
        }
      } catch (failure) {
        setError(failureOf(failure))
      } finally {
        setBusy(false)
      }
    },
    [did, known],
  )

  return {
    did,
    current,
    loading: did === null || loaded !== did,
    busy,
    error,
    notice,
    publish,
  }
}
