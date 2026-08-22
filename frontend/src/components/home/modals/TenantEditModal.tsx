import { useState, useEffect } from 'react'
import { AppModal } from '@/components/shared/AppModal'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Upload, X, FileIcon, ZoomIn, Loader2 } from '@/components/icons'
import type { Room } from '@/api/room'
import type { Tenant } from '@/api/tenant'
import { useTenantStore } from '@/data/tenantData'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { getFileName, isImagePath } from '@/utils/file'
import { ProtectedFileImage } from './ProtectedFileImage'
import { clearProtectedFileCache } from '@/api/files'
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
  const [cccdToDelete, setCccdToDelete] = useState<string | null>(null)
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
      const newPaths = existingCccdPaths.filter(p => p !== cccdToDelete)
      const formData = new FormData()
      formData.append('room_id', room.id)
      formData.append('kept_cccd_paths', newPaths.join(','))
      formData.append('kept_cccd_paths_empty', newPaths.length === 0 ? 'true' : 'false')

      const res = await updateTenant(tenant.id, formData)
      if (res.success) {
        setExistingCccdPaths(newPaths)
        clearProtectedFileCache(cccdToDelete)
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
          <>
            <Button type="button" variant="outline" onClick={onClose} className="flex-1 sm:flex-none">Hủy</Button>
            <Button type="submit" form="tenant-edit-form" disabled={isSubmitting || isUploadingCccd} className="flex-1 sm:flex-none">
              {isSubmitting ? 'Đang lưu...' : 'Lưu thay đổi'}
            </Button>
          </>
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

            <div className="space-y-3 col-span-2 mt-2 p-4 rounded-lg border border-border/50 bg-card">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-medium text-foreground">Ảnh CCCD</h3>
                  <p className="text-xs text-muted-foreground">Ảnh CCCD sẽ tự động lưu ngay khi tải lên hoặc xóa.</p>
                </div>
              </div>

              <div className="flex flex-wrap gap-2.5 items-center">
                {existingCccdPaths.map((path, idx) => {
                  const isImg = isImagePath(path)
                  const fileName = getFileName(path)
                  return (
                    <div
                      key={`exist-cccd-${idx}`}
                      className="relative group w-[100px] h-[100px] rounded-lg overflow-hidden border border-border bg-muted/40 cursor-pointer shadow-xs hover:border-primary/60 transition-all flex items-center justify-center shrink-0"
                      onClick={() => handleOpenGallery(cccdGallery, idx)}
                      title={`Ảnh CCCD ${idx + 1}: ${fileName}`}
                    >
                      {isImg ? (
                        <ProtectedFileImage
                          path={path}
                          alt={fileName}
                          className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-200"
                        />
                      ) : (
                        <div className="flex flex-col items-center justify-center p-2 text-center text-muted-foreground">
                          <FileIcon className="w-6 h-6 mb-1 text-primary/70" />
                          <span className="text-[10px] font-medium leading-tight line-clamp-2 break-all">{fileName}</span>
                        </div>
                      )}

                      {/* Hover Zoom Overlay */}
                      <div className="absolute inset-0 bg-black/30 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center text-white pointer-events-none">
                        <ZoomIn className="w-5 h-5" />
                      </div>

                      {/* Delete Button */}
                      <button
                        type="button"
                        disabled={isUploadingCccd || isDeletingCccd}
                        onClick={(e) => {
                          e.stopPropagation()
                          setCccdToDelete(path)
                        }}
                        className="absolute top-1 right-1 size-5 rounded-full bg-background/80 hover:bg-destructive text-muted-foreground hover:text-white backdrop-blur-xs flex items-center justify-center shadow transition-colors z-10 cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:bg-background/80 disabled:hover:text-muted-foreground"
                        title="Xóa ảnh CCCD"
                      >
                        <X className="w-3 h-3" />
                      </button>

                      {/* Index badge */}
                      <div className="absolute bottom-1 left-1 px-1.5 py-0.5 rounded bg-black/60 backdrop-blur-xs text-[9px] font-semibold text-white pointer-events-none">
                        CCCD {idx + 1}
                      </div>
                    </div>
                  )
                })}

                {isUploadingCccd && (
                  <div className="w-[100px] h-[100px] rounded-lg border border-dashed border-primary/50 bg-primary/5 flex flex-col items-center justify-center gap-1 text-primary text-[10px] font-medium shrink-0 animate-pulse">
                    <Loader2 className="w-5 h-5 animate-spin" />
                    <span>Đang lưu...</span>
                  </div>
                )}

                {/* Add Button Tile */}
                <label
                  className={`flex flex-col items-center justify-center w-[100px] h-[100px] rounded-lg border-2 border-dashed border-border/80 hover:border-primary hover:bg-primary/5 cursor-pointer text-muted-foreground hover:text-primary transition-all shrink-0 ${
                    isUploadingCccd ? 'opacity-50 pointer-events-none' : ''
                  }`}
                  title="Tải lên thêm ảnh CCCD"
                >
                  <Upload className="w-5 h-5 mb-1" />
                  <span className="text-[11px] font-medium text-center px-1 leading-tight">
                    {existingCccdPaths.length === 0 ? 'Tải ảnh CCCD' : 'Thêm ảnh'}
                  </span>
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
            </div>
          </div>
        </form>
      </AppModal>

      {/* Confirmation Dialog for Deleting CCCD Image */}
      {cccdToDelete && (
        <ConfirmModal
          title="Xác nhận xóa ảnh CCCD"
          message={`Bạn có chắc chắn muốn xóa ảnh "${getFileName(cccdToDelete)}"? Thay đổi sẽ được lưu ngay lập tức.`}
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
