import { useCallback, useEffect, useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { AlertCircle, CheckCircle2, Copy, Eye, EyeOff, Loader2, RefreshCw, Trash2 } from '@/components/icons'
import { Controller, useForm, useWatch } from 'react-hook-form'
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
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from '@/components/ui/field'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { useAuth } from '@/contexts/AuthContext'
import { isSePaySupportedBank, SEPAY_SUPPORTED_BANKS } from '@/lib/sepay-banks'
import { cn } from '@/lib/utils'
import { encryptRSA } from '@/utils/encryption'

const sePayConfigSchema = z.object({
  environment: z.enum(['production', 'sandbox']),
  bankShortName: z.string()
    .min(1, 'Vui lòng chọn ngân hàng')
    .refine(isSePaySupportedBank, 'Ngân hàng này chưa được SePay hỗ trợ'),
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
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [reconciling, setReconciling] = useState(false)
  const { user } = useAuth()
  const fallbackWebhookURL = user?.user_id
    ? `${window.location.origin}/api/v1/payments/providers/sepay/managers/${user.user_id}/webhook`
    : ''
  const webhookURL = status?.webhook_url || fallbackWebhookURL
  const form = useForm<SePayConfigForm>({
    resolver: zodResolver(sePayConfigSchema),
    defaultValues: {
      environment: 'production',
      bankShortName: '',
      accountNumber: '',
      accountName: '',
      codePrefix: 'PH',
      webhookAuthMethod: 'hmac',
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
        environment: values.environment,
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

      const result = await saveSePayConfig(payload)

      toast.success(result.bank_account_verified
        ? `Đã xác thực tài khoản ${result.account_holder_name} qua SePay`
        : 'Cấu hình SePay thành công')
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
        environment: status.environment || 'production',
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
        environment: 'production',
        bankShortName: '',
        accountNumber: '',
        accountName: '',
        codePrefix: 'PH',
        webhookAuthMethod: 'hmac',
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
      setDeleteDialogOpen(false)
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
    <Card data-testid="sepay-settings-card">
      <CardHeader>
        <div className="flex flex-col items-start justify-between gap-3 sm:flex-row sm:gap-4">
          <div>
            <CardTitle>Tích hợp SePay</CardTitle>
            <CardDescription>
              Cấu hình tài khoản SePay riêng cho quản lý để tự động xác nhận chuyển khoản.
            </CardDescription>
          </div>
          {status?.is_active && <Badge variant="secondary" className="shrink-0 self-start">Đang ưu tiên tạo QR</Badge>}
        </div>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        {initialLoading ? (
          <div className="flex items-center gap-2 text-muted-foreground">
            <Loader2 className="size-4 animate-spin" />
            Đang tải trạng thái...
          </div>
        ) : (
          <div className={cn(
            'flex items-start gap-3 rounded-lg border p-4',
            status?.has_config ? 'bg-success/10 border-success/30 text-success' : 'bg-warning/10 border-warning/30 text-warning',
          )}>
            {status?.has_config ? <CheckCircle2 className="mt-0.5 size-5 shrink-0" /> : <AlertCircle className="mt-0.5 size-5 shrink-0" />}
            <div className="min-w-0">
              <p className="font-semibold">{status?.has_config ? 'Đã cấu hình SePay' : 'Chưa cấu hình SePay'}</p>
              {status?.has_config && (
                <p className="mt-1 break-words text-sm opacity-90">
                  {status.environment === 'sandbox' ? 'Test Mode' : 'Live'} · {status.bank_short_name} · {status.masked_account_number}
                </p>
              )}
            </div>
          </div>
        )}

        {webhookURL && (
          <Field className="max-w-2xl">
            <FieldLabel htmlFor="sepay-webhook-url">Webhook URL</FieldLabel>
            <div className="flex gap-2">
              <Input id="sepay-webhook-url" value={webhookURL} readOnly className="min-h-11 min-w-0 font-mono text-base md:text-sm" />
              <Button type="button" variant="outline" size="icon" onClick={copyWebhookURL} className="min-h-11 min-w-11 shrink-0 cursor-pointer" aria-label="Sao chép Webhook URL">
                <Copy />
              </Button>
            </div>
          </Field>
        )}

        {(!status?.has_config || isEditing) ? (
          <form onSubmit={handleSave} className="max-w-xl">
            <FieldGroup className="gap-4">
              <Controller
                control={form.control}
                name="environment"
                render={({ field }) => (
                  <Field>
                    <FieldLabel htmlFor="sepay-environment">Môi trường SePay</FieldLabel>
                    <Select value={field.value} onValueChange={field.onChange}>
                      <SelectTrigger id="sepay-environment" className="min-h-11 w-full cursor-pointer text-base md:text-sm">
                        <SelectValue placeholder="Chọn môi trường" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectGroup>
                          <SelectItem value="production">Live (giao dịch thật)</SelectItem>
                          <SelectItem value="sandbox">Test Mode (sandbox)</SelectItem>
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    <FieldDescription>Token Test Mode chỉ hoạt động với API sandbox; dữ liệu không ảnh hưởng môi trường Live.</FieldDescription>
                  </Field>
                )}
              />

              <Controller
                control={form.control}
                name="bankShortName"
                render={({ field }) => (
                  <Field data-invalid={Boolean(form.formState.errors.bankShortName)}>
                    <FieldLabel htmlFor="sepay-bank-short-name">Ngân hàng nhận tiền</FieldLabel>
                    <Select value={field.value || null} onValueChange={(value) => field.onChange(value ?? '')}>
                      <SelectTrigger id="sepay-bank-short-name" className="min-h-11 w-full cursor-pointer text-base md:text-sm" aria-invalid={Boolean(form.formState.errors.bankShortName)}>
                        <SelectValue placeholder="Chọn ngân hàng SePay hỗ trợ" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectGroup>
                          {SEPAY_SUPPORTED_BANKS.map((bank) => (
                            <SelectItem key={bank.bin} value={bank.shortName}>
                              {bank.shortName} ({bank.code})
                            </SelectItem>
                          ))}
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    <FieldDescription>Danh sách chỉ gồm ngân hàng có trạng thái hỗ trợ trong dữ liệu VietQR do SePay tham chiếu.</FieldDescription>
                    <FieldError>{form.formState.errors.bankShortName?.message}</FieldError>
                  </Field>
                )}
              />

              <Field data-invalid={Boolean(form.formState.errors.accountNumber)}>
                <FieldLabel htmlFor="sepay-account-number">Số tài khoản</FieldLabel>
                <Input id="sepay-account-number" placeholder="Nhập số tài khoản" inputMode="numeric" className="min-h-11 text-base md:text-sm" aria-invalid={Boolean(form.formState.errors.accountNumber)} {...form.register('accountNumber')} />
                <FieldError>{form.formState.errors.accountNumber?.message}</FieldError>
              </Field>

              <Field data-invalid={Boolean(form.formState.errors.accountName)}>
                <FieldLabel htmlFor="sepay-account-name">Tên chủ tài khoản</FieldLabel>
                <Input id="sepay-account-name" placeholder="Nhập tên chủ tài khoản" className="min-h-11 text-base md:text-sm" aria-invalid={Boolean(form.formState.errors.accountName)} {...form.register('accountName')} />
                <FieldDescription>Nếu có API Token, hệ thống sẽ đối chiếu tài khoản đã liên kết và dùng tên chính thức do SePay trả về.</FieldDescription>
                <FieldError>{form.formState.errors.accountName?.message}</FieldError>
              </Field>

              <Field data-invalid={Boolean(form.formState.errors.codePrefix)}>
                <FieldLabel htmlFor="sepay-code-prefix">Tiền tố mã thanh toán</FieldLabel>
                <Input id="sepay-code-prefix" placeholder="Nhập tiền tố mã, ví dụ: PH" autoCapitalize="characters" className="min-h-11 text-base uppercase md:text-sm" aria-invalid={Boolean(form.formState.errors.codePrefix)} {...form.register('codePrefix')} />
                <FieldError>{form.formState.errors.codePrefix?.message}</FieldError>
              </Field>

              <Controller
                control={form.control}
                name="webhookAuthMethod"
                render={({ field }) => (
                  <Field data-invalid={Boolean(form.formState.errors.webhookAuthMethod)}>
                    <FieldLabel htmlFor="sepay-webhook-auth-method">Phương thức xác thực webhook</FieldLabel>
                    <Select value={field.value || null} onValueChange={(value) => field.onChange(value ?? '')}>
                      <SelectTrigger id="sepay-webhook-auth-method" className="min-h-11 w-full cursor-pointer text-base md:text-sm" aria-invalid={Boolean(form.formState.errors.webhookAuthMethod)}>
                        <SelectValue placeholder="Chọn phương thức" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectGroup>
                          <SelectItem value="apikey">API Key</SelectItem>
                          <SelectItem value="hmac">HMAC-SHA256 (khuyến nghị)</SelectItem>
                          <SelectItem value="none">Không xác thực</SelectItem>
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    <FieldError>{form.formState.errors.webhookAuthMethod?.message}</FieldError>
                  </Field>
                )}
              />

              {authMethod === 'apikey' && (
                <Field data-invalid={Boolean(form.formState.errors.webhookApiKey)}>
                  <FieldLabel htmlFor="sepay-webhook-api-key">API Key webhook</FieldLabel>
                  <Input id="sepay-webhook-api-key" type={showSecrets ? 'text' : 'password'} placeholder="Nhập API Key webhook" autoComplete="off" className="min-h-11 text-base md:text-sm" aria-invalid={Boolean(form.formState.errors.webhookApiKey)} {...form.register('webhookApiKey')} />
                  <FieldError>{form.formState.errors.webhookApiKey?.message}</FieldError>
                </Field>
              )}

              {authMethod === 'hmac' && (
                <Field data-invalid={Boolean(form.formState.errors.webhookSecret)}>
                  <FieldLabel htmlFor="sepay-webhook-secret">Mã HMAC (Webhook Secret)</FieldLabel>
                  <Input id="sepay-webhook-secret" type={showSecrets ? 'text' : 'password'} placeholder="Nhập mã HMAC Secret từ SePay" autoComplete="off" className="min-h-11 text-base md:text-sm" aria-invalid={Boolean(form.formState.errors.webhookSecret)} {...form.register('webhookSecret')} />
                  <FieldDescription>Hệ thống xác thực raw body và từ chối chữ ký cũ quá 5 phút.</FieldDescription>
                  <FieldError>{form.formState.errors.webhookSecret?.message}</FieldError>
                </Field>
              )}

              <Field data-invalid={Boolean(form.formState.errors.apiToken)}>
                <FieldLabel htmlFor="sepay-api-token">API Token (tùy chọn, dùng cho đối soát)</FieldLabel>
                <Input id="sepay-api-token" type={showSecrets ? 'text' : 'password'} placeholder="Nhập API Token" autoComplete="off" className="min-h-11 text-base md:text-sm" aria-invalid={Boolean(form.formState.errors.apiToken)} {...form.register('apiToken')} />
                <FieldError>{form.formState.errors.apiToken?.message}</FieldError>
              </Field>

              <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap">
                <Button type="submit" disabled={form.formState.isSubmitting} className="min-h-11 cursor-pointer">
                  {form.formState.isSubmitting && <Loader2 data-icon="inline-start" className="animate-spin" />}
                  Lưu cấu hình
                </Button>
                <Button type="button" variant="outline" onClick={() => setShowSecrets((visible) => !visible)} className="min-h-11 cursor-pointer">
                  {showSecrets ? <EyeOff data-icon="inline-start" /> : <Eye data-icon="inline-start" />}
                  {showSecrets ? 'Ẩn khóa' : 'Hiện khóa'}
                </Button>
                {status?.has_config && (
                  <Button type="button" variant="outline" onClick={() => { setIsEditing(false); form.reset() }} disabled={form.formState.isSubmitting} className="min-h-11 cursor-pointer">
                    Hủy
                  </Button>
                )}
              </div>
            </FieldGroup>
          </form>
        ) : (
          <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap">
            <Button type="button" data-testid="sepay-edit-button" onClick={handleEdit} className="min-h-11 cursor-pointer">Cập nhật SePay</Button>
            <Button type="button" variant="outline" onClick={handleReconcile} disabled={reconciling} className="min-h-11 cursor-pointer">
              {reconciling ? <Loader2 data-icon="inline-start" className="animate-spin" /> : <RefreshCw data-icon="inline-start" />}
              Đối soát giao dịch
            </Button>
            <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
              <AlertDialogTrigger render={<Button type="button" variant="outline" className="min-h-11 cursor-pointer" />}>
                <Trash2 data-icon="inline-start" />
                Xóa cấu hình
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Xóa cấu hình SePay?</AlertDialogTitle>
                  <AlertDialogDescription>
                    Các QR SePay đang hoạt động sẽ bị đánh dấu hết hạn. Bạn có thể cấu hình lại bất cứ lúc nào.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel disabled={deleting} className="min-h-11 cursor-pointer">Hủy</AlertDialogCancel>
                  <AlertDialogAction variant="destructive" onClick={handleDelete} disabled={deleting} className="min-h-11 cursor-pointer">
                    {deleting && <Loader2 data-icon="inline-start" className="animate-spin" />}
                    Xóa cấu hình
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
