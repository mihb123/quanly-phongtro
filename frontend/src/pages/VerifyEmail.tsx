import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Mail, CheckCircle2 } from '@/components/icons'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { createOTP, verifyEmailOTP, getMe } from '@/api/auth'
import { useFormError } from '@/hooks/useFormError'
import { useAuth } from '@/contexts/AuthContext'

const verifySchema = z.object({
  otp: z.string().length(6, 'Mã OTP phải gồm 6 chữ số').regex(/^\d+$/, 'Mã OTP chỉ chứa chữ số'),
})

type VerifyFormValues = z.infer<typeof verifySchema>

// Trang xác thực email: gửi OTP qua email rồi nhập mã 6 số để kích hoạt tài khoản
export default function VerifyEmailPage() {
  const navigate = useNavigate()
  const { user, login } = useAuth()
  const [isLoading, setIsLoading] = useState(false)
  const [isSendingOTP, setIsSendingOTP] = useState(false)
  const [otpSent, setOtpSent] = useState(false)
  const [successMsg, setSuccessMsg] = useState('')
  const { formError, handleApiError, clearFormError } = useFormError()

  const { register, handleSubmit, formState: { errors } } = useForm<VerifyFormValues>({
    resolver: zodResolver(verifySchema),
  })

  // If already activated, redirect to home
  if (user?.is_activated) {
    navigate('/')
    return null
  }

  async function handleSendOTP() {
    setIsSendingOTP(true)
    clearFormError()
    setSuccessMsg('')
    try {
      await createOTP()
      setOtpSent(true)
      setSuccessMsg('Đã gửi mã OTP đến email của bạn. Vui lòng kiểm tra mục Spam nếu không thấy.')
    } catch (error) {
      handleApiError(error)
    } finally {
      setIsSendingOTP(false)
    }
  }

  async function onSubmit(formData: VerifyFormValues) {
    setIsLoading(true)
    clearFormError()
    setSuccessMsg('')

    try {
      await verifyEmailOTP(formData.otp)
      // Refresh user context to get is_activated=true
      const updatedUser = await getMe()
      login(updatedUser)
      setSuccessMsg('Xác thực email thành công! Đang chuyển hướng...')
      setTimeout(() => navigate('/'), 1500)
    } catch (error) {
      handleApiError(error)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background px-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <div className="flex items-center justify-center size-12 rounded-lg bg-primary/10 border border-primary/20 mb-2">
            <Mail className="size-6 text-primary" />
          </div>
          <CardTitle className="text-2xl font-semibold tracking-tight">Xác thực Email</CardTitle>
          <CardDescription>
            Vui lòng xác thực email <span className="font-medium text-foreground">{user?.email}</span> để tiếp tục
          </CardDescription>
        </CardHeader>

        <CardContent className="space-y-4">
          {formError?.message && (
            <div className="rounded-lg bg-destructive/10 border border-destructive/20 px-4 py-3 text-sm text-destructive animate-in fade-in zoom-in duration-200">
              {formError.message}
            </div>
          )}

          {successMsg && (
            <div className="rounded-lg bg-success/10 border border-success/20 px-4 py-3 text-sm text-success animate-in fade-in zoom-in duration-200 flex items-start gap-2">
              <CheckCircle2 className="size-4 shrink-0 mt-[2px]" />
              <span>{successMsg}</span>
            </div>
          )}

          {!otpSent ? (
            <div className="space-y-4 py-4 text-center">
              <p className="text-sm text-muted-foreground">
                Hệ thống sẽ gửi một mã OTP gồm 6 chữ số đến email của bạn.
              </p>
              <Button
                onClick={handleSendOTP}
                disabled={isSendingOTP}
                className="w-full"
              >
                {isSendingOTP ? 'Đang gửi...' : 'Gửi mã OTP'}
              </Button>
            </div>
          ) : (
            <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="otp">
                  Mã OTP (6 số)
                </Label>
                <Input
                  id="otp"
                  type="text"
                  maxLength={6}
                  placeholder="Nhập mã OTP..."
                  className="h-10 text-center font-mono text-lg tracking-[0.5em] placeholder:tracking-normal"
                  {...register('otp')}
                />
                {errors.otp && (
                  <p className="text-xs text-destructive">{errors.otp.message}</p>
                )}
              </div>

              <div className="pt-4 space-y-4">
                <Button
                  type="submit"
                  disabled={isLoading}
                  className="w-full"
                >
                  {isLoading ? 'Đang xác thực...' : 'Xác thực'}
                </Button>

                <div className="text-center">
                  <button
                    type="button"
                    onClick={handleSendOTP}
                    disabled={isSendingOTP}
                    className="text-sm text-primary hover:text-primary/80 font-medium disabled:opacity-50 cursor-pointer"
                  >
                    Gửi lại mã OTP
                  </button>
                </div>
              </div>
            </form>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
