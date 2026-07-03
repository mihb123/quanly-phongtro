import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { UserCircle, Shield, Phone, Save, LogOut, Key, Droplets, Check, Sun, Moon, Monitor } from '@/components/icons'
import { AppModal } from '@/components/shared/AppModal'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useAuth } from '@/contexts/AuthContext'
import { updateProfile } from '@/api/auth'
import { THEMES, normalizeTheme, applyTheme, type ThemeId } from '@/lib/theme'
import { COLOR_MODES, normalizeColorMode, applyColorMode, type ColorMode } from '@/lib/colorMode'
import { ChangePasswordModal } from './ChangePasswordModal'

const profileSchema = z.object({
  full_name: z.string().min(1, 'Họ tên không được để trống'),
  phone: z.string().min(1, 'Số điện thoại không được để trống'),
})

type ProfileFormValues = z.infer<typeof profileSchema>

// Icon cho từng chế độ hiển thị, tra theo id để render nút chọn chế độ.
const COLOR_MODE_ICONS: Record<ColorMode, typeof Sun> = {
  light: Sun,
  dark: Moon,
  system: Monitor,
}

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

  // Theme id currently being persisted (drives the spinner + optimistic highlight).
  const [savingTheme, setSavingTheme] = useState<ThemeId | null>(null)
  const currentTheme = normalizeTheme(user?.theme)
  const activeTheme = savingTheme ?? currentTheme

  // Apply the picked theme instantly for preview, then persist it to the DB so
  // it syncs across devices. Reverts the preview if the request fails.
  const handleSelectTheme = async (theme: ThemeId) => {
    if (theme === currentTheme || savingTheme) return
    const previous = currentTheme
    applyTheme(theme)
    setSavingTheme(theme)
    try {
      const updatedUser = await updateProfile({ theme })
      if (updatedUser) {
        updateUser(updatedUser)
      }
    } catch (error) {
      console.error('Cập nhật giao diện thất bại', error)
      applyTheme(previous)
      toast.error('Không thể lưu giao diện, vui lòng thử lại!')
    } finally {
      setSavingTheme(null)
    }
  }

  // Appearance mode (system/light/dark) currently being persisted.
  const [savingMode, setSavingMode] = useState<ColorMode | null>(null)
  const currentMode = normalizeColorMode(user?.color_mode)
  const activeMode = savingMode ?? currentMode

  // Apply the picked mode instantly for preview, then persist it to the DB so
  // it syncs across devices. Reverts the preview if the request fails.
  const handleSelectMode = async (mode: ColorMode) => {
    if (mode === currentMode || savingMode) return
    const previous = currentMode
    applyColorMode(mode)
    setSavingMode(mode)
    try {
      const updatedUser = await updateProfile({ color_mode: mode })
      if (updatedUser) {
        updateUser(updatedUser)
      }
    } catch (error) {
      console.error('Cập nhật chế độ hiển thị thất bại', error)
      applyColorMode(previous)
      toast.error('Không thể lưu chế độ hiển thị, vui lòng thử lại!')
    } finally {
      setSavingMode(null)
    }
  }

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

          {/* Giao diện: chọn theme màu, áp dụng ngay và lưu vào tài khoản để đồng bộ đa thiết bị */}
          <div className="space-y-4">
            <h3 className="font-bold text-foreground text-sm flex items-center gap-2 border-b border-border/40 pb-2">
              <Droplets className="w-4 h-4 text-muted-foreground" />
              Giao diện
            </h3>

            {/* Chế độ sáng/tối/hệ thống — áp dụng ngay và lưu để đồng bộ đa thiết bị */}
            <div className="space-y-2">
              <Label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Chế độ</Label>
              <div className="grid grid-cols-3 gap-2">
                {COLOR_MODES.map((mode) => {
                  const Icon = COLOR_MODE_ICONS[mode.id]
                  const isActive = activeMode === mode.id
                  const isSaving = savingMode === mode.id
                  return (
                    <button
                      key={mode.id}
                      type="button"
                      onClick={() => handleSelectMode(mode.id)}
                      disabled={savingMode !== null}
                      aria-pressed={isActive}
                      className={`cursor-pointer flex flex-col items-center justify-center gap-1.5 rounded-xl border py-3 transition-colors disabled:opacity-60 ${
                        isActive
                          ? 'border-primary ring-2 ring-primary/30 bg-primary/5 text-primary'
                          : 'border-border/60 hover:bg-secondary text-muted-foreground'
                      }`}
                    >
                      {isSaving ? (
                        <span className="h-5 w-5 animate-spin rounded-full border-2 border-primary/30 border-t-primary" />
                      ) : (
                        <Icon className="h-5 w-5" />
                      )}
                      <span className="text-xs font-bold">{mode.label}</span>
                    </button>
                  )
                })}
              </div>
            </div>

            {/* Tông màu — chọn theme màu (Red/Sky) */}
            <Label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Tông màu</Label>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              {THEMES.map((theme) => {
                const isActive = activeTheme === theme.id
                const isSaving = savingTheme === theme.id
                return (
                  <button
                    key={theme.id}
                    type="button"
                    onClick={() => handleSelectTheme(theme.id)}
                    disabled={savingTheme !== null}
                    aria-pressed={isActive}
                    className={`cursor-pointer relative flex items-center gap-3 rounded-xl border p-3 text-left transition-colors disabled:opacity-60 ${
                      isActive
                        ? 'border-primary ring-2 ring-primary/30 bg-primary/5'
                        : 'border-border/60 hover:bg-secondary'
                    }`}
                  >
                    <span
                      className="h-8 w-8 shrink-0 rounded-full border border-border/40 shadow-sm"
                      style={{ backgroundColor: theme.swatch }}
                    />
                    <span className="flex flex-col">
                      <span className="font-bold text-sm">{theme.label}</span>
                      <span className="text-[10px] text-muted-foreground font-normal">{theme.description}</span>
                    </span>
                    {isSaving ? (
                      <span className="ml-auto h-4 w-4 animate-spin rounded-full border-2 border-primary/30 border-t-primary" />
                    ) : isActive ? (
                      <span className="ml-auto flex h-5 w-5 items-center justify-center rounded-full bg-primary text-primary-foreground">
                        <Check className="h-3 w-3" />
                      </span>
                    ) : null}
                  </button>
                )
              })}
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
