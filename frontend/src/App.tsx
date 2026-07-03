import { RouterProvider } from 'react-router-dom'
import router from '@/router'

import { AuthProvider } from '@/contexts/AuthContext'
import { ThemeApplier } from '@/components/ThemeApplier'
import { Toaster } from 'sonner'

export default function App() {
  return (
    <AuthProvider>
      <ThemeApplier />
      <RouterProvider router={router} />
      <Toaster richColors position="top-right" />
    </AuthProvider>
  )
}
