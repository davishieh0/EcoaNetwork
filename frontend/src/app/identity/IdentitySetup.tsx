/**
 * Making this browser's identity: created with an encrypted backup, or
 * restored from one. The private key is never sent anywhere.
 */

// --- IMPORTS ---
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { IdentitySetupProps } from '@/app/identity/IdentitySetup.t'
import { Button } from '@/components/ui/Button'
import { FormAlert } from '@/components/ui/FormAlert'
import { cn } from '@/components/ui/cn'
import { BackupError } from '@/logic/protocol/backup'
import { createIdentity, restoreIdentity } from '@/logic/protocol/identity'

// --- GLOBALS ---
// short passwords fall to a guess even at 600 000 rounds per try
const MIN_PASSWORD = 10

const FIELD = cn(
  'w-full border border-structural bg-canvas p-3',
  'transition-colors duration-interface focus:border-strong',
)

// --- CODE ---
/**
 * Hands the backup file to the browser to save.
 *
 * @param text - The file's text.
 * @param did - The identity, for the file's name.
 * @returns Nothing.
 */
function download(text: string, did: string) {
  const blob = new Blob([text], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')

  link.href = url
  link.download = `ecoa-identity-${did.slice(-12)}.json`
  document.body.append(link)
  link.click()
  link.remove()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

/**
 * Renders the choice to create an identity or restore one.
 *
 * @param props - What to do once there is one.
 * @returns The setup.
 */
export function IdentitySetup({ onDone }: IdentitySetupProps) {
  const { t } = useTranslation()
  const [tab, setTab] = useState<'create' | 'restore'>('create')
  const [password, setPassword] = useState('')
  const [again, setAgain] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const weak = password.length < MIN_PASSWORD
  const mismatch = again !== password

  const create = async (withBackup: boolean) => {
    setBusy(true)
    setError(null)
    try {
      const { identity, backup } = await createIdentity(
        withBackup ? password : null,
      )
      if (backup) download(backup, identity.did)
      onDone(identity.did)
    } catch (failure) {
      setError(
        failure instanceof Error
          ? failure.message
          : 'This browser could not make a key.',
      )
    } finally {
      setBusy(false)
    }
  }

  const restore = async () => {
    if (!file) return
    setBusy(true)
    setError(null)
    try {
      const identity = await restoreIdentity(await file.text(), password)
      onDone(identity.did)
    } catch (failure) {
      setError(
        failure instanceof BackupError
          ? failure.message
          : 'This browser could not restore the identity.',
      )
    } finally {
      setBusy(false)
    }
  }

  return (
    <section
      aria-label={t('Your identity')}
      className="flex flex-col gap-5 border border-structural bg-surface p-6"
    >
      <div className="flex flex-col gap-1">
        <h2 className="font-display text-h3">{t('Your identity')}</h2>
        <p className="text-body-s text-secondary">
          {t(
            'Everything you publish is signed by a key made in this ' +
              'browser. It never leaves it: not to Ecoa, not to anyone. ' +
              'Keep a backup, or the identity ends with this browser’s data.',
          )}
        </p>
      </div>

      <div role="tablist" className="flex gap-2">
        {(['create', 'restore'] as const).map((name) => (
          <button
            key={name}
            type="button"
            role="tab"
            aria-selected={tab === name}
            onClick={() => {
              setTab(name)
              setError(null)
            }}
            className={cn(
              'meta cursor-pointer border px-3 py-2 transition-colors',
              tab === name
                ? 'border-signal bg-signal text-on-signal'
                : 'border-structural text-secondary hover:text-primary',
            )}
          >
            {name === 'create' ? t('Create') : t('Restore from a backup')}
          </button>
        ))}
      </div>

      {tab === 'create' ? (
        <form
          className="flex flex-col gap-4"
          onSubmit={(event) => {
            event.preventDefault()
            if (!weak && !mismatch) void create(true)
          }}
        >
          <label className="flex flex-col gap-1">
            <span className="meta text-muted">{t('Backup password')}</span>
            <input
              type="password"
              value={password}
              autoComplete="new-password"
              onChange={(event) => setPassword(event.target.value)}
              className={FIELD}
            />
            {password && weak ? (
              <span className="text-body-s text-danger">
                {t('At least {{n}} characters.', { n: MIN_PASSWORD })}
              </span>
            ) : null}
          </label>
          <label className="flex flex-col gap-1">
            <span className="meta text-muted">
              {t('The same password again')}
            </span>
            <input
              type="password"
              value={again}
              autoComplete="new-password"
              onChange={(event) => setAgain(event.target.value)}
              className={FIELD}
            />
            {again && mismatch ? (
              <span className="text-body-s text-danger">
                {t('The two passwords differ.')}
              </span>
            ) : null}
          </label>
          <p className="text-body-s text-muted">
            {t(
              'The backup is a file sealed with this password. Nobody can ' +
                'open it without the password, Ecoa included, and nobody can ' +
                'reset it.',
            )}
          </p>
          <FormAlert>{error}</FormAlert>
          <Button type="submit" pending={busy} disabled={weak || mismatch}>
            {busy ? t('Making the key') : t('Create and download the backup')}
          </Button>
          <button
            type="button"
            disabled={busy}
            onClick={() => void create(false)}
            className={cn(
              'meta cursor-pointer self-start text-muted',
              'hover:text-primary',
            )}
          >
            {t('Create without a backup — lost with this browser’s data')}
          </button>
        </form>
      ) : (
        <form
          className="flex flex-col gap-4"
          onSubmit={(event) => {
            event.preventDefault()
            void restore()
          }}
        >
          <label className="flex flex-col gap-1">
            <span className="meta text-muted">{t('Backup file')}</span>
            <input
              type="file"
              accept="application/json,.json"
              onChange={(event) => setFile(event.target.files?.[0] ?? null)}
              className={FIELD}
            />
          </label>
          <label className="flex flex-col gap-1">
            <span className="meta text-muted">{t('Its password')}</span>
            <input
              type="password"
              value={password}
              autoComplete="current-password"
              onChange={(event) => setPassword(event.target.value)}
              className={FIELD}
            />
          </label>
          <FormAlert>{error}</FormAlert>
          <Button type="submit" pending={busy} disabled={!file || !password}>
            {busy ? t('Opening the backup') : t('Restore the identity')}
          </Button>
        </form>
      )}
    </section>
  )
}
