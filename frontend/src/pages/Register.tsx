import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Eye, EyeOff, UserPlus, Loader2 } from '@/components/icons'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from '@/components/ui/card'
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

// Trang đăng ký: form email/mật khẩu/xác nhận, lấy vị trí (geolocation) rồi tạo tài khoản
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

    let shouldFetchLocation = false

    if ('geolocation' in navigator) {
      if (!localStorage.getItem('has_asked_location')) {
        shouldFetchLocation = true
      } else if (navigator.permissions && navigator.permissions.query) {
        try {
          const permission = await navigator.permissions.query({ name: 'geolocation' })
          if (permission.state === 'granted') {
            shouldFetchLocation = true
          }
        } catch (e) {
          console.warn('Could not query geolocation permission:', e)
        }
      }
    }

    if (shouldFetchLocation) {
      try {
        const position = await new Promise<GeolocationPosition>((resolve, reject) => {
          navigator.geolocation.getCurrentPosition(resolve, reject, { timeout: 10000 })
        })
        latitude = position.coords.latitude
        longitude = position.coords.longitude
        localStorage.setItem('has_asked_location', 'true')
      } catch (err) {
        console.error('CRITICAL: Geolocation failed or denied even though permission was granted. Error:', err)
        localStorage.setItem('has_asked_location', 'true')
        if (import.meta.env.DEV) {
          console.log('Mocking GPS for development...')
          latitude = 21.028511 // Hanoi mock
          longitude = 105.804817
        } else {
          console.error('Not in DEV mode, latitude and longitude will be undefined and sent as null to DB.')
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
    <div className="min-h-screen flex items-center justify-center bg-background px-4">
      <Card className="w-full max-w-md">
        <form onSubmit={handleSubmit(onSubmit)} className="contents">
          <CardHeader>
            <div className="flex items-center justify-center size-12 rounded-lg bg-primary/10 border border-primary/20 mb-2">
              <UserPlus className="size-6 text-primary" />
            </div>
            <CardTitle className="text-2xl font-semibold tracking-tight">Đăng ký tài khoản</CardTitle>
            <CardDescription>
              Tạo tài khoản mới để bắt đầu sử dụng
            </CardDescription>
          </CardHeader>

          <CardContent className="space-y-4">
            {formError?.message && (
              <div className="rounded-lg bg-destructive/10 border border-destructive/20 px-4 py-3 text-sm text-destructive animate-in fade-in zoom-in duration-200">
                {formError.message}
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="email">
                Email
              </Label>
              <Input
                id="email"
                type="email"
                placeholder="Nhập địa chỉ email"
               
                {...register('email')}
              />
              {errors.email && (
                <p className="text-xs text-destructive">{errors.email.message}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">
                Mật khẩu
              </Label>
              <div className="relative">
                <Input
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  placeholder="Nhập mật khẩu"
                  className="pr-10"
                  {...register('password')}
                />
                <button
                  type="button"
                  tabIndex={-1}
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                >
                  {showPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                </button>
              </div>
              {errors.password && (
                <p className="text-xs text-destructive">{errors.password.message}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="confirmPassword">
                Xác nhận mật khẩu
              </Label>
              <div className="relative">
                <Input
                  id="confirmPassword"
                  type={showConfirmPassword ? 'text' : 'password'}
                  placeholder="Nhập lại mật khẩu"
                  className="pr-10"
                  {...register('confirmPassword')}
                />
                <button
                  type="button"
                  tabIndex={-1}
                  onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                >
                  {showConfirmPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                </button>
              </div>
              {errors.confirmPassword && (
                <p className="text-xs text-destructive">{errors.confirmPassword.message}</p>
              )}
            </div>
          </CardContent>

          <CardFooter className="flex-col gap-4 pt-4">
            <Button
              type="submit"
              disabled={isLoading}
              className="w-full"
            >
              {isLoading ? (
                <span className="flex items-center gap-2">
                  <Loader2 className="size-4 animate-spin" />
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
          </CardFooter>
        </form>
      </Card>
    </div>
  )
}
