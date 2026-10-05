import { useState } from 'react'
import { Key, Save, Lock } from '@/components/icons'
import { AppModal } from '@/components/shared/AppModal'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { updateProfile } from '@/api/auth'

const changePasswordSchema = z.object({
  old_password: z.string().min(1, 'Vui lòng nhập mật khẩu cũ'),
  password: z.string().min(6, 'Mật khẩu phải có ít nhất 6 ký tự'),
  confirm_password: z.string().min(1, 'Vui lòng xác nhận mật khẩu mới'),
}).refine((data) => data.password === data.confirm_password, {
  message: "Mật khẩu xác nhận không khớp",
  path: ["confirm_password"],
})

type ChangePasswordValues = z.infer<typeof changePasswordSchema>

// Modal đổi mật khẩu: vỏ dùng AppModal (Esc/click nền/X tự xử lý), giữ nguyên logic RHF/Zod đổi mật khẩu.
export function ChangePasswordModal({ onClose }: { onClose: () => void }) {
  const [isLoading, setIsLoading] = useState(false)

  const { register, handleSubmit, formState: { errors } } = useForm<ChangePasswordValues>({
    resolver: zodResolver(changePasswordSchema),
    defaultValues: {
      old_password: '',
      password: '',
      confirm_password: '',
    }
  })

  const onSubmit = async (values: ChangePasswordValues) => {
    setIsLoading(true)
    try {
      const payload = {
        old_password: values.old_password,
        password: values.password,
      }

      const updatedUser = await updateProfile(payload)
      if (updatedUser) {
        alert("Đổi mật khẩu thành công! Bạn sẽ cần đăng nhập lại ở các thiết bị khác bằng mật khẩu mới.")
        onClose()
      }
    } catch (error) {
      console.error("Đổi mật khẩu thất bại", error)
      alert("Đổi mật khẩu thất bại. Vui lòng kiểm tra lại mật khẩu cũ!")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <AppModal
      open
      onClose={onClose}
      title={
        <span className="flex items-center gap-2">
          <Lock className="w-6 h-6 text-primary" />
          Đổi mật khẩu
        </span>
      }
      contentClassName="sm:max-w-md"
      dismissible={!isLoading}
      footer={
        <div className="flex w-full gap-2 sm:justify-end sm:gap-3">
          <Button type="button" variant="outline" onClick={onClose} className="flex-1 sm:flex-none">Hủy</Button>
          <Button type="submit" form="change-password-form" disabled={isLoading} className="flex-1 sm:flex-none">
            {isLoading ? 'Đang xử lý...' : <><Save className="w-4 h-4" /> Xác nhận đổi</>}
          </Button>
        </div>
      }
    >
      <form id="change-password-form" onSubmit={handleSubmit(onSubmit)} className="space-y-6">

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="change-password-old-password" className="text-xs font-semibold text-muted-foreground">Mật khẩu cũ</Label>
            <div className="relative">
              <Key className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input id="change-password-old-password" type="password" autoComplete="current-password" {...register('old_password')} placeholder="Nhập mật khẩu hiện tại" className="pl-9 border-border/60 bg-background" />
            </div>
            {errors.old_password && <span className="text-destructive text-xs">{errors.old_password.message}</span>}
          </div>

          <div className="space-y-2 pt-2 border-t border-border/40">
            <Label htmlFor="change-password-password" className="text-xs font-semibold text-muted-foreground">Mật khẩu mới</Label>
            <div className="relative">
              <Key className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input id="change-password-password" type="password" autoComplete="new-password" {...register('password')} placeholder="Nhập mật khẩu mới" className="pl-9 border-border/60 bg-background" />
            </div>
            {errors.password && <span className="text-destructive text-xs">{errors.password.message}</span>}
          </div>

          <div className="space-y-2">
            <Label htmlFor="change-password-confirm-password" className="text-xs font-semibold text-muted-foreground">Xác nhận mật khẩu mới</Label>
            <div className="relative">
              <Key className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input id="change-password-confirm-password" type="password" autoComplete="new-password" {...register('confirm_password')} placeholder="Nhập lại mật khẩu mới" className="pl-9 border-border/60 bg-background" />
            </div>
            {errors.confirm_password && <span className="text-destructive text-xs">{errors.confirm_password.message}</span>}
          </div>
        </div>

      </form>
    </AppModal>
  )
}
