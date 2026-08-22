import { createBrowserRouter } from 'react-router-dom'
import LoginPage from '@/pages/Login'
import RegisterPage from '@/pages/Register'
import HomePage from '@/pages/Home'
import ProtectedRoute from '@/components/ProtectedRoute'

import VerifyEmailPage from '@/pages/VerifyEmail'
import OptimizeImagePage from '@/pages/OptimizeImage'

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
    // Tiện ích công khai, không cần đăng nhập.
    path: '/optimize-img',
    element: <OptimizeImagePage />,
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
