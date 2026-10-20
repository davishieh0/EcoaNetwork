# Auth: what is left

The routes under `/api/auth` cover registering, confirming the email, signing
in and out, refreshing, resetting the password, completing a profile and
deleting the account. This is what is not built yet.

## Before production

**Rate limiting the routes that send email.** `/register`, `/verify/resend`
and `/forgot-password` each make Cognito send an email, and nothing limits
who calls them. Cognito's own sender (`COGNITO_DEFAULT`) allows about 50
emails a day for the whole pool, so a script can spend the day's quota and
leave every real user without a confirmation or reset code.

- Limit those routes per IP in the backend.
- Send through SES in production (`email_sending_account = "DEVELOPER"` in
  the Cognito module), which lifts the daily cap.

**Checking the database in `/health`.** It answers OK while Postgres is down,
so an orchestrator using it would not notice.

## Features

| Feature | Cognito operation | Notes |
|---|---|---|
| Change the password while signed in | `ChangePassword` | Public, takes the access token and the current password |
| Change the email | `UpdateUserAttributes`, then `VerifyUserAttribute` | The new address must be confirmed with a code before it counts |
| Sign out everywhere | `GlobalSignOut` | Public, takes the access token. Revokes every refresh token of the account; useful after a password change |
| MFA with an authenticator app | `AssociateSoftwareToken`, `VerifySoftwareToken`, `SetUserMFAPreference` | See below |

**MFA needs the sign-in to change too.** The pool already allows optional
TOTP, but nobody can turn it on without these routes. Once someone has, Cognito
answers the password step with a `SOFTWARE_TOKEN_MFA` challenge instead of
tokens. Today `CognitoClient.sign_in` treats any challenge as a provider error
(502), so the sign-in has to learn to ask for the code and answer with
`RespondToAuthChallenge` in the same change that ships the MFA routes.

## Deferred

**Unconfirmed accounts keep their handle forever.** To be cleaned up by a
scheduled task. See [unconfirmed-accounts.md](./unconfirmed-accounts.md).

## Known limits, accepted for now

**A signed-out access token keeps working for up to 60 minutes** on the
routes that only check it locally. `/me` and `/login` ask Cognito, so they
refuse it at once. If that window starts to matter:

- shorten the access token (`access_token_validity_minutes`), or
- keep a list of signed-out sessions: store the token's `origin_jti` at sign
  out until it expires, and refuse any token carrying it. One lookup per
  authenticated request.

**The session cookies need the frontend and the API on the same site**, e.g.
`ecoa.com.br` and `api.ecoa.com.br`, because they are `SameSite=Lax`. On
different sites they would need `SameSite=None` and a CSRF token.
