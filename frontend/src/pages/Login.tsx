import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Eye, EyeOff, LogIn, Loader2 } from '@/components/icons'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from '@/components/ui/card'
import { login as loginAccount, getMe } from '@/api/auth'
import { useFormError } from '@/hooks/useFormError'
import { useAuth } from '@/contexts/AuthContext'

const loginSchema = z.object({
  email: z.string().email('Địa chỉ email không hợp lệ'),
  password: z.string().min(8, 'Mật khẩu phải có ít nhất 8 ký tự'),
})

type LoginFormValues = z.infer<typeof loginSchema>

// Thời gian tối đa chờ định vị GPS trước khi bỏ qua để không chặn luồng đăng nhập.
const LOCATION_TIMEOUT_MS = 4000

// Lấy toạ độ kiểu best-effort: trả vị trí nếu có sẵn/nhanh (ưu tiên cache gần đây),
// còn hết thời gian hoặc bị từ chối thì trả null để đăng nhập không bị chặn.
function getBestEffortPosition(): Promise<GeolocationPosition | null> {
  return new Promise((resolve) => {
    let settled = false
    const finish = (value: GeolocationPosition | null) => {
      if (settled) return
      settled = true
      resolve(value)
    }
    const timer = setTimeout(() => finish(null), LOCATION_TIMEOUT_MS)
    navigator.geolocation.getCurrentPosition(
      (position) => { clearTimeout(timer); finish(position) },
      () => { clearTimeout(timer); finish(null) },
      { timeout: LOCATION_TIMEOUT_MS, maximumAge: 600000 },
    )
  })
}

// Trang đăng nhập: form email/mật khẩu, có lấy vị trí (geolocation) trước khi gọi API
export default function LoginPage() {
  const navigate = useNavigate()
  const [showPassword, setShowPassword] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const { formError, handleApiError, clearFormError } = useFormError()

  const { login } = useAuth()
  const { register, handleSubmit, formState: { errors } } = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
  })

  async function onSubmit(formData: LoginFormValues) {
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
      localStorage.setItem('has_asked_location', 'true')
      const position = await getBestEffortPosition()
      if (position) {
        latitude = position.coords.latitude
        longitude = position.coords.longitude
      } else if (import.meta.env.DEV) {
        // Không lấy được GPS: mock toạ độ Hà Nội khi chạy dev.
        latitude = 21.028511
        longitude = 105.804817
      }
    }

    try {
      const result = await loginAccount({
        email: formData.email,
        password: formData.password,
        Latitude: latitude,
        Longitude: longitude,
      })
      // Ưu tiên user trả kèm trong login response; fallback getMe nếu thiếu.
      const user = result.user ?? (await getMe())
      login(user)
      navigate('/')
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
              <LogIn className="size-6 text-primary" />
            </div>
            <CardTitle className="text-2xl font-semibold tracking-tight">Đăng nhập</CardTitle>
            <CardDescription>
              Nhập email và mật khẩu của bạn để tiếp tục
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
                Địa chỉ Email
              </Label>
              <Input
                id="email"
                type="email"
                placeholder="vd: user@example.com"
               
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
                  Đang đăng nhập...
                </span>
              ) : 'Đăng nhập'}
            </Button>
            <p className="text-sm text-muted-foreground text-center">
              Chưa có tài khoản?{' '}
              <Link to="/register" className="text-primary hover:text-primary/90 font-medium transition-colors hover:underline underline-offset-4">
                Đăng ký ngay
              </Link>
            </p>
          </CardFooter>
        </form>
      </Card>
    </div>
  )
}
