# Unconfirmed accounts

**Status: known gap, not fixed.** To be solved later with a scheduled task.

## The problem

`/register` reserves the handle the moment the account is created: it writes
the `users` row right after Cognito's `SignUp`, before the email is
confirmed. An account that never confirms keeps that row, and the handle,
forever.

Two effects:

- **Squatting.** Anyone can hold any free handle by registering with an
  address they never confirm.
- **Retrying hits your own handle.** Someone who registers, loses the code and
  registers again with the same handle gets `handle_taken_error`, taken by
  their own first attempt. The right path is to sign in, get
  `email_not_verified_error` and confirm, but only if the frontend leads them
  there.

## The fix

A periodic task that removes accounts still unconfirmed after a grace period
(a few days):

1. List the pool's users with status `UNCONFIRMED` created before the cutoff.
   This is an admin operation (`ListUsers` with a filter), so the task needs
   AWS credentials with permission for it; the backend's own Cognito client
   is unsigned and cannot do it.
2. For each one, delete the `users` row by `sub`, then the Cognito user
   (`AdminDeleteUser`). The row goes first: if the second step fails, the
   account is left without a profile, a state the next run cleans up again.

## Until then

- The frontend should send a user who registered but did not confirm to the
  sign-in screen, where `email_not_verified_error` leads to confirming, rather
  than back to registering.
- In dev, a stuck handle can be freed by hand: delete the user in the Cognito
  console and the row in `users`.
