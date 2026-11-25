/**
 * Types for moving between screens.
 */

// --- IMPORTS ---
import type { ForgotPasswordState } from '@/app/auth/forgotPassword/ForgotPassword.t'
import type { LoginState } from '@/app/auth/login/Login.t'
import type { ResetPasswordState } from '@/app/auth/resetPassword/ResetPassword.t'
import type { VerifyState } from '@/app/auth/verifyEmail/VerifyEmail.t'
import type { WriteState } from '@/app/write/Write.t'

// --- GLOBALS ---
/** What each screen accepts from the one before it, by its ROUTES name. */
export type RouteStates = {
  login: LoginState
  verify: VerifyState
  forgotPassword: ForgotPasswordState
  resetPassword: ResetPasswordState
  write: WriteState
}
