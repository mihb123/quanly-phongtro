import { createBrowserRouter } from 'react-router-dom'
import LoginPage from '@/pages/Login'
import RegisterPage from '@/pages/Register'

const router = createBrowserRouter([
  {
    path: '/login',
    element: <LoginPage />,
  },
  {
    path: '/register',
    element: <RegisterPage />,
  },
])

export default router
