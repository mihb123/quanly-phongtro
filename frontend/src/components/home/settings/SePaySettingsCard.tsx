import { useCallback, useEffect, useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { AlertCircle, CheckCircle2, Copy, Eye, EyeOff, Loader2, RefreshCw, Trash2 } from '@/components/icons'
import { useForm, useWatch } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'
import {
  deleteSePayConfig,
  getPaymentPublicKey,
  getSePayConfig,
  reconcileSePay,
  saveSePayConfig,
  type SePayConfigPayload,
  type SePayConfigStatus,
  type SePayReconcileResult,
} from '@/api/payment'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useAuth } from '@/contexts/AuthContext'
import { encryptRSA } from '@/utils/encryption'

const sePayConfigSchema = z.object({
  bankShortName: z.string().min(1, 'Vui lòng nhập Tên ngân hàng'),
  accountNumber: z.string().min(1, 'Vui lòng nhập Số tài khoản'),
  accountName: z.string().min(1, 'Vui lòng nhập Tên chủ tài khoản'),
  codePrefix: z.string().min(1, 'Vui lòng nhập Tiền tố mã thanh toán'),
  webhookAuthMethod: z.string().min(1, 'Vui lòng chọn phương thức xác thực'),
  webhookApiKey: z.string().optional(),
  webhookSecret: z.string().optional(),
  apiToken: z.string().optional(),
}).superRefine((data, ctx) => {
  if (data.webhookAuthMethod === 'apikey' && !data.webhookApiKey) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: 'Vui lòng nhập API Key webhook',
      path: ['webhookApiKey'],
    })
  }
  if (data.webhookAuthMethod === 'hmac' && !data.webhookSecret) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: 'Vui lòng nhập Webhook Secret',
      path: ['webhookSecret'],
    })
  }
})

type SePayConfigForm = z.infer<typeof sePayConfigSchema>

// Card cấu hình tích hợp SePay: lưu tài khoản nhận tiền, webhook, đối soát & xóa cấu hình.
export function SePaySettingsCard() {
  const [status, setStatus] = useState<SePayConfigStatus | null>(null)
  const [initialLoading, setInitialLoading] = useState(true)
  const [isEditing, setIsEditing] = useState(false)
  const [showSecrets, setShowSecrets] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [reconciling, setReconciling] = useState(false)
  const { user } = useAuth()
  const fallbackWebhookURL = user?.user_id
    ? `${window.location.origin}/api/v1/payments/providers/sepay/managers/${user.user_id}/webhook`
    : ''
  const webhookURL = status?.webhook_url || fallbackWebhookURL
  const form = useForm<SePayConfigForm>({
    resolver: zodResolver(sePayConfigSchema),
    defaultValues: {
      bankShortName: '',
      accountNumber: '',
      accountName: '',
      codePrefix: 'PH',
      webhookAuthMethod: '',
      webhookApiKey: '',
      webhookSecret: '',
      apiToken: '',
    },
  })

  const authMethod = useWatch({
    control: form.control,
    name: 'webhookAuthMethod',
  })

  const refreshStatus = useCallback(async () => {
    const nextStatus = await getSePayConfig()
    setStatus(nextStatus)
  }, [])

  useEffect(() => {
    const fetchStatus = async () => {
      try {
        await refreshStatus()
      } catch (err) {
        console.error('Failed to load sepay config status', err)
      } finally {
        setInitialLoading(false)
      }
    }
    fetchStatus()
  }, [refreshStatus])

  const handleSave = form.handleSubmit(async (values) => {
    try {
      const { public_key: publicKey } = await getPaymentPublicKey()
      
      const payload: SePayConfigPayload = {
        bank_short_name: values.bankShortName,
        account_number: values.accountNumber,
        account_name: values.accountName,
        code_prefix: values.codePrefix,
        webhook_auth_method: values.webhookAuthMethod,
      }
      
      if (values.webhookApiKey) {
        payload.webhook_api_key = await encryptRSA(values.webhookApiKey, publicKey)
      }
      if (values.webhookSecret) {
        payload.webhook_secret = await encryptRSA(values.webhookSecret, publicKey)
      }
      if (values.apiToken) {
        payload.api_token = await encryptRSA(values.apiToken, publicKey)
      }

      await saveSePayConfig(payload)

      toast.success('Cấu hình SePay thành công')
      form.reset()
      setIsEditing(false)
      await refreshStatus()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { message?: string } }, message?: string }
      toast.error(error.response?.data?.message || error.message || 'Lỗi cấu hình SePay')
    }
  })

  const handleEdit = () => {
    if (status?.has_config) {
      form.reset({
        bankShortName: status.bank_short_name || '',
        accountNumber: '',
        accountName: '',
        codePrefix: status.code_prefix || 'PH',
        webhookAuthMethod: status.webhook_auth_method || '',
        webhookApiKey: '',
        webhookSecret: '',
        apiToken: '',
      })
    } else {
      form.reset({
        bankShortName: '',
        accountNumber: '',
        accountName: '',
        codePrefix: 'PH',
        webhookAuthMethod: '',
        webhookApiKey: '',
        webhookSecret: '',
        apiToken: '',
      })
    }
    setIsEditing(true)
  }

  const handleDelete = async () => {
    setDeleting(true)
    try {
      await deleteSePayConfig()
      toast.success('Đã xóa cấu hình SePay')
      setIsEditing(false)
      form.reset()
      await refreshStatus()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { message?: string } }, message?: string }
      toast.error(error.response?.data?.message || error.message || 'Lỗi xóa cấu hình SePay')
    } finally {
      setDeleting(false)
    }
  }

  const handleReconcile = async () => {
    setReconciling(true)
    try {
      const result: SePayReconcileResult = await reconcileSePay()
      let msg = `Đối soát xong: quét ${result.scanned}, xử lý ${result.processed} giao dịch`
      if (result.truncated) {
        msg += ' (đã chạm giới hạn, chạy lại để tiếp tục)'
      }
      toast.success(msg)
    } catch (err: unknown) {
      const error = err as { response?: { data?: { message?: string } }, message?: string }
      toast.error(error.response?.data?.message || error.message || 'Lỗi đối soát SePay')
    } finally {
      setReconciling(false)
    }
  }

  const copyWebhookURL = async () => {
    if (!webhookURL) {
      return
    }
    await navigator.clipboard.writeText(webhookURL)
    toast.success('Đã copy webhook URL')
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex justify-between items-start gap-4">
          <div>
            <CardTitle>Tích hợp SePay</CardTitle>
            <CardDescription>
              Cấu hình tài khoản SePay riêng cho quản lý để tự động xác nhận chuyển khoản.
            </CardDescription>
          </div>
          {status?.is_active && (
            <Badge className="shrink-0 bg-info/10 text-info">
              Đang ưu tiên tạo QR
            </Badge>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {initialLoading ? (
          <div className="flex items-center gap-2 text-muted-foreground">
            <Loader2 className="w-4 h-4 animate-spin" />
            Đang tải trạng thái...
          </div>
        ) : (
          <div className={`p-4 rounded-xl flex items-start gap-3 border ${status?.has_config ? 'bg-success/10 border-success/30 text-success' : 'bg-warning/10 border-warning/30 text-warning'}`}>
            {status?.has_config ? <CheckCircle2 className="w-5 h-5 mt-0.5 flex-shrink-0" /> : <AlertCircle className="w-5 h-5 mt-0.5 flex-shrink-0" />}
            <div className="min-w-0">
              <p className="font-semibold">{status?.has_config ? 'Đã cấu hình SePay' : 'Chưa cấu hình SePay'}</p>
              {status?.has_config && (
                <p className="text-sm opacity-90 mt-1">Ngân hàng: {status.bank_short_name} - Số TK: {status.masked_account_number}</p>
              )}
            </div>
          </div>
        )}

        {webhookURL && (
          <div className="space-y-2 max-w-2xl">
            <Label htmlFor="sepay-webhook-url">Webhook URL</Label>
            <div className="flex gap-2">
              <Input id="sepay-webhook-url" value={webhookURL} readOnly className="font-mono text-xs" />
              <Button type="button" variant="outline" size="icon" onClick={copyWebhookURL} className="cursor-pointer flex-shrink-0">
                <Copy className="h-4 w-4" />
              </Button>
            </div>
          </div>
        )}

        {(!status?.has_config || isEditing) ? (
          <form onSubmit={handleSave} className="grid gap-4 max-w-xl">
            <div className="space-y-2">
              <Label htmlFor="sepay-bank-short-name">Tên ngân hàng (ví dụ: MBBank)</Label>
              <Input
                id="sepay-bank-short-name"
                placeholder="Nhập tên viết tắt ngân hàng"
                {...form.register('bankShortName')}
              />
              {form.formState.errors.bankShortName && (
                <p className="text-sm text-destructive">{form.formState.errors.bankShortName.message}</p>
              )}
            </div>
            
            <div className="space-y-2">
              <Label htmlFor="sepay-account-number">Số tài khoản</Label>
              <Input
                id="sepay-account-number"
                placeholder="Nhập số tài khoản"
                {...form.register('accountNumber')}
              />
              {form.formState.errors.accountNumber && (
                <p className="text-sm text-destructive">{form.formState.errors.accountNumber.message}</p>
              )}
            </div>
            
            <div className="space-y-2">
              <Label htmlFor="sepay-account-name">Tên chủ tài khoản</Label>
              <Input
                id="sepay-account-name"
                placeholder="Nhập tên chủ tài khoản"
                {...form.register('accountName')}
              />
              {form.formState.errors.accountName && (
                <p className="text-sm text-destructive">{form.formState.errors.accountName.message}</p>
              )}
            </div>
            
            <div className="space-y-2">
              <Label htmlFor="sepay-code-prefix">Tiền tố mã thanh toán</Label>
              <Input
                id="sepay-code-prefix"
                placeholder="Nhập tiền tố mã, ví dụ: PT"
                {...form.register('codePrefix')}
              />
              {form.formState.errors.codePrefix && (
                <p className="text-sm text-destructive">{form.formState.errors.codePrefix.message}</p>
              )}
            </div>
            
            <div className="space-y-2">
              <Label htmlFor="sepay-webhook-auth-method">Phương thức xác thực webhook</Label>
              <select
                id="sepay-webhook-auth-method"
                className="flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                {...form.register('webhookAuthMethod')}
              >
                <option value="" disabled>Chọn phương thức</option>
                <option value="apikey">API Key</option>
                <option value="hmac">HMAC-SHA256</option>
                <option value="none">Không xác thực</option>
              </select>
            </div>

            {authMethod === 'apikey' && (
              <div className="space-y-2">
                <Label htmlFor="sepay-webhook-api-key">API Key webhook</Label>
                <Input
                  id="sepay-webhook-api-key"
                  type={showSecrets ? 'text' : 'password'}
                  placeholder="Nhập API Key webhook"
                  {...form.register('webhookApiKey')}
                />
                {form.formState.errors.webhookApiKey && (
                  <p className="text-sm text-destructive">{form.formState.errors.webhookApiKey.message}</p>
                )}
              </div>
            )}
            
            {authMethod === 'hmac' && (
              <div className="space-y-2">
                <Label htmlFor="sepay-webhook-secret">Mã HMAC (Webhook Secret)</Label>
                <Input
                  id="sepay-webhook-secret"
                  type={showSecrets ? 'text' : 'password'}
                  placeholder="Nhập mã HMAC Secret từ SePay"
                  {...form.register('webhookSecret')}
                />
                {form.formState.errors.webhookSecret && (
                  <p className="text-sm text-destructive">{form.formState.errors.webhookSecret.message}</p>
                )}
              </div>
            )}
            
            <div className="space-y-2">
              <Label htmlFor="sepay-api-token">API Token (Tùy chọn, dùng cho đối soát)</Label>
              <Input
                id="sepay-api-token"
                type={showSecrets ? 'text' : 'password'}
                placeholder="Nhập API Token"
                {...form.register('apiToken')}
              />
              {form.formState.errors.apiToken && (
                <p className="text-sm text-destructive">{form.formState.errors.apiToken.message}</p>
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
            <Button type="button" onClick={handleEdit} className="cursor-pointer">
              Cập nhật SePay
            </Button>
            <Button type="button" variant="outline" onClick={handleReconcile} disabled={reconciling} className="cursor-pointer">
              {reconciling ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <RefreshCw className="mr-2 h-4 w-4" />}
              Đối soát giao dịch
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
