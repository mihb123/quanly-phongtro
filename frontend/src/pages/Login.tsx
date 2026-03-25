import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Eye, EyeOff, LogIn } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card } from '@/components/ui/card'
import { authApi } from '@/api/auth'
import { useFormError } from '@/hooks/useFormError'

const loginSchema = z.object({
  email: z.string().email('Địa chỉ email không hợp lệ'),
  password: z.string().min(8, 'Mật khẩu phải có ít nhất 8 ký tự'),
})

type LoginFormValues = z.infer<typeof loginSchema>

export default function LoginPage() {
  const navigate = useNavigate()
  const [showPassword, setShowPassword] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const { formError, handleApiError, clearFormError } = useFormError()

  const { register, handleSubmit, formState: { errors } } = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
  })

  async function onSubmit(data: LoginFormValues) {
    setIsLoading(true)
    clearFormError()
    try {
      const response = await authApi.login({
        email: data.email,
        password: data.password,
      })
      localStorage.setItem('access_token', response.data.access_token)
      localStorage.setItem('refresh_token', response.data.refresh_token)
      navigate('/')
    } catch (error) {
      handleApiError(error)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-900 via-purple-950 to-slate-900 px-4">
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute -top-40 -right-40 w-96 h-96 rounded-full bg-purple-500/20 blur-3xl" />
        <div className="absolute -bottom-40 -left-40 w-96 h-96 rounded-full bg-indigo-500/20 blur-3xl" />
      </div>

      <Card className="w-full max-w-md relative bg-white/5 border-white/10 backdrop-blur-2xl shadow-2xl overflow-hidden border">
        <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col">
          {/* Header Section */}
          <div className="p-8 pb-6 space-y-2">
            <div className="flex items-center justify-center w-12 h-12 rounded-xl bg-purple-500/20 border border-purple-500/30 mb-2">
              <LogIn className="w-6 h-6 text-purple-400" />
            </div>
            <h2 className="text-2xl font-bold text-white">Đăng nhập</h2>
            <p className="text-slate-400 text-sm">
              Nhập email và mật khẩu của bạn để tiếp tục
            </p>
          </div>

          {/* Form Content */}
          <div className="px-8 space-y-4">
            {formError?.message && (
              <div className="rounded-lg bg-red-500/10 border border-red-500/30 px-4 py-3 text-sm text-red-400 animate-in fade-in zoom-in duration-200">
                {formError.message}
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="email" className="text-slate-300 text-sm font-medium">
                Địa chỉ Email
              </Label>
              <Input
                id="email"
                type="email"
                placeholder="vd: user@example.com"
                className="bg-white/5 border-white/10 text-white placeholder:text-slate-500 focus:border-purple-500/50 focus:ring-purple-500/20 h-11 transition-all"
                {...register('email')}
              />
              {errors.email && (
                <p className="text-xs text-red-400">{errors.email.message}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="password" className="text-slate-300 text-sm font-medium">
                Mật khẩu
              </Label>
              <div className="relative">
                <Input
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  placeholder="Nhập mật khẩu"
                  className="bg-white/5 border-white/10 text-white placeholder:text-slate-500 focus:border-purple-500/50 focus:ring-purple-500/20 h-11 pr-10 transition-all"
                  {...register('password')}
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-200 transition-colors"
                >
                  {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>
              {errors.password && (
                <p className="text-xs text-red-400">{errors.password.message}</p>
              )}
            </div>
          </div>

          {/* Action Section */}
          <div className="p-8 pt-10 flex flex-col gap-4">
            <Button
              type="submit"
              disabled={isLoading}
              className="w-full h-11 bg-purple-600 hover:bg-purple-700 text-white font-semibold transition-all duration-300 shadow-lg shadow-purple-500/25 active:scale-[0.95]"
            >
              {isLoading ? (
                <span className="flex items-center gap-2">
                  <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  Đang đăng nhập...
                </span>
              ) : 'Đăng nhập'}
            </Button>
            <p className="text-sm text-slate-400 text-center">
              Chưa có tài khoản?{' '}
              <Link to="/register" className="text-purple-400 hover:text-purple-300 font-medium transition-colors hover:underline underline-offset-4">
                Đăng ký ngay
              </Link>
            </p>
          </div>
        </form>
      </Card>
    </div>
  )
}
