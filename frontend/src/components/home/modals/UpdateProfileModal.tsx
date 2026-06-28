import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { UserCircle, Shield, Phone, Save, LogOut, Key } from '@/components/icons'
import { AppModal } from '@/components/shared/AppModal'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useAuth } from '@/contexts/AuthContext'
import { updateProfile } from '@/api/auth'
import { ChangePasswordModal } from './ChangePasswordModal'

const profileSchema = z.object({
  full_name: z.string().min(1, 'Họ tên không được để trống'),
  phone: z.string().min(1, 'Số điện thoại không được để trống'),
})

type ProfileFormValues = z.infer<typeof profileSchema>

// Modal hồ sơ & cài đặt: vỏ dùng AppModal (Esc/click nền/X tự xử lý), giữ nguyên logic RHF/Zod cập nhật hồ sơ.
export function UpdateProfileModal({ onClose }: { onClose: () => void }) {
  const { user, updateUser, logout } = useAuth()
  const navigate = useNavigate()
  const [isLoading, setIsLoading] = useState(false)

  const { register, handleSubmit, formState: { errors } } = useForm<ProfileFormValues>({
    resolver: zodResolver(profileSchema),
    defaultValues: {
      full_name: user?.full_name || '',
      phone: user?.phone || '',
    }
  })

  const [showChangePassword, setShowChangePassword] = useState(false)

  const onSubmit = async (values: ProfileFormValues) => {
    setIsLoading(true)
    try {
      const payload = {
        full_name: values.full_name,
        phone: values.phone,
      }

      const updatedUser = await updateProfile(payload)
      if (updatedUser) {
        updateUser(updatedUser)
        onClose()
      }
    } catch (error) {
      console.error("Cập nhật profile thất bại", error)
      alert("Cập nhật thông tin thất bại, vui lòng thử lại!")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <>
      {showChangePassword && (
        <ChangePasswordModal onClose={() => setShowChangePassword(false)} />
      )}
      <AppModal
        open
        onClose={onClose}
        title={
          <span className="flex items-center gap-2">
            <UserCircle className="w-6 h-6 text-primary" />
            Hồ sơ & Cài đặt
          </span>
        }
        contentClassName="sm:max-w-2xl"
        dismissible={!isLoading}
      >
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-8">

          {/* Thông tin cá nhân */}
          <div className="space-y-4">
            <h3 className="font-bold text-foreground text-sm flex items-center gap-2 border-b border-border/40 pb-2">
              <Shield className="w-4 h-4 text-muted-foreground"/>
              Thông tin cá nhân
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
              <div className="space-y-2">
                <Label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Họ và tên</Label>
                <div className="relative">
                  <UserCircle className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                  <Input {...register('full_name')} placeholder="Nhập họ tên" className="pl-9 border-border/60 bg-background" />
                </div>
                {errors.full_name && <span className="text-destructive text-xs">{errors.full_name.message}</span>}
              </div>

              <div className="space-y-2">
                <Label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Số điện thoại</Label>
                <div className="relative">
                  <Phone className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                  <Input {...register('phone')} placeholder="09xxxxxxx" className="pl-9 border-border/60 bg-background" />
                </div>
                {errors.phone && <span className="text-destructive text-xs">{errors.phone.message}</span>}
              </div>

              <div className="space-y-2 col-span-1 md:col-span-2 pt-4">
                <Button
                  type="button"
                  variant="outline"
                  className="w-full justify-start gap-3 h-12 rounded-xl border-border/60 hover:bg-secondary transition-colors"
                  onClick={(e) => {
                    e.preventDefault()
                    e.stopPropagation()
                    setShowChangePassword(true)
                  }}
                >
                  <div className="w-8 h-8 rounded-lg bg-primary/10 flex items-center justify-center">
                    <Key className="w-4 h-4 text-primary" />
                  </div>
                  <div className="flex flex-col items-start">
                    <span className="font-bold text-sm">Đổi mật khẩu</span>
                    <span className="text-[10px] text-muted-foreground font-normal">Bảo mật tài khoản của bạn</span>
                  </div>
                </Button>
              </div>
            </div>
          </div>

          <div className="flex items-center justify-between pt-4 border-t border-border/40">
            <Button
              type="button"
              variant="ghost"
              onClick={async () => {
                await logout()
                navigate('/login')
              }}
              className="text-destructive hover:bg-destructive/10 hover:text-destructive gap-2 font-bold rounded-xl px-4"
            >
              <LogOut className="w-4 h-4" /> Đăng xuất
            </Button>

            <div className="flex gap-3">
              <Button type="button" variant="outline" onClick={onClose} className="font-bold rounded-xl px-6">Hủy</Button>
              <Button type="submit" disabled={isLoading} className="shadow-sm font-bold rounded-xl px-6 flex items-center gap-2">
                {isLoading ? 'Đang lưu...' : <><Save className="w-4 h-4" /> Lưu thay đổi</>}
              </Button>
            </div>
          </div>
        </form>
      </AppModal>
    </>
  )
}
