import { createBrowserRouter } from 'react-router-dom'
import LoginPage from '@/pages/Login'
import RegisterPage from '@/pages/Register'
import HomePage from '@/pages/Home'
import ProtectedRoute from '@/components/ProtectedRoute'

import VerifyEmailPage from '@/pages/VerifyEmail'

const router = createBrowserRouter([
  {
    path: '/',
    element: (
      <ProtectedRoute>
        <HomePage />
      </ProtectedRoute>
    ),
  },
  {
    path: '/login',
    element: <LoginPage />,
  },
  {
    path: '/register',
    element: <RegisterPage />,
  },
  {
    path: '/verify-email',
    element: (
      <ProtectedRoute>
        <VerifyEmailPage />
      </ProtectedRoute>
    ),
  },
])

export default router
