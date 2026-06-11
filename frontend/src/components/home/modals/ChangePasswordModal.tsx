import { useState, useEffect } from 'react'
import { createPortal } from 'react-dom'
import { Key, Save, Lock } from 'lucide-react'
import { Card } from '@/components/ui/card'
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

  useEffect(() => {
    const handleEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !isLoading) onClose()
    }
    window.addEventListener('keydown', handleEsc)
    return () => window.removeEventListener('keydown', handleEsc)
  }, [onClose, isLoading])

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

  return createPortal(
    <div 
      className="fixed inset-0 flex items-center justify-center bg-background/80 backdrop-blur-sm p-4"
      style={{ zIndex: 70 }}
      onMouseDown={e => {
        if (e.target === e.currentTarget && !isLoading) onClose()
      }}
    >
      <Card className="w-full max-w-md bg-card text-card-foreground shadow-xl border border-border/40 safe-fade-in flex flex-col rounded-3xl">
        <div className="p-5 md:p-6 border-b border-border/40 shrink-0 flex justify-between items-center bg-secondary/10 rounded-t-3xl">
          <h2 className="text-xl font-bold text-foreground flex items-center gap-2">
            <Lock className="w-6 h-6 text-primary" />
            Đổi mật khẩu
          </h2>
        </div>
        <div className="p-5 md:p-6 flex-1">
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
            
            <div className="space-y-4">
              <div className="space-y-2">
                <Label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Mật khẩu cũ</Label>
                <div className="relative">
                  <Key className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                  <Input type="password" {...register('old_password')} placeholder="Nhập mật khẩu hiện tại" className="pl-9 border-border/60 bg-background" />
                </div>
                {errors.old_password && <span className="text-destructive text-xs">{errors.old_password.message}</span>}
              </div>

              <div className="space-y-2 pt-2 border-t border-border/40">
                <Label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Mật khẩu mới</Label>
                <div className="relative">
                  <Key className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                  <Input type="password" {...register('password')} placeholder="Nhập mật khẩu mới" className="pl-9 border-border/60 bg-background" />
                </div>
                {errors.password && <span className="text-destructive text-xs">{errors.password.message}</span>}
              </div>

              <div className="space-y-2">
                <Label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Xác nhận mật khẩu mới</Label>
                <div className="relative">
                  <Key className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                  <Input type="password" {...register('confirm_password')} placeholder="Nhập lại mật khẩu mới" className="pl-9 border-border/60 bg-background" />
                </div>
                {errors.confirm_password && <span className="text-destructive text-xs">{errors.confirm_password.message}</span>}
              </div>
            </div>

            <div className="flex justify-end gap-3 pt-4 border-t border-border/40">
              <Button type="button" variant="outline" onClick={onClose} className="font-bold rounded-xl px-6">Hủy</Button>
              <Button type="submit" disabled={isLoading} className="shadow-sm font-bold rounded-xl px-6 flex items-center gap-2">
                {isLoading ? 'Đang xử lý...' : <><Save className="w-4 h-4" /> Xác nhận đổi</>}
              </Button>
            </div>
          </form>
        </div>
      </Card>
    </div>,
    document.body
  )
}
