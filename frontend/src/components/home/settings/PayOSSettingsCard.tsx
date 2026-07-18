import { useCallback, useEffect, useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { AlertCircle, CheckCircle2, Copy, Eye, EyeOff, Loader2, Trash2 } from '@/components/icons'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'
import {
  deletePayOSConfig,
  getPaymentPublicKey,
  getPayOSConfig,
  savePayOSConfig,
  type PayOSConfigStatus,
} from '@/api/payment'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { encryptRSA } from '@/utils/encryption'

const payOSConfigSchema = z.object({
  clientId: z.string().min(1, 'Vui lòng nhập Client ID'),
  apiKey: z.string().min(1, 'Vui lòng nhập API Key'),
  checksumKey: z.string().min(1, 'Vui lòng nhập Checksum Key'),
})

type PayOSConfigForm = z.infer<typeof payOSConfigSchema>

// Card cấu hình tích hợp PayOS: lưu khóa API tạo QR thanh toán, hiển thị webhook & xóa cấu hình.
export function PayOSSettingsCard() {
  const [status, setStatus] = useState<PayOSConfigStatus | null>(null)
  const [initialLoading, setInitialLoading] = useState(true)
  const [isEditing, setIsEditing] = useState(false)
  const [showSecrets, setShowSecrets] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const form = useForm<PayOSConfigForm>({
    resolver: zodResolver(payOSConfigSchema),
    defaultValues: {
      clientId: '',
      apiKey: '',
      checksumKey: '',
    },
  })

  const refreshStatus = useCallback(async () => {
    const nextStatus = await getPayOSConfig()
    setStatus(nextStatus)
  }, [])

  useEffect(() => {
    const fetchStatus = async () => {
      try {
        await refreshStatus()
      } catch (err) {
        console.error('Failed to load payos config status', err)
      } finally {
        setInitialLoading(false)
      }
    }
    fetchStatus()
  }, [refreshStatus])

  const handleSave = form.handleSubmit(async (values) => {
    try {
      const { public_key: publicKey } = await getPaymentPublicKey()
      await savePayOSConfig({
        client_id: await encryptRSA(values.clientId, publicKey),
        api_key: await encryptRSA(values.apiKey, publicKey),
        checksum_key: await encryptRSA(values.checksumKey, publicKey),
      })

      toast.success('Cấu hình PayOS thành công')
      form.reset()
      setIsEditing(false)
      await refreshStatus()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { message?: string } }, message?: string }
      toast.error(error.response?.data?.message || error.message || 'Lỗi cấu hình PayOS')
    }
  })

  const handleDelete = async () => {
    setDeleting(true)
    try {
      await deletePayOSConfig()
      toast.success('Đã xóa cấu hình PayOS')
      setIsEditing(false)
      form.reset()
      await refreshStatus()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { message?: string } }, message?: string }
      toast.error(error.response?.data?.message || error.message || 'Lỗi xóa cấu hình PayOS')
    } finally {
      setDeleting(false)
    }
  }

  const copyWebhookURL = async () => {
    if (!status?.webhook_url) {
      return
    }
    await navigator.clipboard.writeText(status.webhook_url)
    toast.success('Đã copy webhook URL')
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Tích hợp PayOS</CardTitle>
        <CardDescription>
          Cấu hình tài khoản PayOS riêng cho quản lý để tạo QR thanh toán hóa đơn.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {initialLoading ? (
          <div className="flex items-center gap-2 text-muted-foreground">
            <Loader2 className="w-4 h-4 animate-spin" />
            Đang tải trạng thái...
          </div>
        ) : (
          <div className={`p-4 rounded-lg flex items-start gap-3 border ${status?.has_config ? 'bg-success/10 border-success/30 text-success' : 'bg-warning/10 border-warning/30 text-warning'}`}>
            {status?.has_config ? <CheckCircle2 className="w-5 h-5 mt-0.5 flex-shrink-0" /> : <AlertCircle className="w-5 h-5 mt-0.5 flex-shrink-0" />}
            <div className="min-w-0">
              <p className="font-semibold">{status?.has_config ? 'Đã cấu hình PayOS' : 'Chưa cấu hình PayOS'}</p>
              {status?.has_config && (
                <p className="text-sm opacity-90 mt-1">Client ID: {status.masked_client_id}</p>
              )}
            </div>
          </div>
        )}

        {status?.webhook_url && (
          <div className="space-y-2 max-w-2xl">
            <Label htmlFor="payos-webhook-url">Webhook URL</Label>
            <div className="flex gap-2">
              <Input id="payos-webhook-url" value={status.webhook_url} readOnly className="font-mono text-xs" />
              <Button type="button" variant="outline" size="icon" onClick={copyWebhookURL} className="cursor-pointer flex-shrink-0">
                <Copy className="h-4 w-4" />
              </Button>
            </div>
          </div>
        )}

        {(!status?.has_config || isEditing) ? (
          <form onSubmit={handleSave} className="grid gap-4 max-w-xl">
            <div className="space-y-2">
              <Label htmlFor="payos-client-id">Client ID</Label>
              <Input
                id="payos-client-id"
                type={showSecrets ? 'text' : 'password'}
                placeholder="Nhập Client ID"
                {...form.register('clientId')}
              />
              {form.formState.errors.clientId && (
                <p className="text-sm text-destructive">{form.formState.errors.clientId.message}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="payos-api-key">API Key</Label>
              <Input
                id="payos-api-key"
                type={showSecrets ? 'text' : 'password'}
                placeholder="Nhập API Key"
                {...form.register('apiKey')}
              />
              {form.formState.errors.apiKey && (
                <p className="text-sm text-destructive">{form.formState.errors.apiKey.message}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="payos-checksum-key">Checksum Key</Label>
              <Input
                id="payos-checksum-key"
                type={showSecrets ? 'text' : 'password'}
                placeholder="Nhập Checksum Key"
                {...form.register('checksumKey')}
              />
              {form.formState.errors.checksumKey && (
                <p className="text-sm text-destructive">{form.formState.errors.checksumKey.message}</p>
              )}
            </div>

            <div className="flex flex-wrap gap-2">
              <Button type="submit" disabled={form.formState.isSubmitting} className="cursor-pointer">
                {form.formState.isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Lưu cấu hình
              </Button>
              <Button type="button" variant="outline" onClick={() => setShowSecrets(!showSecrets)} className="cursor-pointer">
                {showSecrets ? <EyeOff className="mr-2 h-4 w-4" /> : <Eye className="mr-2 h-4 w-4" />}
                {showSecrets ? 'Ẩn khóa' : 'Hiện khóa'}
              </Button>
              {status?.has_config && (
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => {
                    setIsEditing(false)
                    form.reset()
                  }}
                  disabled={form.formState.isSubmitting}
                  className="cursor-pointer"
                >
                  Hủy
                </Button>
              )}
            </div>
          </form>
        ) : (
          <div className="flex flex-wrap gap-2">
            <Button type="button" onClick={() => setIsEditing(true)} className="cursor-pointer">
              Cập nhật PayOS
            </Button>
            <Button type="button" variant="outline" onClick={handleDelete} disabled={deleting} className="cursor-pointer">
              {deleting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Trash2 className="mr-2 h-4 w-4" />}
              Xóa cấu hình
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
