import { useCallback, useEffect, useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { AlertCircle, CheckCircle2, Eye, EyeOff, Loader2 } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'
import { getZaloConfigStatus, getZaloPublicKey, saveZaloConfig } from '@/api/zalo'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useAuth } from '@/contexts/AuthContext'
import { encryptRSA } from '@/utils/encryption'

const zaloConfigSchema = z.object({
  botToken: z.string().min(1, 'Vui lòng nhập Bot Token'),
})

type ZaloConfigForm = z.infer<typeof zaloConfigSchema>

export function ZaloBotSettingsCard() {
  const [showBotToken, setShowBotToken] = useState(false)
  const [hasConfig, setHasConfig] = useState<boolean | null>(null)
  const [isLinked, setIsLinked] = useState(false)
  const [botId, setBotId] = useState('')
  const [managerId, setManagerId] = useState('')
  const [initialLoading, setInitialLoading] = useState(true)
  const [isUpdatingToken, setIsUpdatingToken] = useState(false)
  const { user } = useAuth()
  const form = useForm<ZaloConfigForm>({
    resolver: zodResolver(zaloConfigSchema),
    defaultValues: { botToken: '' },
  })

  const isMobile = /iPhone|iPad|iPod|Android/i.test(navigator.userAgent)
  const managerName = user?.full_name || managerId

  const refreshStatus = useCallback(async () => {
    const status = await getZaloConfigStatus()
    setHasConfig(status.has_config)
    setIsLinked(status.is_linked)
    setBotId(status.bot_id)
    setManagerId(status.manager_id)
  }, [])

  useEffect(() => {
    const fetchStatus = async () => {
      try {
        await refreshStatus()
      } catch (err) {
        console.error('Failed to load zalo config status', err)
      } finally {
        setInitialLoading(false)
      }
    }
    fetchStatus()
  }, [refreshStatus, user?.user_id])

  const handleSave = form.handleSubmit(async (values) => {
    try {
      const { public_key: publicKey } = await getZaloPublicKey()
      await saveZaloConfig({
        bot_token: await encryptRSA(values.botToken, publicKey),
      })

      toast.success('Cấu hình Zalo Bot thành công')
      form.reset()
      setIsUpdatingToken(false)
      await refreshStatus()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { message?: string } }, message?: string }
      toast.error(error.response?.data?.message || error.message || 'Lỗi cấu hình Zalo')
    }
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle>Tích hợp Zalo Bot</CardTitle>
        <CardDescription>
          Cấu hình Bot để hệ thống tự động gửi thông báo hóa đơn qua Zalo.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {initialLoading ? (
          <div className="flex items-center gap-2 text-slate-500">
            <Loader2 className="w-4 h-4 animate-spin" />
            Đang tải trạng thái...
          </div>
        ) : (
          <div className={`p-4 rounded-xl flex items-start gap-3 border ${hasConfig ? 'bg-green-50 border-green-200 text-green-700' : 'bg-amber-50 border-amber-200 text-amber-700'}`}>
            {hasConfig ? <CheckCircle2 className="w-5 h-5 mt-0.5 flex-shrink-0" /> : <AlertCircle className="w-5 h-5 mt-0.5 flex-shrink-0" />}
            <div>
              <p className="font-semibold">{hasConfig ? 'Đã kết nối Zalo Bot' : 'Chưa cấu hình Zalo Bot'}</p>
              <div className="text-sm opacity-90 space-y-1 mt-1">
                {hasConfig ? (
                  <p>Hệ thống đã sẵn sàng gửi tin nhắn qua Zalo. Bạn có thể cập nhật Token bên dưới nếu cần thiết.</p>
                ) : (
                  <p>Vui lòng cung cấp Bot Token để kích hoạt tính năng.</p>
                )}
              </div>
            </div>
          </div>
        )}

        {!initialLoading && hasConfig && !isLinked && botId && (
          <div className="p-4 rounded-xl flex items-start gap-3 border bg-blue-50 border-blue-200 text-blue-800">
            <AlertCircle className="w-5 h-5 mt-0.5 flex-shrink-0 text-blue-600" />
            <div>
              <p className="font-semibold text-blue-900 mb-3">Yêu cầu hoàn tất liên kết tài khoản</p>
              {isMobile ? (
                <a
                  href={`https://zalo.me/${botId}?text=Kich hoat bot cho tai khoan: ${managerName}`}
                  target="_blank"
                  rel="noreferrer"
                  className="inline-flex cursor-pointer items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors bg-blue-600 text-white shadow hover:bg-blue-600/90 h-9 px-4 py-2"
                >
                  Mở ứng dụng Zalo ngay
                </a>
              ) : (
                <a
                  href={`https://chat.zalo.me/?c=${botId}&text=Kich hoat bot cho tai khoan: ${managerName}`}
                  target="_blank"
                  rel="noreferrer"
                  className="inline-flex cursor-pointer items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors bg-blue-600 text-white shadow hover:bg-blue-600/90 h-9 px-4 py-2"
                >
                  Mở Zalo Web / PC
                </a>
              )}
            </div>
          </div>
        )}

        <form onSubmit={handleSave} className="grid gap-4 mt-6 max-w-xl">
          {(!hasConfig || isUpdatingToken) ? (
            <>
              <div className="space-y-2">
                <Label htmlFor="bot-token">Bot Token</Label>
                <div className="relative">
                  <Input
                    id="bot-token"
                    type={showBotToken ? 'text' : 'password'}
                    placeholder="Nhập Bot Token mới"
                    className="pr-10"
                    {...form.register('botToken')}
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent cursor-pointer"
                    onClick={() => setShowBotToken(!showBotToken)}
                  >
                    {showBotToken ? <EyeOff className="h-4 w-4 text-slate-500" /> : <Eye className="h-4 w-4 text-slate-500" />}
                  </Button>
                </div>
                {form.formState.errors.botToken && (
                  <p className="text-sm text-red-600">{form.formState.errors.botToken.message}</p>
                )}
              </div>

              <div className="flex gap-2 mt-2">
                <Button type="submit" disabled={form.formState.isSubmitting} className="w-fit cursor-pointer">
                  {form.formState.isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  Lưu cấu hình
                </Button>
                {hasConfig && (
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => {
                      setIsUpdatingToken(false)
                      form.reset()
                    }}
                    disabled={form.formState.isSubmitting}
                    className="w-fit cursor-pointer"
                  >
                    Hủy
                  </Button>
                )}
              </div>
            </>
          ) : (
            <Button type="button" onClick={() => setIsUpdatingToken(true)} className="w-fit cursor-pointer">
              Cập nhật Bot Token
            </Button>
          )}
        </form>
      </CardContent>
    </Card>
  )
}
