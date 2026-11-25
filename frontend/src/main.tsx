/**
 * Application entry point. Mounts the router.
 */

// --- IMPORTS ---
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from 'react-router'
import '@/i18n/i18n'
import { router } from '@/router'
import './styles/global.css'

// --- CODE ---
createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <RouterProvider router={router} />
  </StrictMode>,
)
