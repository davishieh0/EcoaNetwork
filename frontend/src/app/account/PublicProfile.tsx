/**
 * The public profile: the display name and bio others see beside this
 * browser's did:key. Signed, and verified by nothing: readers see it beside
 * the key, never instead of it.
 */

// --- IMPORTS ---
import { useEffect, useState } from 'react'
import { Link } from 'react-router'
import type { PublicProfileProps } from '@/app/account/PublicProfile.t'
import { usePublicProfile } from '@/app/account/usePublicProfile'
import { Button } from '@/components/ui/Button'
import { FormAlert } from '@/components/ui/FormAlert'
import { cn } from '@/components/ui/cn'
import { BIO_MAX, NAME_MAX, checkProfile } from '@/logic/protocol/profile'
import { characters } from '@/logic/protocol/review'
import { ROUTES } from '@/routes'

// --- GLOBALS ---
const FIELD = cn(
  'w-full border border-structural bg-surface p-3',
  'transition-colors duration-interface focus:border-strong',
)

// --- CODE ---
/**
 * Renders the profile and the form that changes it.
 *
 * @param props - The account's handle, offered as a display name.
 * @returns The section.
 */
export function PublicProfile({ handle }: PublicProfileProps) {
  const { did, current, loading, busy, error, notice, publish } =
    usePublicProfile()
  const [name, setName] = useState('')
  const [bio, setBio] = useState('')
  const [tried, setTried] = useState(false)
  const [filled, setFilled] = useState(false)

  // start from what is published, or the handle when nothing is
  useEffect(() => {
    if (loading || filled) return
    const timer = setTimeout(() => {
      setName(current?.displayName ?? handle)
      setBio(current?.bio ?? '')
      setFilled(true)
    }, 0)
    return () => clearTimeout(timer)
  }, [loading, filled, current, handle])

  const checked = checkProfile({ displayName: name, bio })
  const problems = tried && !checked.ok ? checked.problems : {}

  return (
    <section className="flex flex-col gap-4 border-t border-line pt-6">
      <div className="flex flex-col gap-1">
        <h2 className="font-display text-h4">Public profile</h2>
        <p className="text-body-s text-secondary">
          Signed by your identity and public. Nothing verifies a display
          name, so readers always see it beside your key.
        </p>
      </div>

      {!did ? (
        <p className="text-body-s text-secondary">
          This browser has no identity yet.{' '}
          <Link to={ROUTES.identity} className="border-b border-signal">
            Create it
          </Link>{' '}
          first.
        </p>
      ) : (
        <form
          className="flex flex-col gap-4"
          onSubmit={(event) => {
            event.preventDefault()
            setTried(true)
            if (checked.ok) void publish(checked.payload)
          }}
        >
          <p className="font-mono text-mono-s break-all text-muted">{did}</p>
          <label className="flex flex-col gap-1">
            <span className="meta text-muted">
              Display name · {characters(name)}/{NAME_MAX}
            </span>
            <input
              value={name}
              onChange={(event) => setName(event.target.value)}
              className={FIELD}
            />
            {problems.displayName ? (
              <span className="text-body-s text-danger">
                {problems.displayName}
              </span>
            ) : null}
          </label>
          <label className="flex flex-col gap-1">
            <span className="meta text-muted">
              Bio · {characters(bio)}/{BIO_MAX}
            </span>
            <textarea
              value={bio}
              onChange={(event) => setBio(event.target.value)}
              rows={3}
              className={FIELD}
            />
            {problems.bio ? (
              <span className="text-body-s text-danger">{problems.bio}</span>
            ) : null}
          </label>
          <FormAlert>{error}</FormAlert>
          {notice ? (
            <FormAlert tone="valid">{notice}</FormAlert>
          ) : null}
          <Button type="submit" variant="secondary" pending={busy}>
            {busy ? 'Signing and sending' : 'Publish the profile'}
          </Button>
        </form>
      )}
    </section>
  )
}
