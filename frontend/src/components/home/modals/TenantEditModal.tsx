import { useState, useEffect, useMemo } from 'react'
import { AppModal } from '@/components/shared/AppModal'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Upload, CheckCircle2, X, FileIcon, ZoomIn, Eye, Loader2 } from '@/components/icons'
import type { Room } from '@/api/room'
import type { Tenant } from '@/api/tenant'
import { useTenantStore } from '@/data/tenantData'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { getFileName, isImagePath } from '@/utils/file'
import { getProtectedFileObjectUrl, openProtectedFile } from '@/api/files'
import { ProtectedFileImage } from './ProtectedFileImage'
import { ImageLightboxModal } from '@/components/shared/ImageLightboxModal'
import { toast } from 'sonner'
import { useUploadFiles } from '@/hooks/useUploadFiles'

const tenantSchema = z.object({
  fullName: z.string().min(1, 'Bắt buộc'),
  phone: z.string().optional().or(z.literal('')),
  email: z.string().email('Email không hợp lệ').optional().or(z.literal('')),
  identityCard: z.string().optional().or(z.literal('')),
  startDate: z.string(),
})

type TenantFormValues = z.infer<typeof tenantSchema>

interface TenantEditModalProps {
  room: Room
  tenant: Tenant
  onClose: () => void
  onSuccess: () => void
}

function LocalFileThumbnail({ file, alt, className }: { file: File; alt: string; className?: string }) {
  const url = useMemo(() => URL.createObjectURL(file), [file])
  useEffect(() => () => URL.revokeObjectURL(url), [url])

  return <img src={url} alt={alt} className={className} />
}

// Modal sửa thông tin người thuê. Vỏ dùng AppModal; giữ nguyên form RHF/Zod, quản lý file cũ/mới và ImageLightboxModal xem trước.
export function TenantEditModal({ room, tenant, onClose, onSuccess }: TenantEditModalProps) {
  const updateTenant = useTenantStore(state => state.updateTenant)

  const [previewImage, setPreviewImage] = useState<{ url: string; title: string; filename: string } | null>(null)
  const cccd = useUploadFiles()
  const contract = useUploadFiles()
  const [existingCccdPaths, setExistingCccdPaths] = useState<string[]>([])
  const [existingContractPaths, setExistingContractPaths] = useState<string[]>([])
  const [isSubmitting, setIsSubmitting] = useState(false)
  const isOptimizing = cccd.isOptimizing || contract.isOptimizing
  const [loadingFilePath, setLoadingFilePath] = useState<string | null>(null)

  const { register, handleSubmit, formState: { errors } } = useForm<TenantFormValues>({
    resolver: zodResolver(tenantSchema),
    defaultValues: {
      fullName: tenant.full_name,
      phone: tenant.phone,
      email: tenant.email || '',
      identityCard: tenant.identity_card,
      startDate: tenant.start_date.split('T')[0]
    }
  })

  useEffect(() => {
    setExistingCccdPaths(tenant.cccd_path ? tenant.cccd_path.split(',').filter(Boolean) : [])
    setExistingContractPaths(tenant.contract_path ? tenant.contract_path.split(',').filter(Boolean) : [])
  }, [tenant])

  useEffect(() => {
    return () => {
      if (previewImage?.url.startsWith('blob:')) {
        URL.revokeObjectURL(previewImage.url)
      }
    }
  }, [previewImage])

  const handleClosePreview = () => {
    if (previewImage?.url.startsWith('blob:')) {
      URL.revokeObjectURL(previewImage.url)
    }
    setPreviewImage(null)
  }

  const handleExistingImagePreview = async (path: string, label?: string) => {
    setLoadingFilePath(path)
    try {
      if (isImagePath(path)) {
        const objectUrl = await getProtectedFileObjectUrl(path)
        setPreviewImage((current) => {
          if (current?.url.startsWith('blob:')) URL.revokeObjectURL(current.url)
          return {
            url: objectUrl,
            title: label ? `${label} - ${getFileName(path)}` : getFileName(path),
            filename: getFileName(path),
          }
        })
      } else {
        await openProtectedFile(path)
      }
    } catch (error) {
      console.error('Không tải được file tenant:', error)
      toast.error('Không thể tải file, vui lòng thử lại!')
    } finally {
      setLoadingFilePath(null)
    }
  }

  const handlePreviewLocalFile = (file: File, label: string) => {
    const url = URL.createObjectURL(file)
    setPreviewImage((current) => {
      if (current?.url.startsWith('blob:')) URL.revokeObjectURL(current.url)
      return {
        url,
        title: `${label} - ${file.name}`,
        filename: file.name,
      }
    })
  }

  const onSubmit = async (values: TenantFormValues) => {
    setIsSubmitting(true)
    try {
      const formData = new FormData()
      formData.append('room_id', room.id)
      formData.append('full_name', values.fullName)
      formData.append('phone', values.phone || '')
      if (values.email) formData.append('email', values.email)
      formData.append('identity_card', values.identityCard || '')
      formData.append('start_date', values.startDate)

      cccd.files.forEach(f => formData.append('cccd_file', f))
      contract.files.forEach(f => formData.append('contract_file', f))

      formData.append('kept_cccd_paths', existingCccdPaths.join(','))
      formData.append('kept_cccd_paths_empty', existingCccdPaths.length === 0 ? 'true' : 'false')
      formData.append('kept_contract_paths', existingContractPaths.join(','))
      formData.append('kept_contract_paths_empty', existingContractPaths.length === 0 ? 'true' : 'false')

      const res = await updateTenant(tenant.id, formData)
      if (res.success) {
        onSuccess()
      } else {
        alert(res.error || "Lỗi khi cập nhật thông tin người thuê!")
      }
    } catch {
      alert("Lỗi khi cập nhật thông tin người thuê!")
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <>
      <AppModal open onClose={onClose} title="Sửa thông tin người thuê" description={`Phòng ${room.name}`} contentClassName="sm:max-w-2xl">
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2 col-span-2">
              <Label>Họ và tên <span className="text-destructive">*</span></Label>
              <Input {...register('fullName')} placeholder="Nguyễn Văn A" className="border-border bg-background" />
              {errors.fullName && <span className="text-destructive text-xs">{errors.fullName.message}</span>}
            </div>
            <div className="space-y-2 col-span-2 md:col-span-1">
              <Label>Số điện thoại</Label>
              <Input {...register('phone')} placeholder="09..." className="border-border bg-background" />
              {errors.phone && <span className="text-destructive text-xs">{errors.phone.message}</span>}
            </div>
            <div className="space-y-2 col-span-2 md:col-span-1">
              <Label>Email</Label>
              <Input type="email" {...register('email')} placeholder="abc@gmail.com" className="border-border bg-background" />
              {errors.email && <span className="text-destructive text-xs">{errors.email.message}</span>}
            </div>
            <div className="space-y-2 col-span-2 md:col-span-1">
              <Label>Căn cước công dân</Label>
              <Input {...register('identityCard')} placeholder="12 số CCCD" className="border-border bg-background" />
              {errors.identityCard && <span className="text-destructive text-xs">{errors.identityCard.message}</span>}
            </div>
            <div className="space-y-2 col-span-2 md:col-span-1">
              <Label>Ngày bắt đầu thuê</Label>
              <Input type="date" {...register('startDate')} className="border-border bg-background" />
              {errors.startDate && <span className="text-destructive text-xs">{errors.startDate.message}</span>}
            </div>

            <div className="space-y-4 col-span-2 mt-4 p-4 rounded-lg border border-border/50 bg-card">
              <h3 className="text-sm font-medium text-foreground mb-2">Tài liệu đính kèm</h3>
              <div className="grid grid-cols-2 gap-4 max-h-[350px] overflow-y-auto pr-2">
                {/* CCCD Upload */}
                <div className="space-y-2 col-span-2 md:col-span-1">
                  <Label>Ảnh CCCD</Label>
                  {existingCccdPaths.length === 0 && cccd.files.length === 0 ? (
                    <label className="flex flex-col gap-2 items-center justify-center h-24 rounded-lg border-2 border-dashed border-border bg-muted/30 cursor-pointer hover:bg-secondary hover:border-primary/50 transition-colors">
                      <Upload className="w-5 h-5 text-muted-foreground" />
                      <span className="text-xs text-muted-foreground font-semibold px-4 text-center">Tải lên ảnh CCCD</span>
                      <input type="file" accept="image/*" multiple className="hidden" onChange={e => { void cccd.addFiles(e.target.files); e.target.value = '' }} />
                    </label>
                  ) : (
                    <div className="flex flex-col gap-2">
                      {existingCccdPaths.map((path, idx) => {
                        const label = `Ảnh CCCD ${idx > 0 ? idx + 1 : ''}`.trim()
                        const isImg = isImagePath(path)
                        const isLoading = loadingFilePath === path
                        return (
                          <div key={`exist-cccd-${idx}`} className="flex flex-col gap-2">
                            <div className="flex items-center justify-between h-10 px-3 rounded-lg border border-primary/20 bg-primary/10">
                              <div className="flex items-center gap-2 overflow-hidden truncate">
                                <CheckCircle2 className="w-3.5 h-3.5 text-success flex-shrink-0" />
                                <span className="text-xs font-semibold text-primary truncate">{getFileName(path)}</span>
                              </div>
                              <div className="flex items-center gap-1">
                                <button
                                  type="button"
                                  onClick={() => handleExistingImagePreview(path, label)}
                                  disabled={isLoading}
                                  className="p-1 hover:bg-primary/20 rounded-md text-primary cursor-pointer disabled:opacity-50"
                                  title="Xem trước"
                                >
                                  {isLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Eye className="w-4 h-4" />}
                                </button>
                                <button
                                  type="button"
                                  onClick={() => setExistingCccdPaths(prev => prev.filter((_, i) => i !== idx))}
                                  className="p-1 hover:bg-destructive/10 rounded-md text-muted-foreground hover:text-destructive cursor-pointer"
                                  title="Xóa"
                                >
                                  <X className="w-3.5 h-3.5" />
                                </button>
                              </div>
                            </div>
                            {isImg ? (
                              <div
                                className="relative rounded-lg overflow-hidden border border-border aspect-video bg-muted cursor-pointer group"
                                onClick={() => handleExistingImagePreview(path, label)}
                              >
                                <ProtectedFileImage path={path} alt={getFileName(path)} className="w-full h-full object-cover" />
                                <div className="absolute inset-0 bg-black/20 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center text-white">
                                  <ZoomIn className="w-5 h-5" />
                                </div>
                              </div>
                            ) : (
                              <div className="rounded-lg border border-border/50 aspect-video bg-muted/30 flex flex-col items-center justify-center text-muted-foreground">
                                <FileIcon className="w-8 h-8" />
                                <span className="text-xs font-medium">FILE TÀI LIỆU</span>
                              </div>
                            )}
                          </div>
                        )
                      })}
                      {cccd.files.map((file, idx) => (
                        <div key={`new-cccd-${idx}`} className="flex flex-col gap-2">
                          <div className="flex items-center justify-between h-10 px-3 rounded-lg border border-primary/20 bg-primary/10">
                            <div className="flex items-center gap-2 overflow-hidden truncate">
                              <CheckCircle2 className="w-3.5 h-3.5 text-success flex-shrink-0" />
                              <span className="text-xs font-semibold text-primary truncate">{file.name}</span>
                            </div>
                            <div className="flex items-center gap-1">
                              <button type="button" onClick={() => handlePreviewLocalFile(file, 'Ảnh CCCD')} className="p-1 hover:bg-primary/20 rounded-md text-primary cursor-pointer" title="Xem trước">
                                <Eye className="w-4 h-4" />
                              </button>
                              <button type="button" onClick={() => cccd.removeFile(idx)} className="p-1 hover:bg-destructive/10 rounded-md text-muted-foreground hover:text-destructive cursor-pointer" title="Xóa">
                                <X className="w-3.5 h-3.5" />
                              </button>
                            </div>
                          </div>
                          <div
                            className="relative rounded-lg overflow-hidden border border-border aspect-video bg-muted cursor-pointer group"
                            onClick={() => handlePreviewLocalFile(file, 'Ảnh CCCD')}
                          >
                            <LocalFileThumbnail file={file} alt={file.name} className="w-full h-full object-cover" />
                            <div className="absolute inset-0 bg-black/20 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center text-white">
                              <ZoomIn className="w-5 h-5" />
                            </div>
                          </div>
                        </div>
                      ))}
                      <label className="flex items-center justify-center gap-2 h-9 mt-1 rounded-lg border border-dashed border-border bg-muted/30 cursor-pointer hover:bg-secondary transition-colors">
                        <Upload className="w-4 h-4 text-muted-foreground" />
                        <span className="text-xs font-semibold text-foreground">Tải ảnh khác</span>
                        <input type="file" accept="image/*" multiple className="hidden" onChange={e => { void cccd.addFiles(e.target.files); e.target.value = '' }} />
                      </label>
                    </div>
                  )}
                </div>

                {/* Contract Upload */}
                <div className="space-y-2 col-span-2 md:col-span-1">
                  <Label>Hợp đồng</Label>
                  {existingContractPaths.length === 0 && contract.files.length === 0 ? (
                    <label className="flex flex-col gap-2 items-center justify-center h-24 rounded-lg border-2 border-dashed border-border bg-muted/30 cursor-pointer hover:bg-secondary hover:border-primary/50 transition-colors">
                      <Upload className="w-5 h-5 text-muted-foreground" />
                      <span className="text-xs text-muted-foreground font-semibold px-4 text-center">Tải lên hợp đồng</span>
                      <input type="file" accept=".pdf,.doc,.docx,image/*" multiple className="hidden" onChange={e => { void contract.addFiles(e.target.files); e.target.value = '' }} />
                    </label>
                  ) : (
                    <div className="flex flex-col gap-2">
                      {existingContractPaths.map((path, idx) => {
                        const label = `Hợp đồng ${idx > 0 ? idx + 1 : ''}`.trim()
                        const isImg = isImagePath(path)
                        const isLoading = loadingFilePath === path
                        return (
                          <div key={`exist-contract-${idx}`} className="flex flex-col gap-2">
                            <div className="flex items-center justify-between h-10 px-3 rounded-lg border border-primary/20 bg-primary/10">
                              <div className="flex items-center gap-2 overflow-hidden truncate">
                                <CheckCircle2 className="w-3.5 h-3.5 text-success flex-shrink-0" />
                                <span className="text-xs font-semibold text-primary truncate">{getFileName(path)}</span>
                              </div>
                              <div className="flex items-center gap-1">
                                {isImg && (
                                  <button
                                    type="button"
                                    onClick={() => handleExistingImagePreview(path, label)}
                                    disabled={isLoading}
                                    className="p-1 hover:bg-primary/20 rounded-md text-primary cursor-pointer disabled:opacity-50"
                                    title="Xem trước"
                                  >
                                    {isLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Eye className="w-4 h-4" />}
                                  </button>
                                )}
                                <button
                                  type="button"
                                  onClick={() => setExistingContractPaths(prev => prev.filter((_, i) => i !== idx))}
                                  className="p-1 hover:bg-destructive/10 rounded-md text-muted-foreground hover:text-destructive cursor-pointer"
                                  title="Xóa"
                                >
                                  <X className="w-3.5 h-3.5" />
                                </button>
                              </div>
                            </div>
                            {isImg ? (
                              <div
                                className="relative rounded-lg overflow-hidden border border-border aspect-video bg-muted cursor-pointer group"
                                onClick={() => handleExistingImagePreview(path, label)}
                              >
                                <ProtectedFileImage path={path} alt={getFileName(path)} className="w-full h-full object-cover" />
                                <div className="absolute inset-0 bg-black/20 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center text-white">
                                  <ZoomIn className="w-5 h-5" />
                                </div>
                              </div>
                            ) : (
                              <div className="rounded-lg border border-border/50 aspect-video bg-muted/30 flex flex-col items-center justify-center text-muted-foreground">
                                <FileIcon className="w-8 h-8" />
                                <span className="text-xs font-medium">FILE TÀI LIỆU</span>
                              </div>
                            )}
                          </div>
                        )
                      })}
                      {contract.files.map((file, idx) => (
                        <div key={`new-contract-${idx}`} className="flex flex-col gap-2">
                          <div className="flex items-center justify-between h-10 px-3 rounded-lg border border-primary/20 bg-primary/10">
                            <div className="flex items-center gap-2 overflow-hidden truncate">
                              <CheckCircle2 className="w-3.5 h-3.5 text-success flex-shrink-0" />
                              <span className="text-xs font-semibold text-primary truncate">{file.name}</span>
                            </div>
                            <div className="flex items-center gap-1">
                              {file.type.startsWith('image/') && (
                                <button type="button" onClick={() => handlePreviewLocalFile(file, 'Hợp đồng')} className="p-1 hover:bg-primary/20 rounded-md text-primary cursor-pointer" title="Xem trước">
                                  <Eye className="w-4 h-4" />
                                </button>
                              )}
                              <button type="button" onClick={() => contract.removeFile(idx)} className="p-1 hover:bg-destructive/10 rounded-md text-muted-foreground hover:text-destructive cursor-pointer" title="Xóa">
                                <X className="w-3.5 h-3.5" />
                              </button>
                            </div>
                          </div>
                          {file.type.startsWith('image/') ? (
                            <div
                              className="relative rounded-lg overflow-hidden border border-border aspect-video bg-muted cursor-pointer group"
                              onClick={() => handlePreviewLocalFile(file, 'Hợp đồng')}
                            >
                              <LocalFileThumbnail file={file} alt={file.name} className="w-full h-full object-cover" />
                              <div className="absolute inset-0 bg-black/20 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center text-white">
                                <ZoomIn className="w-5 h-5" />
                              </div>
                            </div>
                          ) : (
                            <div className="rounded-lg border border-border/50 aspect-video bg-muted/30 flex flex-col items-center justify-center text-muted-foreground">
                              <FileIcon className="w-8 h-8" />
                              <span className="text-xs font-medium">FILE TÀI LIỆU</span>
                            </div>
                          )}
                        </div>
                      ))}
                      <label className="flex items-center justify-center gap-2 h-9 mt-1 rounded-lg border border-dashed border-border bg-muted/30 cursor-pointer hover:bg-secondary transition-colors">
                        <Upload className="w-4 h-4 text-muted-foreground" />
                        <span className="text-xs font-semibold text-foreground">Tải file khác</span>
                        <input type="file" accept=".pdf,.doc,.docx,image/*" multiple className="hidden" onChange={e => { void contract.addFiles(e.target.files); e.target.value = '' }} />
                      </label>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>

          <div className="flex justify-end gap-3 pt-6 border-t border-border/40">
            <Button type="button" variant="outline" onClick={onClose}>Hủy</Button>
            <Button type="submit" disabled={isSubmitting || isOptimizing}>
              {isOptimizing ? 'Đang tối ưu ảnh...' : isSubmitting ? 'Đang lưu...' : 'Lưu thay đổi'}
            </Button>
          </div>
        </form>
      </AppModal>

      <ImageLightboxModal
        isOpen={Boolean(previewImage)}
        imageUrl={previewImage?.url || null}
        title={previewImage?.title}
        filename={previewImage?.filename}
        onClose={handleClosePreview}
      />
    </>
  )
}
