import { useState, useEffect } from 'react'
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
import { ProtectedFileImage } from './ProtectedFileImage'
import { ImageLightboxModal, type LightboxImageItem } from '@/components/shared/ImageLightboxModal'
import { ConfirmModal } from './ConfirmModal'
import { compressImagesForUpload } from '@/services/imageCompression'
import { toast } from 'sonner'

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
  onDataChange?: () => void
}

// Modal sửa thông tin người thuê. Tải ảnh CCCD tự động lưu ngay, xóa có xác nhận.
export function TenantEditModal({ room, tenant, onClose, onSuccess, onDataChange }: TenantEditModalProps) {
  const updateTenant = useTenantStore(state => state.updateTenant)

  const [gallery, setGallery] = useState<{ items: LightboxImageItem[]; initialIndex: number } | null>(null)
  const [existingCccdPaths, setExistingCccdPaths] = useState<string[]>([])
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isUploadingCccd, setIsUploadingCccd] = useState(false)
  const [cccdToDelete, setCccdToDelete] = useState<{ path: string; index: number } | null>(null)
  const [isDeletingCccd, setIsDeletingCccd] = useState(false)

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
  }, [tenant])

  const handleOpenGallery = (items: LightboxImageItem[], initialIndex: number) => {
    setGallery({ items, initialIndex })
  }

  const handleCloseGallery = () => {
    setGallery(null)
  }

  const cccdGallery: LightboxImageItem[] = existingCccdPaths.map((path, idx) => ({
    path,
    title: `Ảnh CCCD ${idx + 1}`,
    filename: getFileName(path),
  }))

  const handleUploadCccd = async (files: FileList | null) => {
    const incoming = files ? Array.from(files) : []
    if (incoming.length === 0) return

    setIsUploadingCccd(true)
    try {
      const compressed = await compressImagesForUpload(incoming)
      const formData = new FormData()
      formData.append('room_id', room.id)
      formData.append('kept_cccd_paths', existingCccdPaths.join(','))
      formData.append('kept_cccd_paths_empty', existingCccdPaths.length === 0 ? 'true' : 'false')
      compressed.forEach(item => {
        formData.append('cccd_file', item.file)
      })

      const res = await updateTenant(tenant.id, formData)
      if (res.success && res.data) {
        const newPaths = res.data.cccd_path ? res.data.cccd_path.split(',').filter(Boolean) : []
        setExistingCccdPaths(newPaths)
        toast.success('Tải lên và lưu ảnh CCCD thành công!')
        if (onDataChange) onDataChange()
      } else if (res.success) {
        toast.success('Tải lên và lưu ảnh CCCD thành công!')
        if (onDataChange) onDataChange()
      } else {
        toast.error(res.error || 'Lỗi khi tải lên ảnh CCCD!')
      }
    } catch (error) {
      console.error('Lỗi khi tải ảnh CCCD:', error)
      toast.error('Lỗi khi tải lên ảnh CCCD!')
    } finally {
      setIsUploadingCccd(false)
    }
  }

  const handleConfirmDeleteCccd = async () => {
    if (!cccdToDelete) return

    setIsDeletingCccd(true)
    try {
      const newPaths = existingCccdPaths.filter((_, i) => i !== cccdToDelete.index)
      const formData = new FormData()
      formData.append('room_id', room.id)
      formData.append('kept_cccd_paths', newPaths.join(','))
      formData.append('kept_cccd_paths_empty', newPaths.length === 0 ? 'true' : 'false')

      const res = await updateTenant(tenant.id, formData)
      if (res.success) {
        setExistingCccdPaths(newPaths)
        toast.success('Đã xóa ảnh CCCD thành công!')
        setCccdToDelete(null)
        if (onDataChange) onDataChange()
      } else {
        toast.error(res.error || 'Lỗi khi xóa ảnh CCCD!')
      }
    } catch (error) {
      console.error('Lỗi khi xóa ảnh CCCD:', error)
      toast.error('Lỗi khi xóa ảnh CCCD!')
    } finally {
      setIsDeletingCccd(false)
    }
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
      formData.append('kept_cccd_paths', existingCccdPaths.join(','))
      formData.append('kept_cccd_paths_empty', existingCccdPaths.length === 0 ? 'true' : 'false')

      const res = await updateTenant(tenant.id, formData)
      if (res.success) {
        toast.success('Cập nhật thông tin người thuê thành công!')
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
      <AppModal
        open
        onClose={onClose}
        title="Sửa thông tin người thuê"
        description={`Phòng ${room.name}`}
        contentClassName="sm:max-w-2xl"
        initialFocus={false}
        footer={
          <div className="flex justify-end gap-3 w-full">
            <Button type="button" variant="outline" onClick={onClose}>Hủy</Button>
            <Button type="submit" form="tenant-edit-form" disabled={isSubmitting || isUploadingCccd}>
              {isSubmitting ? 'Đang lưu...' : 'Lưu thay đổi'}
            </Button>
          </div>
        }
      >
        <form id="tenant-edit-form" onSubmit={handleSubmit(onSubmit)} className="space-y-4">
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
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-medium text-foreground">Tài liệu đính kèm</h3>
                  <p className="text-xs text-muted-foreground">Ảnh CCCD sẽ tự động lưu ngay khi tải lên hoặc xóa.</p>
                </div>
                {isUploadingCccd && (
                  <span className="flex items-center gap-1.5 text-xs text-primary font-medium">
                    <Loader2 className="w-3.5 h-3.5 animate-spin" /> Đang tải & lưu ảnh...
                  </span>
                )}
              </div>

              <div className="grid grid-cols-2 gap-4 max-h-[350px] overflow-y-auto pr-2">
                {/* CCCD Upload */}
                <div className="space-y-2 col-span-2">
                  <Label>Ảnh CCCD</Label>
                  {existingCccdPaths.length === 0 && !isUploadingCccd ? (
                    <label className="flex flex-col gap-2 items-center justify-center h-24 rounded-lg border-2 border-dashed border-border bg-muted/30 cursor-pointer hover:bg-secondary hover:border-primary/50 transition-colors">
                      <Upload className="w-5 h-5 text-muted-foreground" />
                      <span className="text-xs text-muted-foreground font-semibold px-4 text-center">Tải lên ảnh CCCD</span>
                      <input
                        type="file"
                        accept="image/*"
                        multiple
                        className="hidden"
                        onChange={e => {
                          void handleUploadCccd(e.target.files)
                          e.target.value = ''
                        }}
                      />
                    </label>
                  ) : (
                    <div className="flex flex-col gap-2">
                      {existingCccdPaths.map((path, idx) => {
                        const isImg = isImagePath(path)
                        return (
                          <div key={`exist-cccd-${idx}`} className="flex flex-col gap-2">
                            <div className="flex items-center justify-between h-10 px-3 rounded-lg border border-primary/20 bg-primary/10">
                              <div
                                className="flex items-center gap-2 overflow-hidden truncate cursor-pointer flex-1 min-w-0"
                                onClick={() => handleOpenGallery(cccdGallery, idx)}
                                title="Nhấn để xem ảnh"
                              >
                                <CheckCircle2 className="w-3.5 h-3.5 text-success flex-shrink-0" />
                                <span className="text-xs font-semibold text-primary truncate">{getFileName(path)}</span>
                              </div>
                              <div className="flex items-center gap-1 shrink-0">
                                <button
                                  type="button"
                                  onClick={() => handleOpenGallery(cccdGallery, idx)}
                                  className="p-1 hover:bg-primary/20 rounded-md text-primary cursor-pointer"
                                  title="Xem trước"
                                >
                                  <Eye className="w-4 h-4" />
                                </button>
                                <button
                                  type="button"
                                  onClick={() => setCccdToDelete({ path, index: idx })}
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
                                onClick={() => handleOpenGallery(cccdGallery, idx)}
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

                      {isUploadingCccd && (
                        <div className="flex items-center justify-center gap-2 p-4 rounded-lg border border-dashed border-primary/40 bg-primary/5 text-primary text-xs font-medium">
                          <Loader2 className="w-4 h-4 animate-spin" />
                          Đang tải lên & lưu ảnh CCCD...
                        </div>
                      )}

                      <label className="flex items-center justify-center gap-2 h-9 mt-1 rounded-lg border border-dashed border-border bg-muted/30 cursor-pointer hover:bg-secondary transition-colors">
                        <Upload className="w-4 h-4 text-muted-foreground" />
                        <span className="text-xs font-semibold text-foreground">Tải thêm ảnh CCCD</span>
                        <input
                          type="file"
                          accept="image/*"
                          multiple
                          disabled={isUploadingCccd}
                          className="hidden"
                          onChange={e => {
                            void handleUploadCccd(e.target.files)
                            e.target.value = ''
                          }}
                        />
                      </label>
                    </div>
                  )}
                </div>

              </div>
            </div>
          </div>
        </form>
      </AppModal>

      {/* Confirmation Dialog for Deleting CCCD Image */}
      {cccdToDelete && (
        <ConfirmModal
          title="Xác nhận xóa ảnh CCCD"
          message={`Bạn có chắc chắn muốn xóa ảnh "${getFileName(cccdToDelete.path)}"? Thay đổi sẽ được lưu ngay lập tức.`}
          confirmText="Xóa ảnh"
          cancelText="Hủy"
          isLoading={isDeletingCccd}
          onConfirm={handleConfirmDeleteCccd}
          onCancel={() => {
            if (!isDeletingCccd) setCccdToDelete(null)
          }}
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
