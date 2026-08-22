import { useState, useEffect, useMemo } from 'react'
import { AppModal } from '@/components/shared/AppModal'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Upload, X, ZoomIn, ChevronDown, ChevronUp, Loader2 } from '@/components/icons'
import type { Room } from '@/api/room'
import { useTenantStore } from '@/data/tenantData'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { ImageLightboxModal, type LightboxImageItem } from '@/components/shared/ImageLightboxModal'
import { useUploadFiles } from '@/hooks/useUploadFiles'
import { ConfirmModal } from './ConfirmModal'

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

  const [gallery, setGallery] = useState<{ items: LightboxImageItem[]; initialIndex: number } | null>(null)
  const [fileToDelete, setFileToDelete] = useState<{ name: string; index: number } | null>(null)
  const cccd = useUploadFiles()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const isOptimizing = cccd.isOptimizing
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

  const handleOpenGallery = (items: LightboxImageItem[], initialIndex: number) => {
    setGallery({ items, initialIndex })
  }

  const handleCloseGallery = () => {
    setGallery(null)
  }

  const cccdGallery: LightboxImageItem[] = cccd.files.map((file, idx) => ({
    file,
    title: `Ảnh CCCD ${idx + 1}`,
    filename: file.name,
  }))

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
      <AppModal
        open
        onClose={onClose}
        title="Thêm người thuê mới"
        description={`Phòng ${room.name}`}
        contentClassName="sm:max-w-2xl"
        footer={
          <>
            <Button type="button" variant="outline" onClick={onClose} className="flex-1 sm:flex-none">Hủy</Button>
            <Button type="submit" form="tenant-add-form" disabled={isSubmitting || isOptimizing} className="flex-1 sm:flex-none">
              {isOptimizing ? 'Đang tối ưu ảnh...' : isSubmitting ? 'Đang thêm...' : 'Xác nhận'}
            </Button>
          </>
        }
      >
        <form id="tenant-add-form" onSubmit={handleSubmit(onSubmit)} className="space-y-4">
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

                <div className="space-y-3 col-span-2 mt-2 p-4 rounded-lg border border-border/50 bg-card">
                  <h3 className="text-sm font-medium text-foreground">Ảnh CCCD (Tùy chọn)</h3>
                  <div className="flex flex-wrap gap-2.5 items-center">
                    {cccd.files.map((file, idx) => (
                      <div
                        key={`new-cccd-${idx}`}
                        className="relative group w-[100px] h-[100px] rounded-lg overflow-hidden border border-border bg-muted/40 cursor-pointer shadow-xs hover:border-primary/60 transition-all flex items-center justify-center shrink-0"
                        onClick={() => handleOpenGallery(cccdGallery, idx)}
                        title={`Ảnh CCCD ${idx + 1}: ${file.name}`}
                      >
                        <LocalFileThumbnail file={file} alt={file.name} className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-200" />
                        
                        {/* Hover Zoom Overlay */}
                        <div className="absolute inset-0 bg-black/30 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center text-white pointer-events-none">
                          <ZoomIn className="w-5 h-5" />
                        </div>

                        {/* Delete Button */}
                        <button
                          type="button"
                          disabled={isOptimizing}
                          onClick={(e) => {
                            e.stopPropagation()
                            setFileToDelete({ name: file.name, index: idx })
                          }}
                          className="absolute top-1 right-1 size-5 rounded-full bg-background/80 hover:bg-destructive text-muted-foreground hover:text-white backdrop-blur-xs flex items-center justify-center shadow transition-colors z-10 cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:bg-background/80 disabled:hover:text-muted-foreground"
                          title="Xóa ảnh"
                        >
                          <X className="w-3 h-3" />
                        </button>

                        {/* Index badge */}
                        <div className="absolute bottom-1 left-1 px-1.5 py-0.5 rounded bg-black/60 backdrop-blur-xs text-[9px] font-semibold text-white pointer-events-none">
                          CCCD {idx + 1}
                        </div>
                      </div>
                    ))}

                    {isOptimizing && (
                      <div className="w-[100px] h-[100px] rounded-lg border border-dashed border-primary/50 bg-primary/5 flex flex-col items-center justify-center gap-1 text-primary text-[10px] font-medium shrink-0 animate-pulse">
                        <Loader2 className="w-5 h-5 animate-spin" />
                        <span>Đang tối ưu...</span>
                      </div>
                    )}

                    {/* Add Button Tile */}
                    <label
                      className={`flex flex-col items-center justify-center w-[100px] h-[100px] rounded-lg border-2 border-dashed border-border/80 hover:border-primary hover:bg-primary/5 cursor-pointer text-muted-foreground hover:text-primary transition-all shrink-0 ${
                        isOptimizing ? 'opacity-50 pointer-events-none' : ''
                      }`}
                      title="Tải lên ảnh CCCD"
                    >
                      <Upload className="w-5 h-5 mb-1" />
                      <span className="text-[11px] font-medium text-center px-1 leading-tight">
                        {cccd.files.length === 0 ? 'Tải ảnh CCCD' : 'Thêm ảnh'}
                      </span>
                      <input
                        type="file"
                        accept="image/*"
                        multiple
                        disabled={isOptimizing}
                        className="hidden"
                        onChange={e => {
                          void cccd.addFiles(e.target.files)
                          e.target.value = ''
                        }}
                      />
                    </label>
                  </div>
                </div>
              </div>
            )}
          </div>
        </form>
      </AppModal>

      {/* Confirmation Dialog for Deleting Staged Image */}
      {fileToDelete && (
        <ConfirmModal
          title="Xác nhận xóa ảnh"
          message={`Bạn có chắc chắn muốn xóa ảnh "${fileToDelete.name}" không?`}
          confirmText="Xóa ảnh"
          cancelText="Hủy"
          onConfirm={() => {
            cccd.removeFile(fileToDelete.index)
            setFileToDelete(null)
          }}
          onCancel={() => setFileToDelete(null)}
        />
      )}

      {/* Lightbox / Gallery Modal - Portaled to document.body */}
      <ImageLightboxModal
        isOpen={Boolean(gallery)}
        images={gallery?.items}
        initialIndex={gallery?.initialIndex ?? 0}
        onClose={handleCloseGallery}
      />
    </>
  )
}
