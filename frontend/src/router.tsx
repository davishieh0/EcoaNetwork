/**
 * The route tree for the site and the platform.
 */

// --- IMPORTS ---
import { createBrowserRouter } from 'react-router'

import { Account } from '@/app/account/Account'
import { ChooseHandle } from '@/app/auth/chooseHandle/ChooseHandle'
import { Dashboard } from '@/app/dashboard/Dashboard'
import { ForgotPassword } from '@/app/auth/forgotPassword/ForgotPassword'
import { Login } from '@/app/auth/login/Login'
import { Register } from '@/app/auth/register/Register'
import { ResetPassword } from '@/app/auth/resetPassword/ResetPassword'
import { VerifyEmail } from '@/app/auth/verifyEmail/VerifyEmail'
import { AppLayout } from '@/app/layout/AppLayout'
import { GuestOnly } from '@/app/layout/GuestOnly'
import { RequireAuth } from '@/app/layout/RequireAuth'
import { Explore } from '@/app/reviews/Explore'
import { Write } from '@/app/write/Write'
import { Landing } from '@/site/landing/Landing'
import { NotFound } from '@/site/NotFound'
import { SiteLayout } from '@/site/layout/SiteLayout'

// --- GLOBALS ---
export const router = createBrowserRouter([
  // the site is for visitors: someone signed in goes to the dashboard
  {
    element: <GuestOnly />,
    children: [
      {
        element: <SiteLayout />,
        children: [{ index: true, element: <Landing /> }],
      },
    ],
  },
  {
    path: 'app',
    element: <AppLayout />,
    children: [
      { path: 'explore', element: <Explore /> },
      // for someone not signed in yet: signing in, or recovering an account
      {
        element: <GuestOnly />,
        children: [
          { path: 'login', element: <Login /> },
          { path: 'register', element: <Register /> },
          { path: 'verify', element: <VerifyEmail /> },
          { path: 'forgot-password', element: <ForgotPassword /> },
          { path: 'reset-password', element: <ResetPassword /> },
        ],
      },

      // signed in, before the handle is chosen
      {
        element: <RequireAuth profile={false} />,
        children: [{ path: 'welcome', element: <ChooseHandle /> }],
      },

      // signed in, with a complete account
      {
        element: <RequireAuth profile />,
        children: [
          { index: true, element: <Dashboard /> },
          { path: 'account', element: <Account /> },
          { path: 'write', element: <Write /> },
        ],
      },
    ],
  },
  {
    element: <SiteLayout nav={false} />,
    children: [{ path: '*', element: <NotFound /> }],
  },
])
