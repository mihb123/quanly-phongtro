import { useState, useEffect, useMemo } from 'react'
import { AppModal } from '@/components/shared/AppModal'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Upload, CheckCircle2, X, FileIcon, ZoomIn, Eye, ChevronDown, ChevronUp } from '@/components/icons'
import type { Room } from '@/api/room'
import { useTenantStore } from '@/data/tenantData'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { ImageLightboxModal } from '@/components/shared/ImageLightboxModal'
import { useUploadFiles } from '@/hooks/useUploadFiles'

const tenantSchema = z.object({
  fullName: z.string().min(1, 'Bắt buộc'),
  phone: z.string().optional().or(z.literal('')),
  email: z.string().email('Email không hợp lệ').optional().or(z.literal('')),
  identityCard: z.string().optional().or(z.literal('')),
  startDate: z.string().optional().or(z.literal('')),
})

type TenantFormValues = z.infer<typeof tenantSchema>

interface TenantAddModalProps {
  room: Room
  onClose: () => void
  onSuccess: () => void
}

function LocalFileThumbnail({ file, alt, className }: { file: File; alt: string; className?: string }) {
  const url = useMemo(() => URL.createObjectURL(file), [file])
  useEffect(() => () => URL.revokeObjectURL(url), [url])

  return <img src={url} alt={alt} className={className} />
}

// Modal thêm người thuê mới. Vỏ dùng AppModal; giữ nguyên form RHF/Zod, upload ảnh CCCD/hợp đồng và ImageLightboxModal xem trước.
export function TenantAddModal({ room, onClose, onSuccess }: TenantAddModalProps) {
  const createTenant = useTenantStore(state => state.createTenant)

  const [previewImage, setPreviewImage] = useState<{ url: string; title: string; filename: string } | null>(null)
  const cccd = useUploadFiles()
  const contract = useUploadFiles()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const isOptimizing = cccd.isOptimizing || contract.isOptimizing
  const [showOptionalFields, setShowOptionalFields] = useState(false)

  const { register, handleSubmit, formState: { errors } } = useForm<TenantFormValues>({
    resolver: zodResolver(tenantSchema),
    defaultValues: {
      fullName: '',
      phone: '',
      email: '',
      identityCard: '',
      startDate: new Date().toISOString().split('T')[0]
    }
  })

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
      if (values.phone) formData.append('phone', values.phone)
      if (values.email) formData.append('email', values.email)
      if (values.identityCard) formData.append('identity_card', values.identityCard)
      if (values.startDate) formData.append('start_date', values.startDate)

      cccd.files.forEach(f => formData.append('cccd_file', f))
      contract.files.forEach(f => formData.append('contract_file', f))

      const res = await createTenant(formData)
      if (res.success) {
        onSuccess()
      } else {
        alert(res.error || "Lỗi khi thêm người thuê!")
      }
    } catch {
      alert("Lỗi khi thêm người thuê!")
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <>
      <AppModal open onClose={onClose} title="Thêm người thuê mới" description={`Phòng ${room.name}`} contentClassName="sm:max-w-2xl">
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Họ và tên <span className="text-destructive">*</span></Label>
              <Input {...register('fullName')} placeholder="Nguyễn Văn A" className="border-border bg-background" />
              {errors.fullName && <span className="text-destructive text-xs">{errors.fullName.message}</span>}
            </div>

            <div className="pt-2">
              <button
                type="button"
                onClick={() => setShowOptionalFields(!showOptionalFields)}
                className="w-full flex items-center justify-between py-3 px-4 rounded-lg border border-border/50 bg-muted/30 hover:bg-secondary transition-all font-medium text-sm text-foreground select-none cursor-pointer"
              >
                <span className="flex items-center gap-2">
                  ⚙️ Điền thêm thông tin khác (Tùy chọn)
                </span>
                {showOptionalFields ? <ChevronUp className="w-4 h-4 text-muted-foreground" /> : <ChevronDown className="w-4 h-4 text-muted-foreground" />}
              </button>
            </div>

            {showOptionalFields && (
              <div className="grid grid-cols-2 gap-4 safe-fade-in">
                <div className="space-y-2 col-span-2 md:col-span-1">
                  <Label>Số điện thoại (Tùy chọn)</Label>
                  <Input {...register('phone')} placeholder="09..." className="border-border bg-background" />
                  {errors.phone && <span className="text-destructive text-xs">{errors.phone.message}</span>}
                </div>
                <div className="space-y-2 col-span-2 md:col-span-1">
                  <Label>Email (Tùy chọn)</Label>
                  <Input type="email" {...register('email')} placeholder="abc@gmail.com" className="border-border bg-background" />
                  {errors.email && <span className="text-destructive text-xs">{errors.email.message}</span>}
                </div>
                <div className="space-y-2 col-span-2 md:col-span-1">
                  <Label>Căn cước công dân (Tùy chọn)</Label>
                  <Input {...register('identityCard')} placeholder="12 số CCCD" className="border-border bg-background" />
                  {errors.identityCard && <span className="text-destructive text-xs">{errors.identityCard.message}</span>}
                </div>
                <div className="space-y-2 col-span-2 md:col-span-1">
                  <Label>Ngày bắt đầu thuê (Tùy chọn)</Label>
                  <Input type="date" {...register('startDate')} className="border-border bg-background" />
                  {errors.startDate && <span className="text-destructive text-xs">{errors.startDate.message}</span>}
                </div>

                <div className="space-y-4 col-span-2 mt-4 p-4 rounded-lg border border-border/50 bg-card">
                  <h3 className="text-sm font-medium text-foreground mb-2">Tài liệu đính kèm</h3>
                  <div className="grid grid-cols-2 gap-4 max-h-[350px] overflow-y-auto pr-2">
                    {/* CCCD Upload */}
                    <div className="space-y-2 col-span-2 md:col-span-1">
                      <Label>Ảnh CCCD (Tùy chọn)</Label>
                      {cccd.files.length === 0 ? (
                        <label className="flex flex-col gap-2 items-center justify-center h-24 rounded-lg border-2 border-dashed border-border bg-muted/30 cursor-pointer hover:bg-secondary hover:border-primary/50 transition-colors">
                          <Upload className="w-5 h-5 text-muted-foreground" />
                          <span className="text-xs text-muted-foreground font-semibold px-4 text-center">Tải lên ảnh CCCD</span>
                          <input type="file" accept="image/*" multiple className="hidden" onChange={e => { void cccd.addFiles(e.target.files); e.target.value = '' }} />
                        </label>
                      ) : (
                        <div className="flex flex-col gap-2">
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
                      <Label>Hợp đồng (Tùy chọn)</Label>
                      {contract.files.length === 0 ? (
                        <label className="flex flex-col gap-2 items-center justify-center h-24 rounded-lg border-2 border-dashed border-border bg-muted/30 cursor-pointer hover:bg-secondary hover:border-primary/50 transition-colors">
                          <Upload className="w-5 h-5 text-muted-foreground" />
                          <span className="text-xs text-muted-foreground font-semibold px-4 text-center">Tải lên hợp đồng</span>
                          <input type="file" accept=".pdf,.doc,.docx,image/*" multiple className="hidden" onChange={e => { void contract.addFiles(e.target.files); e.target.value = '' }} />
                        </label>
                      ) : (
                        <div className="flex flex-col gap-2">
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
            )}
          </div>

          <div className="flex justify-end gap-3 pt-6 border-t border-border/40">
            <Button type="button" variant="outline" onClick={onClose}>Hủy</Button>
            <Button type="submit" disabled={isSubmitting || isOptimizing}>
              {isOptimizing ? 'Đang tối ưu ảnh...' : isSubmitting ? 'Đang thêm...' : 'Xác nhận'}
            </Button>
          </div>
        </form>
      </AppModal>

      {/* Lightbox / Gallery Modal - Portaled to document.body */}
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
