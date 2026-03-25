import { createBrowserRouter } from 'react-router-dom'
import LoginPage from '@/pages/Login'
import RegisterPage from '@/pages/Register'
import HomePage from '@/pages/Home'

const router = createBrowserRouter([
  {
    path: '/',
    element: <HomePage />,
  },
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
