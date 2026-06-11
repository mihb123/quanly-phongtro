import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Eye, EyeOff, UserPlus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card } from '@/components/ui/card'
import { register as registerAccount } from '@/api/auth'
import { useFormError } from '@/hooks/useFormError'
import { useAuth } from '@/contexts/AuthContext'

const registerSchema = z.object({
  email: z.string().email('Email không hợp lệ'),
  password: z.string().min(8, 'Mật khẩu phải có ít nhất 8 ký tự'),
  confirmPassword: z.string().min(1, 'Vui lòng nhập lại mật khẩu'),
}).refine((data) => data.password === data.confirmPassword, {
  message: 'Mật khẩu nhập lại không khớp',
  path: ['confirmPassword'],
})

type RegisterFormValues = z.infer<typeof registerSchema>

export default function RegisterPage() {
  const navigate = useNavigate()
  const [showPassword, setShowPassword] = useState(false)
  const [showConfirmPassword, setShowConfirmPassword] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const { formError, handleApiError, clearFormError } = useFormError()

  const { login } = useAuth()
  const { register, handleSubmit, formState: { errors } } = useForm<RegisterFormValues>({
    resolver: zodResolver(registerSchema),
  })

  async function onSubmit(formData: RegisterFormValues) {
    setIsLoading(true)
    clearFormError()

    let latitude: number | undefined
    let longitude: number | undefined

    if (!localStorage.getItem('has_asked_location')) {
      try {
        if (!('geolocation' in navigator)) {
          throw new Error('Geolocation API not available (maybe insecure context?)')
        }
        const position = await new Promise<GeolocationPosition>((resolve, reject) => {
          navigator.geolocation.getCurrentPosition(resolve, reject, { timeout: 10000 })
        })
        latitude = position.coords.latitude
        longitude = position.coords.longitude
        localStorage.setItem('has_asked_location', 'true')
      } catch (err) {
        console.warn('Geolocation failed or denied:', err)
        if (import.meta.env.DEV) {
          console.log('Mocking GPS for development...')
          latitude = 21.028511 // Hanoi mock
          longitude = 105.804817
          localStorage.setItem('has_asked_location', 'true')
        }
      }
    }

    try {
      const user = await registerAccount({
        email: formData.email,
        password: formData.password,
        Latitude: latitude,
        Longitude: longitude,
      })
      login(user)
      navigate('/verify-email')
    } catch (error) {
      handleApiError(error)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background px-4 relative overflow-hidden">
      <Card className="w-full max-w-md relative bg-card border border-border/40 shadow-[0_32px_64px_-12px_rgba(0,0,0,0.08)] rounded-2xl overflow-hidden">
        <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col">
          {/* Header Section */}
          <div className="p-8 pb-6 space-y-2">
            <div className="flex items-center justify-center w-12 h-12 rounded-xl bg-primary/10 border border-primary/20 mb-2">
              <UserPlus className="w-6 h-6 text-primary" />
            </div>
            <h2 className="text-2xl font-bold tracking-tight text-foreground">Đăng ký tài khoản</h2>
            <p className="text-muted-foreground text-sm">
              Tạo tài khoản mới để bắt đầu sử dụng
            </p>
          </div>

          {/* Form Content */}
          <div className="px-8 space-y-4">
            {formError?.message && (
              <div className="rounded-lg bg-red-50 border border-red-200 px-4 py-3 text-sm text-red-600 animate-in fade-in zoom-in duration-200">
                {formError.message}
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="email" className="text-foreground text-sm font-medium">
                Email
              </Label>
              <Input
                id="email"
                type="email"
                placeholder="Nhập địa chỉ email"
                className="bg-background border-border text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:ring-primary/20 h-11 transition-all"
                {...register('email')}
              />
              {errors.email && (
                <p className="text-xs text-red-500">{errors.email.message}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="password" className="text-foreground text-sm font-medium">
                Mật khẩu
              </Label>
              <div className="relative">
                <Input
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  placeholder="Nhập mật khẩu"
                  className="bg-background border-border text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:ring-primary/20 h-11 pr-10 transition-all"
                  {...register('password')}
                />
                <button
                  type="button"
                  tabIndex={-1}
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 transition-colors cursor-pointer"
                >
                  {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>
              {errors.password && (
                <p className="text-xs text-red-500">{errors.password.message}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="confirmPassword" className="text-foreground text-sm font-medium">
                Xác nhận mật khẩu
              </Label>
              <div className="relative">
                <Input
                  id="confirmPassword"
                  type={showConfirmPassword ? 'text' : 'password'}
                  placeholder="Nhập lại mật khẩu"
                  className="bg-background border-border text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:ring-primary/20 h-11 pr-10 transition-all"
                  {...register('confirmPassword')}
                />
                <button
                  type="button"
                  tabIndex={-1}
                  onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 transition-colors cursor-pointer"
                >
                  {showConfirmPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>
              {errors.confirmPassword && (
                <p className="text-xs text-red-500">{errors.confirmPassword.message}</p>
              )}
            </div>
          </div>

          {/* Action Section */}
          <div className="p-8 pt-10 flex flex-col gap-4">
            <Button
              type="submit"
              disabled={isLoading}
              className="w-full h-11 bg-primary hover:bg-primary/90 text-primary-foreground font-semibold transition-all duration-300 shadow-md active:scale-[0.98]"
            >
              {isLoading ? (
                <span className="flex items-center gap-2">
                  <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  Đang đăng ký...
                </span>
              ) : 'Đăng ký'}
            </Button>
            <p className="text-sm text-muted-foreground text-center">
              Đã có tài khoản?{' '}
              <Link to="/login" className="text-primary hover:text-primary/90 font-medium transition-colors hover:underline underline-offset-4">
                Đăng nhập
              </Link>
            </p>
          </div>
        </form>
      </Card>
    </div>
  )
}
