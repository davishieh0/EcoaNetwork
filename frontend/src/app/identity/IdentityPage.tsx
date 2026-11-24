/**
 * The identity this browser signs with: created, restored, and what it is
 * backed up as.
 */

// --- IMPORTS ---
import { motion } from 'motion/react'
import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router'
import { IdentitySetup } from '@/app/identity/IdentitySetup'
import { dateOf } from '@/app/reads/words'
import { Badge } from '@/components/ui/Badge'
import { Container } from '@/components/ui/Container'
import { cn } from '@/components/ui/cn'
import { useRise } from '@/components/ui/useRise'
import { identityInfo, knownIdentities } from '@/logic/protocol/identity'
import type { IdentityInfo } from '@/logic/protocol/identity.t'
import { ROUTES } from '@/routes'

// --- GLOBALS ---
const NEXT = [
  {
    to: ROUTES.claims,
    title: 'Verification',
    body: 'Prove you are human, or that you speak for a website.',
  },
  {
    to: ROUTES.account,
    title: 'Public profile',
    body: 'The display name readers see beside your key.',
  },
  {
    to: ROUTES.moderation,
    title: 'Moderation',
    body: 'Whose labels you follow, and labeling yourself.',
  },
]

// --- CODE ---
/**
 * Renders the identity screen.
 *
 * @returns The screen.
 */
export function IdentityPage() {
  const rise = useRise()
  const [info, setInfo] = useState<IdentityInfo | null | undefined>(undefined)
  const [fresh, setFresh] = useState(0)

  const read = useCallback(() => {
    identityInfo().then(setInfo, () => setInfo(null))
    knownIdentities().then(
      (list) => setFresh(list.filter((i) => i.kind === 'fresh').length),
      () => setFresh(0),
    )
  }, [])

  useEffect(read, [read])

  return (
    <Container className="max-w-3xl pt-28 pb-16">
      <motion.h1 {...rise()} className="font-display text-h1">
        Identity
      </motion.h1>

      {info === undefined ? null : info === null ? (
        <motion.div {...rise(0.08)} className="mt-8">
          <IdentitySetup onDone={read} />
        </motion.div>
      ) : (
        <>
          <motion.section
            {...rise(0.08)}
            className={cn(
              'mt-8 flex flex-col gap-3 border border-structural bg-surface',
              'p-6',
            )}
          >
            <p className="meta text-muted">This browser signs as</p>
            <p className="font-mono text-mono-s break-all">{info.did}</p>
            <div className="flex flex-wrap items-center gap-2">
              {info.backedUp ? (
                <Badge tone="valid" glyph="◆">
                  Backed up
                </Badge>
              ) : (
                <Badge tone="warning" glyph="●">
                  No backup: lost with this browser’s data
                </Badge>
              )}
              {info.createdAt ? (
                <span className="meta text-muted">
                  since {dateOf(info.createdAt)}
                </span>
              ) : null}
              {fresh > 0 ? (
                <span className="meta text-muted">
                  · {fresh} fresh {fresh === 1 ? 'identity' : 'identities'}{' '}
                  for single reviews, never backed up
                </span>
              ) : null}
            </div>
            <p className="text-body-s text-secondary">
              The private key cannot be read out of this browser, even by
              this page. Another browser takes it only from the backup file
              and its password.
            </p>
          </motion.section>

          <motion.ul {...rise(0.14)} className="mt-6 grid gap-3 sm:grid-cols-3">
            {NEXT.map((item) => (
              <li key={item.to}>
                <Link
                  to={item.to}
                  className={cn(
                    'flex h-full flex-col gap-2 border border-structural p-5',
                    'transition-colors hover:border-strong',
                  )}
                >
                  <span className="font-display text-h4">{item.title}</span>
                  <span className="text-body-s text-secondary">
                    {item.body}
                  </span>
                </Link>
              </li>
            ))}
          </motion.ul>
        </>
      )}
    </Container>
  )
}
