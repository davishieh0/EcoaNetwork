/**
 * Who is reading: whether they are signed in, and the identity this
 * browser signs with.
 */

// --- IMPORTS ---
import { useCallback, useEffect, useState } from 'react'
import { knownIdentities } from '@/logic/protocol/identity'
import { useSession } from '@/logic/session/session'

// --- CODE ---
/**
 * Reads the session and this browser's did:key, never making a key.
 *
 * @returns Whether they can act, their did:key once there is one, and
 * known, to call after publishing made one.
 */
export function useMe() {
  const session = useSession()
  const [dids, setDids] = useState<{ main: string | null; all: string[] }>({
    main: null,
    all: [],
  })

  const known = useCallback(() => {
    knownIdentities().then(
      (list) =>
        setDids({
          main: list.find((i) => i.kind === 'main')?.did ?? null,
          all: list.map((i) => i.did),
        }),
      () => setDids({ main: null, all: [] }),
    )
  }, [])

  useEffect(known, [known])

  return {
    // acting asks for a complete account, as writing a review does
    canAct:
      session.status === 'signed-in' && Boolean(session.account.handle),
    sessionKnown: session.status !== 'loading',
    // the main identity, and every one this browser signs as
    did: dids.main,
    dids: dids.all,
    known,
  }
}
