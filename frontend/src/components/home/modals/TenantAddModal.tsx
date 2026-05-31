import { useState, useEffect } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { UserPlus, Upload, CheckCircle2, X, FileIcon, ZoomIn, Eye } from 'lucide-react'
import type { Room } from '@/api/room'
import { useTenantStore } from '@/data/tenantData'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'

const tenantSchema = z.object({
  fullName: z.string().min(1, 'Bắt buộc'),
  phone: z.string().min(1, 'Bắt buộc'),
  email: z.string().email('Email không hợp lệ').optional().or(z.literal('')),
  identityCard: z.string().min(1, 'Bắt buộc'),
  startDate: z.string(),
})

type TenantFormValues = z.infer<typeof tenantSchema>

interface TenantAddModalProps {
  room: Room
  onClose: () => void
  onSuccess: () => void
}

export function TenantAddModal({ room, onClose, onSuccess }: TenantAddModalProps) {
  const createTenant = useTenantStore(state => state.createTenant)
  
  const [selectedImageUrl, setSelectedImageUrl] = useState<string | null>(null)
  const [cccdFiles, setCccdFiles] = useState<File[]>([])
  const [contractFiles, setContractFiles] = useState<File[]>([])
  const [isSubmitting, setIsSubmitting] = useState(false)

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
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (selectedImageUrl) {
          setSelectedImageUrl(null)
        } else {
          onClose()
        }
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [onClose, selectedImageUrl])

  const onSubmit = async (values: TenantFormValues) => {
    setIsSubmitting(true)
    try {
      const formData = new FormData()
      formData.append('room_id', room.id)
      formData.append('full_name', values.fullName)
      formData.append('phone', values.phone)
      if (values.email) formData.append('email', values.email)
      formData.append('identity_card', values.identityCard)
      formData.append('start_date', values.startDate)
      
      cccdFiles.forEach(f => formData.append('cccd_file', f))
      contractFiles.forEach(f => formData.append('contract_file', f))
      
      await createTenant(formData)
      onSuccess()
    } catch {
      alert("Lỗi khi thêm người thuê!")
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <>
      <div 
        className="fixed inset-0 z-[60] flex items-center justify-center bg-black/40 backdrop-blur-sm px-4 overflow-y-auto py-10"
        onClick={onClose}
      >
        <Card 
          className="w-full max-w-2xl bg-white shadow-xl border-0 animate-in zoom-in-95 duration-200"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="p-6 border-b border-slate-100 flex items-center gap-3 bg-slate-50/50 rounded-t-xl">
            <div className="w-10 h-10 rounded-full flex items-center justify-center bg-blue-100 text-blue-600">
              <UserPlus className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-xl font-bold text-slate-800">Thêm người thuê mới</h2>
              <p className="text-sm text-slate-500">Phòng {room.name}</p>
            </div>
            <button onClick={onClose} className="ml-auto p-2 text-slate-400 hover:text-slate-600 rounded-full hover:bg-slate-100 transition-colors cursor-pointer">
              <X className="w-5 h-5" />
            </button>
          </div>

          <div className="p-6">
            <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2 col-span-2 md:col-span-1">
                  <Label>Họ và tên <span className="text-red-500">*</span></Label>
                  <Input {...register('fullName')} placeholder="Nguyễn Văn A" className="border-slate-200" />
                  {errors.fullName && <span className="text-red-500 text-xs">{errors.fullName.message}</span>}
                </div>
                <div className="space-y-2 col-span-2 md:col-span-1">
                  <Label>Số điện thoại <span className="text-red-500">*</span></Label>
                  <Input {...register('phone')} placeholder="09..." className="border-slate-200" />
                  {errors.phone && <span className="text-red-500 text-xs">{errors.phone.message}</span>}
                </div>
                <div className="space-y-2 col-span-2 md:col-span-1">
                  <Label>Email (Tùy chọn)</Label>
                  <Input type="email" {...register('email')} placeholder="abc@gmail.com" className="border-slate-200" />
                  {errors.email && <span className="text-red-500 text-xs">{errors.email.message}</span>}
                </div>
                <div className="space-y-2 col-span-2 md:col-span-1">
                  <Label>Căn cước công dân <span className="text-red-500">*</span></Label>
                  <Input {...register('identityCard')} placeholder="12 số CCCD" className="border-slate-200" />
                  {errors.identityCard && <span className="text-red-500 text-xs">{errors.identityCard.message}</span>}
                </div>
                <div className="space-y-2 col-span-2 md:col-span-1">
                  <Label>Ngày bắt đầu thuê</Label>
                  <Input type="date" {...register('startDate')} className="border-slate-200" />
                  {errors.startDate && <span className="text-red-500 text-xs">{errors.startDate.message}</span>}
                </div>

                <div className="space-y-4 col-span-2 mt-4 p-4 rounded-xl border border-slate-200 bg-white">
                  <h3 className="text-sm font-bold text-slate-800 mb-2">Tài liệu đính kèm</h3>
                  <div className="grid grid-cols-2 gap-4 max-h-[350px] overflow-y-auto pr-2">
                    {/* CCCD Upload */}
                    <div className="space-y-2 col-span-2 md:col-span-1">
                      <Label>Ảnh CCCD</Label>
                      {cccdFiles.length === 0 ? (
                        <label className="flex flex-col gap-2 items-center justify-center h-24 rounded-xl border-2 border-dashed border-slate-300 bg-slate-50 cursor-pointer hover:bg-slate-100 hover:border-blue-300 transition-colors">
                          <Upload className="w-5 h-5 text-slate-400" />
                          <span className="text-[10px] text-slate-500 font-semibold px-4 text-center">Tải lên ảnh CCCD</span>
                          <input type="file" accept="image/*" multiple className="hidden" onChange={e => { if (e.target.files) setCccdFiles(prev => [...prev, ...Array.from(e.target.files!)]) }} />
                        </label>
                      ) : (
                        <div className="flex flex-col gap-2">
                          {cccdFiles.map((file, idx) => (
                            <div key={`new-cccd-${idx}`} className="flex flex-col gap-2">
                              <div className="flex items-center justify-between h-10 px-3 rounded-lg border border-blue-200 bg-blue-50/50">
                                <div className="flex items-center gap-2 overflow-hidden truncate">
                                  <CheckCircle2 className="w-3.5 h-3.5 text-green-500 flex-shrink-0" />
                                  <span className="text-[10px] font-semibold text-blue-700 truncate">{file.name}</span>
                                </div>
                                <div className="flex items-center gap-1">
                                  <button type="button" onClick={() => setSelectedImageUrl(URL.createObjectURL(file))} className="p-1 hover:bg-blue-100 rounded-md text-blue-600 cursor-pointer" title="Xem trước">
                                    <Eye className="w-4 h-4" />
                                  </button>
                                  <button type="button" onClick={() => setCccdFiles(prev => prev.filter((_, i) => i !== idx))} className="p-1 hover:bg-blue-100 rounded-md text-slate-500 cursor-pointer" title="Xóa">
                                    <X className="w-3.5 h-3.5" />
                                  </button>
                                </div>
                              </div>
                              <div
                                className="relative rounded-lg overflow-hidden border border-slate-200 aspect-video bg-slate-100 cursor-pointer group"
                                onClick={() => setSelectedImageUrl(URL.createObjectURL(file))}
                              >
                                <img src={URL.createObjectURL(file)} className="w-full h-full object-cover" />
                                <div className="absolute inset-0 bg-black/20 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center text-white">
                                  <ZoomIn className="w-5 h-5" />
                                </div>
                              </div>
                            </div>
                          ))}
                          <label className="flex items-center justify-center gap-2 h-9 mt-1 rounded-lg border border-dashed border-slate-300 bg-slate-50 cursor-pointer hover:bg-slate-100 transition-colors">
                            <Upload className="w-4 h-4 text-slate-500" />
                            <span className="text-xs font-semibold text-slate-600">Tải ảnh khác</span>
                            <input type="file" accept="image/*" multiple className="hidden" onChange={e => { if (e.target.files) setCccdFiles(prev => [...prev, ...Array.from(e.target.files!)]) }} />
                          </label>
                        </div>
                      )}
                    </div>

                    {/* Contract Upload */}
                    <div className="space-y-2 col-span-2 md:col-span-1">
                      <Label>Hợp đồng</Label>
                      {contractFiles.length === 0 ? (
                        <label className="flex flex-col gap-2 items-center justify-center h-24 rounded-xl border-2 border-dashed border-slate-300 bg-slate-50 cursor-pointer hover:bg-slate-100 hover:border-blue-300 transition-colors">
                          <Upload className="w-5 h-5 text-slate-400" />
                          <span className="text-[10px] text-slate-500 font-semibold px-4 text-center">Tải lên hợp đồng</span>
                          <input type="file" accept=".pdf,.doc,.docx,image/*" multiple className="hidden" onChange={e => { if (e.target.files) setContractFiles(prev => [...prev, ...Array.from(e.target.files!)]) }} />
                        </label>
                      ) : (
                        <div className="flex flex-col gap-2">
                          {contractFiles.map((file, idx) => (
                            <div key={`new-contract-${idx}`} className="flex flex-col gap-2">
                              <div className="flex items-center justify-between h-10 px-3 rounded-lg border border-blue-200 bg-blue-50/50">
                                <div className="flex items-center gap-2 overflow-hidden truncate">
                                  <CheckCircle2 className="w-3.5 h-3.5 text-green-500 flex-shrink-0" />
                                  <span className="text-[10px] font-semibold text-blue-700 truncate">{file.name}</span>
                                </div>
                                <div className="flex items-center gap-1">
                                  {file.type.startsWith('image/') && (
                                    <button type="button" onClick={() => setSelectedImageUrl(URL.createObjectURL(file))} className="p-1 hover:bg-blue-100 rounded-md text-blue-600 cursor-pointer" title="Xem trước">
                                      <Eye className="w-4 h-4" />
                                    </button>
                                  )}
                                  <button type="button" onClick={() => setContractFiles(prev => prev.filter((_, i) => i !== idx))} className="p-1 hover:bg-blue-100 rounded-md text-slate-500 cursor-pointer" title="Xóa">
                                    <X className="w-3.5 h-3.5" />
                                  </button>
                                </div>
                              </div>
                              {file.type.startsWith('image/') ? (
                                <div
                                  className="relative rounded-lg overflow-hidden border border-slate-200 aspect-video bg-slate-100 cursor-pointer group"
                                  onClick={() => setSelectedImageUrl(URL.createObjectURL(file))}
                                >
                                  <img src={URL.createObjectURL(file)} className="w-full h-full object-cover" />
                                  <div className="absolute inset-0 bg-black/20 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center text-white">
                                    <ZoomIn className="w-5 h-5" />
                                  </div>
                                </div>
                              ) : (
                                <div className="rounded-lg border border-slate-200 aspect-video bg-slate-50 flex flex-col items-center justify-center text-slate-400">
                                  <FileIcon className="w-8 h-8" />
                                  <span className="text-[10px] font-bold">FILE TÀI LIỆU</span>
                                </div>
                              )}
                            </div>
                          ))}
                          <label className="flex items-center justify-center gap-2 h-9 mt-1 rounded-lg border border-dashed border-slate-300 bg-slate-50 cursor-pointer hover:bg-slate-100 transition-colors">
                            <Upload className="w-4 h-4 text-slate-500" />
                            <span className="text-xs font-semibold text-slate-600">Tải file khác</span>
                            <input type="file" accept=".pdf,.doc,.docx,image/*" multiple className="hidden" onChange={e => { if (e.target.files) setContractFiles(prev => [...prev, ...Array.from(e.target.files!)]) }} />
                          </label>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              </div>

              <div className="flex justify-end gap-3 pt-6 border-t border-slate-100">
                <Button type="button" variant="outline" onClick={onClose} className="border-slate-200 text-slate-600 font-bold cursor-pointer">Hủy</Button>
                <Button type="submit" disabled={isSubmitting} className="bg-blue-600 hover:bg-blue-700 text-white font-bold shadow-md shadow-blue-200 transition-all cursor-pointer">
                  {isSubmitting ? 'Đang thêm...' : 'Xác nhận'}
                </Button>
              </div>
            </form>
          </div>
        </Card>
      </div>

      {/* Lightbox / Gallery Modal */}
      {selectedImageUrl && (
        <div
          className="fixed inset-0 z-[100] flex items-center justify-center bg-black/90 backdrop-blur-md animate-in fade-in duration-200 p-4"
          onClick={() => setSelectedImageUrl(null)}
        >
          <button className="absolute top-6 right-6 p-2 rounded-full bg-white/10 text-white hover:bg-white/20 transition-colors z-[101] cursor-pointer">
            <X className="w-6 h-6" />
          </button>
          <div className="max-w-5xl max-h-[90vh] w-full flex items-center justify-center relative">
            <img
              src={selectedImageUrl}
              alt="Preview"
              className="max-w-full max-h-[90vh] object-contain shadow-2xl rounded-sm animate-in zoom-in-95 duration-300"
              onClick={(e) => e.stopPropagation()}
            />
          </div>
        </div>
      )}
    </>
  )
}
