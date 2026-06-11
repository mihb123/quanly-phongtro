import { RouterProvider } from 'react-router-dom'
import router from '@/router'

import { AuthProvider } from '@/contexts/AuthContext'
import { Toaster } from 'sonner'

export default function App() {
  return (
    <AuthProvider>
      <RouterProvider router={router} />
      <Toaster richColors position="top-right" />
    </AuthProvider>
  )
}
