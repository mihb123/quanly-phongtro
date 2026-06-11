import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Mail, CheckCircle2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card } from '@/components/ui/card'
import { createOTP, verifyEmailOTP, getMe } from '@/api/auth'
import { useFormError } from '@/hooks/useFormError'
import { useAuth } from '@/contexts/AuthContext'

const verifySchema = z.object({
  otp: z.string().length(6, 'Mã OTP phải gồm 6 chữ số').regex(/^\d+$/, 'Mã OTP chỉ chứa chữ số'),
})

type VerifyFormValues = z.infer<typeof verifySchema>

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
      setSuccessMsg('Đã gửi mã OTP đến email của bạn')
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
    <div className="min-h-screen flex items-center justify-center bg-background px-4 relative overflow-hidden">
      <Card className="w-full max-w-md relative bg-card border border-border/40 shadow-[0_32px_64px_-12px_rgba(0,0,0,0.08)] rounded-2xl overflow-hidden">
        <div className="flex flex-col">
          {/* Header Section */}
          <div className="p-8 pb-6 space-y-2">
            <div className="flex items-center justify-center w-12 h-12 rounded-xl bg-primary/10 border border-primary/20 mb-2">
              <Mail className="w-6 h-6 text-primary" />
            </div>
            <h2 className="text-2xl font-bold tracking-tight text-foreground">Xác thực Email</h2>
            <p className="text-muted-foreground text-sm">
              Vui lòng xác thực email <span className="font-medium text-foreground">{user?.email}</span> để tiếp tục
            </p>
          </div>

          {/* Content */}
          <div className="px-8 space-y-4">
            {formError?.message && (
              <div className="rounded-lg bg-red-50 border border-red-200 px-4 py-3 text-sm text-red-600 animate-in fade-in zoom-in duration-200">
                {formError.message}
              </div>
            )}
            
            {successMsg && (
              <div className="rounded-lg bg-green-50 border border-green-200 px-4 py-3 text-sm text-green-600 animate-in fade-in zoom-in duration-200 flex items-center gap-2">
                <CheckCircle2 className="w-4 h-4" />
                {successMsg}
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
                  className="w-full h-11 bg-primary hover:bg-primary/90"
                >
                  {isSendingOTP ? 'Đang gửi...' : 'Gửi mã OTP'}
                </Button>
              </div>
            ) : (
              <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="otp" className="text-foreground text-sm font-medium">
                    Mã OTP (6 số)
                  </Label>
                  <Input
                    id="otp"
                    type="text"
                    maxLength={6}
                    placeholder="Nhập mã OTP..."
                    className="bg-background border-border text-center tracking-[0.5em] text-lg font-mono placeholder:tracking-normal focus:border-primary/50 focus:ring-primary/20 h-12 transition-all"
                    {...register('otp')}
                  />
                  {errors.otp && (
                    <p className="text-xs text-red-500">{errors.otp.message}</p>
                  )}
                </div>
                
                <div className="pt-4 space-y-4">
                  <Button
                    type="submit"
                    disabled={isLoading}
                    className="w-full h-11 bg-primary hover:bg-primary/90 text-primary-foreground font-semibold"
                  >
                    {isLoading ? 'Đang xác thực...' : 'Xác thực'}
                  </Button>
                  
                  <div className="text-center">
                    <button
                      type="button"
                      onClick={handleSendOTP}
                      disabled={isSendingOTP}
                      className="text-sm text-primary hover:text-primary/80 font-medium disabled:opacity-50"
                    >
                      Gửi lại mã OTP
                    </button>
                  </div>
                </div>
              </form>
            )}
          </div>
          
          <div className="p-8 pb-4" />
        </div>
      </Card>
    </div>
  )
}
