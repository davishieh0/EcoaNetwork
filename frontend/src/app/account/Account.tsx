/**
 * The signed-in account: who it is, signing out, and deleting it.
 */

// --- IMPORTS ---
import { motion } from 'motion/react'

import { DeleteAccount } from '@/app/account/DeleteAccount'
import { PublicProfile } from '@/app/account/PublicProfile'
import { useAccountData } from '@/app/account/useAccountData'
import { Button } from '@/components/ui/Button'
import { Container } from '@/components/ui/Container'
import { FormAlert } from '@/components/ui/FormAlert'
import { cn } from '@/components/ui/cn'
import { useRise } from '@/components/ui/useRise'

// --- CODE ---
/**
 * Builds the account screen.
 *
 * @returns The account screen, or nothing until the account is known.
 */
export function Account() {
  const { account, signOut, remove } = useAccountData()
  const rise = useRise()

  // RequireAuth only renders this with a complete account
  if (!account?.handle) return null

  const details = [
    { label: 'Handle', value: `@${account.handle}` },
    { label: 'Email', value: account.email },
    { label: 'Account id', value: account.sub, mono: true },
  ]

  return (
    <section className="flex min-h-svh items-center">
      <Container className="flex justify-center py-32">
        <div className="flex w-full max-w-md flex-col gap-10">
          <header className="flex flex-col gap-3">
            <motion.p {...rise()} className="meta text-muted">
              Account
            </motion.p>

            <motion.h1
              {...rise(0.06)}
              className={cn(
                'font-display text-h1 font-semibold tracking-display',
                'break-all',
              )}
            >
              @{account.handle}
            </motion.h1>
          </header>

          <motion.dl
            {...rise(0.12)}
            className="flex flex-col border-t border-line"
          >
            {details.map((detail) => (
              <div
                key={detail.label}
                className={cn(
                  'flex flex-col gap-1 border-b border-line py-4',
                  'sm:flex-row sm:items-baseline sm:justify-between',
                  'sm:gap-6',
                )}
              >
                <dt className="meta shrink-0 text-muted">{detail.label}</dt>
                <dd
                  className={cn(
                    'min-w-0 break-all text-primary sm:text-right',
                    detail.mono ? 'font-mono text-mono-m' : 'text-body-m',
                  )}
                >
                  {detail.value}
                </dd>
              </div>
            ))}
          </motion.dl>
          <motion.div {...rise(0.15)}>
            <PublicProfile handle={account.handle} />
          </motion.div>

          <motion.div {...rise(0.18)} className="flex flex-col gap-4">
            <FormAlert>{signOut.message}</FormAlert>

            <Button
              size="lg"
              variant="secondary"
              pending={signOut.pending}
              onClick={() => void signOut.run()}
            >
              {signOut.pending ? 'Signing out' : 'Sign out'}
            </Button>
          </motion.div>

          <motion.div
            {...rise(0.26)}
            className="flex flex-col gap-4 border-t border-line pt-8"
          >
            <div className="flex flex-col gap-1">
              <h2 className="meta text-danger">Delete account</h2>
              <p className="text-body-s text-secondary text-pretty">
                Removes the account, frees the handle and ends every session.
                There is no undo.
              </p>
            </div>

            <div>
              <DeleteAccount handle={account.handle} remove={remove} />
            </div>
          </motion.div>
        </div>
      </Container>
    </section>
  )
}
