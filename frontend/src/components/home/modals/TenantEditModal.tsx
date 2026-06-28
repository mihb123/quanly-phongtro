import { useState, useEffect } from 'react'
import { AppModal } from '@/components/shared/AppModal'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Upload, CheckCircle2, X, FileIcon, ZoomIn, Eye } from '@/components/icons'
import type { Room } from '@/api/room'
import type { Tenant } from '@/api/tenant'
import { useTenantStore } from '@/data/tenantData'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { getFileName, isImagePath } from '@/utils/file'
import { getProtectedFileObjectUrl } from '@/api/files'
import { ProtectedFileImage } from './ProtectedFileImage'

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

// Modal sửa thông tin người thuê. Vỏ dùng AppModal; giữ nguyên form RHF/Zod, quản lý file cũ/mới và lightbox xem trước thủ công.
export function TenantEditModal({ room, tenant, onClose, onSuccess }: TenantEditModalProps) {
  const updateTenant = useTenantStore(state => state.updateTenant)

  const [selectedImageUrl, setSelectedImageUrl] = useState<string | null>(null)
  const [cccdFiles, setCccdFiles] = useState<File[]>([])
  const [contractFiles, setContractFiles] = useState<File[]>([])
  const [existingCccdPaths, setExistingCccdPaths] = useState<string[]>([])
  const [existingContractPaths, setExistingContractPaths] = useState<string[]>([])
  const [isSubmitting, setIsSubmitting] = useState(false)
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
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && selectedImageUrl) {
        setSelectedImageUrl(null)
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [selectedImageUrl])

  useEffect(() => {
    return () => {
      if (selectedImageUrl?.startsWith('blob:')) {
        URL.revokeObjectURL(selectedImageUrl)
      }
    }
  }, [selectedImageUrl])

  const handleExistingImagePreview = async (path: string) => {
    setLoadingFilePath(path)
    try {
      const objectUrl = await getProtectedFileObjectUrl(path)
      setSelectedImageUrl((current) => {
        if (current?.startsWith('blob:')) URL.revokeObjectURL(current)
        return objectUrl
      })
    } catch (error) {
      console.error('Không tải được file tenant:', error)
    } finally {
      setLoadingFilePath(null)
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

      cccdFiles.forEach(f => formData.append('cccd_file', f))
      contractFiles.forEach(f => formData.append('contract_file', f))

      formData.append('kept_cccd_paths', existingCccdPaths.join(','))
      formData.append('kept_cccd_paths_empty', existingCccdPaths.length === 0 ? 'true' : 'false')
      formData.append('kept_contract_paths', existingContractPaths.join(','))
      formData.append('kept_contract_paths_empty', existingContractPaths.length === 0 ? 'true' : 'false')

      const res = await updateTenant(tenant.id, formData)
      if (res.success) {
        onSuccess()
      } else {
        alert(res.error || "Lỗi khi sửa người thuê!")
      }
    } catch {
      alert("Lỗi khi sửa người thuê!")
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <>
      <AppModal open onClose={onClose} title="Sửa thông tin người thuê" description={`Phòng ${room.name}`} contentClassName="sm:max-w-2xl">
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2 col-span-2 md:col-span-1">
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
              <Label>Email (Tùy chọn)</Label>
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

            <div className="space-y-4 col-span-2 mt-4 p-4 rounded-xl border border-border/50 bg-card">
              <h3 className="text-sm font-bold text-foreground mb-2">Tài liệu đính kèm</h3>
              <div className="grid grid-cols-2 gap-4 max-h-[350px] overflow-y-auto pr-2">
                {/* CCCD Upload */}
                <div className="space-y-2 col-span-2 md:col-span-1">
                  <Label>Ảnh CCCD</Label>
                  {existingCccdPaths.length === 0 && cccdFiles.length === 0 ? (
                    <label className="flex flex-col gap-2 items-center justify-center h-24 rounded-xl border-2 border-dashed border-border bg-muted/30 cursor-pointer hover:bg-secondary hover:border-primary/50 transition-colors">
                      <Upload className="w-5 h-5 text-muted-foreground" />
                      <span className="text-[10px] text-muted-foreground font-semibold px-4 text-center">Tải lên ảnh CCCD</span>
                      <input type="file" accept="image/*" multiple className="hidden" onChange={e => { if (e.target.files) setCccdFiles(prev => [...prev, ...Array.from(e.target.files!)]) }} />
                    </label>
                  ) : (
                    <div className="flex flex-col gap-2">
                      {existingCccdPaths.map((path, idx) => (
                        <div key={`exist-cccd-${idx}`} className="flex flex-col gap-2">
                          <div className="flex items-center justify-between h-10 px-3 rounded-lg border border-primary/20 bg-primary/10">
                            <div className="flex items-center gap-2 overflow-hidden truncate">
                              <CheckCircle2 className="w-3.5 h-3.5 text-green-500 flex-shrink-0" />
                              <span className="text-[10px] font-semibold text-primary truncate">{getFileName(path)}</span>
                            </div>
                            <div className="flex items-center gap-1">
                              <button type="button" onClick={() => handleExistingImagePreview(path)} disabled={loadingFilePath === path} className="p-1 hover:bg-primary/20 rounded-md text-primary cursor-pointer disabled:opacity-50" title="Xem trước">
                                <Eye className="w-4 h-4" />
                              </button>
                              <button type="button" onClick={() => setExistingCccdPaths(prev => prev.filter((_, i) => i !== idx))} className="p-1 hover:bg-destructive/10 rounded-md text-muted-foreground hover:text-destructive cursor-pointer" title="Xóa">
                                <X className="w-3.5 h-3.5" />
                              </button>
                            </div>
                          </div>
                          {isImagePath(path) ? (
                            <div
                              className="relative rounded-lg overflow-hidden border border-border aspect-video bg-muted cursor-pointer group"
                              onClick={() => handleExistingImagePreview(path)}
                            >
                              <ProtectedFileImage path={path} alt={getFileName(path)} className="w-full h-full object-cover" />
                              <div className="absolute inset-0 bg-black/20 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center text-white">
                                <ZoomIn className="w-5 h-5" />
                              </div>
                            </div>
                          ) : (
                            <div className="rounded-lg border border-border/50 aspect-video bg-muted/30 flex flex-col items-center justify-center text-muted-foreground">
                              <FileIcon className="w-8 h-8" />
                              <span className="text-[10px] font-bold">FILE TÀI LIỆU</span>
                            </div>
                          )}
                        </div>
                      ))}
                      {cccdFiles.map((file, idx) => (
                        <div key={`new-cccd-${idx}`} className="flex flex-col gap-2">
                          <div className="flex items-center justify-between h-10 px-3 rounded-lg border border-primary/20 bg-primary/10">
                            <div className="flex items-center gap-2 overflow-hidden truncate">
                              <CheckCircle2 className="w-3.5 h-3.5 text-green-500 flex-shrink-0" />
                              <span className="text-[10px] font-semibold text-primary truncate">{file.name}</span>
                            </div>
                            <div className="flex items-center gap-1">
                              <button type="button" onClick={() => setSelectedImageUrl(URL.createObjectURL(file))} className="p-1 hover:bg-primary/20 rounded-md text-primary cursor-pointer" title="Xem trước">
                                <Eye className="w-4 h-4" />
                              </button>
                              <button type="button" onClick={() => setCccdFiles(prev => prev.filter((_, i) => i !== idx))} className="p-1 hover:bg-destructive/10 rounded-md text-muted-foreground hover:text-destructive cursor-pointer" title="Xóa">
                                <X className="w-3.5 h-3.5" />
                              </button>
                            </div>
                          </div>
                          <div
                            className="relative rounded-lg overflow-hidden border border-border aspect-video bg-muted cursor-pointer group"
                            onClick={() => setSelectedImageUrl(URL.createObjectURL(file))}
                          >
                            <img src={URL.createObjectURL(file)} className="w-full h-full object-cover" />
                            <div className="absolute inset-0 bg-black/20 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center text-white">
                              <ZoomIn className="w-5 h-5" />
                            </div>
                          </div>
                        </div>
                      ))}
                      <label className="flex items-center justify-center gap-2 h-9 mt-1 rounded-lg border border-dashed border-border bg-muted/30 cursor-pointer hover:bg-secondary transition-colors">
                        <Upload className="w-4 h-4 text-muted-foreground" />
                        <span className="text-xs font-semibold text-foreground">Tải ảnh khác</span>
                        <input type="file" accept="image/*" multiple className="hidden" onChange={e => { if (e.target.files) setCccdFiles(prev => [...prev, ...Array.from(e.target.files!)]) }} />
                      </label>
                    </div>
                  )}
                </div>

                {/* Contract Upload */}
                <div className="space-y-2 col-span-2 md:col-span-1">
                  <Label>Hợp đồng</Label>
                  {existingContractPaths.length === 0 && contractFiles.length === 0 ? (
                    <label className="flex flex-col gap-2 items-center justify-center h-24 rounded-xl border-2 border-dashed border-border bg-muted/30 cursor-pointer hover:bg-secondary hover:border-primary/50 transition-colors">
                      <Upload className="w-5 h-5 text-muted-foreground" />
                      <span className="text-[10px] text-muted-foreground font-semibold px-4 text-center">Tải lên hợp đồng</span>
                      <input type="file" accept=".pdf,.doc,.docx,image/*" multiple className="hidden" onChange={e => { if (e.target.files) setContractFiles(prev => [...prev, ...Array.from(e.target.files!)]) }} />
                    </label>
                  ) : (
                    <div className="flex flex-col gap-2">
                      {existingContractPaths.map((path, idx) => (
                        <div key={`exist-contract-${idx}`} className="flex flex-col gap-2">
                          <div className="flex items-center justify-between h-10 px-3 rounded-lg border border-primary/20 bg-primary/10">
                            <div className="flex items-center gap-2 overflow-hidden truncate">
                              <CheckCircle2 className="w-3.5 h-3.5 text-green-500 flex-shrink-0" />
                              <span className="text-[10px] font-semibold text-primary truncate">{getFileName(path)}</span>
                            </div>
                            <div className="flex items-center gap-1">
                              {isImagePath(path) && (
                                <button type="button" onClick={() => handleExistingImagePreview(path)} disabled={loadingFilePath === path} className="p-1 hover:bg-primary/20 rounded-md text-primary cursor-pointer disabled:opacity-50" title="Xem trước">
                                  <Eye className="w-4 h-4" />
                                </button>
                              )}
                              <button type="button" onClick={() => setExistingContractPaths(prev => prev.filter((_, i) => i !== idx))} className="p-1 hover:bg-destructive/10 rounded-md text-muted-foreground hover:text-destructive cursor-pointer" title="Xóa">
                                <X className="w-3.5 h-3.5" />
                              </button>
                            </div>
                          </div>
                          {isImagePath(path) ? (
                            <div
                              className="relative rounded-lg overflow-hidden border border-border aspect-video bg-muted cursor-pointer group"
                              onClick={() => handleExistingImagePreview(path)}
                            >
                              <ProtectedFileImage path={path} alt={getFileName(path)} className="w-full h-full object-cover" />
                              <div className="absolute inset-0 bg-black/20 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center text-white">
                                <ZoomIn className="w-5 h-5" />
                              </div>
                            </div>
                          ) : (
                            <div className="rounded-lg border border-border/50 aspect-video bg-muted/30 flex flex-col items-center justify-center text-muted-foreground">
                              <FileIcon className="w-8 h-8" />
                              <span className="text-[10px] font-bold">FILE TÀI LIỆU</span>
                            </div>
                          )}
                        </div>
                      ))}
                      {contractFiles.map((file, idx) => (
                        <div key={`new-contract-${idx}`} className="flex flex-col gap-2">
                          <div className="flex items-center justify-between h-10 px-3 rounded-lg border border-primary/20 bg-primary/10">
                            <div className="flex items-center gap-2 overflow-hidden truncate">
                              <CheckCircle2 className="w-3.5 h-3.5 text-green-500 flex-shrink-0" />
                              <span className="text-[10px] font-semibold text-primary truncate">{file.name}</span>
                            </div>
                            <div className="flex items-center gap-1">
                              {file.type.startsWith('image/') && (
                                <button type="button" onClick={() => setSelectedImageUrl(URL.createObjectURL(file))} className="p-1 hover:bg-primary/20 rounded-md text-primary cursor-pointer" title="Xem trước">
                                  <Eye className="w-4 h-4" />
                                </button>
                              )}
                              <button type="button" onClick={() => setContractFiles(prev => prev.filter((_, i) => i !== idx))} className="p-1 hover:bg-destructive/10 rounded-md text-muted-foreground hover:text-destructive cursor-pointer" title="Xóa">
                                <X className="w-3.5 h-3.5" />
                              </button>
                            </div>
                          </div>
                          {file.type.startsWith('image/') ? (
                            <div
                              className="relative rounded-lg overflow-hidden border border-border aspect-video bg-muted cursor-pointer group"
                              onClick={() => setSelectedImageUrl(URL.createObjectURL(file))}
                            >
                              <img src={URL.createObjectURL(file)} className="w-full h-full object-cover" />
                              <div className="absolute inset-0 bg-black/20 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center text-white">
                                <ZoomIn className="w-5 h-5" />
                              </div>
                            </div>
                          ) : (
                            <div className="rounded-lg border border-border/50 aspect-video bg-muted/30 flex flex-col items-center justify-center text-muted-foreground">
                              <FileIcon className="w-8 h-8" />
                              <span className="text-[10px] font-bold">FILE TÀI LIỆU</span>
                            </div>
                          )}
                        </div>
                      ))}
                      <label className="flex items-center justify-center gap-2 h-9 mt-1 rounded-lg border border-dashed border-border bg-muted/30 cursor-pointer hover:bg-secondary transition-colors">
                        <Upload className="w-4 h-4 text-muted-foreground" />
                        <span className="text-xs font-semibold text-foreground">Tải file khác</span>
                        <input type="file" accept=".pdf,.doc,.docx,image/*" multiple className="hidden" onChange={e => { if (e.target.files) setContractFiles(prev => [...prev, ...Array.from(e.target.files!)]) }} />
                      </label>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>

          <div className="flex justify-end gap-3 pt-6 border-t border-border/40">
            <Button type="button" variant="outline" onClick={onClose} className="font-bold cursor-pointer">Hủy</Button>
            <Button type="submit" disabled={isSubmitting} className="shadow-sm font-bold transition-all cursor-pointer">
              {isSubmitting ? 'Đang lưu...' : 'Lưu thay đổi'}
            </Button>
          </div>
        </form>
      </AppModal>

      {/* Lightbox / Gallery Modal */}
      {selectedImageUrl && (
        <div
          className="fixed inset-0 z-[100] flex items-center justify-center bg-black/90 backdrop-blur-md safe-fade-in p-4"
          onClick={() => setSelectedImageUrl(null)}
        >
          <button className="absolute top-6 right-6 p-2 rounded-full bg-white/10 text-white hover:bg-white/20 transition-colors z-[101] cursor-pointer">
            <X className="w-6 h-6" />
          </button>
          <div className="max-w-5xl max-h-[90vh] w-full flex items-center justify-center relative">
            {selectedImageUrl.startsWith('blob:') || selectedImageUrl.match(/\.(jpg|jpeg|png|gif|webp)$/i) || selectedImageUrl.includes('/files/') ? (
              <img
                src={selectedImageUrl}
                alt="Preview"
                className="max-w-full max-h-[90vh] object-contain shadow-2xl rounded-sm safe-fade-in"
                onClick={(e) => e.stopPropagation()}
              />
            ) : (
              <div className="bg-card text-card-foreground p-8 rounded-2xl flex flex-col items-center gap-4 text-center" onClick={(e) => e.stopPropagation()}>
                <FileIcon className="w-16 h-16 text-primary" />
                <div>
                  <h3 className="font-bold text-foreground text-xl">Định dạng file đặc biệt</h3>
                  <p className="text-muted-foreground">File này không thể xem trước trực tiếp.</p>
                </div>
                <Button onClick={() => window.open(selectedImageUrl, '_blank')} className="shadow-sm">Tải về hoặc Mở tab mới</Button>
              </div>
            )}
          </div>
        </div>
      )}
    </>
  )
}
