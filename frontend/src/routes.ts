/**
 * Every destination the frontend links to.
 */

// --- GLOBALS ---
// the platform lives under /app
const APP = '/app'

export const ROUTES = {
  home: '/',
  dashboard: APP,
  explore: `${APP}/explore`,
  write: `${APP}/write`,
  login: `${APP}/login`,
  register: `${APP}/register`,
  verify: `${APP}/verify`,
  forgotPassword: `${APP}/forgot-password`,
  resetPassword: `${APP}/reset-password`,
  welcome: `${APP}/welcome`,
  account: `${APP}/account`,
  claims: `${APP}/claims`,
  identity: `${APP}/identity`,
  moderation: `${APP}/moderation`,
  entity: (entity: string) => `${APP}/entities/${encodeURIComponent(entity)}`,
  review: (eventId: string) => `${APP}/reviews/${eventId}`,
} as const
