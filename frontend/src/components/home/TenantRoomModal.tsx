import { useState, useEffect } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { UserPlus, User, Upload, CheckCircle2, Plus, Trash2, Pencil, X, ExternalLink, FileIcon, ZoomIn, Eye, Download } from 'lucide-react'
import { createTenant, updateTenant, type Tenant } from '@/api/tenant'
import type { Room } from '@/api/room'
import { useRoomStore } from '@/data/roomData'
import { useTenantList } from '@/hooks/useTenantList'
import { getFileName, isImagePath } from '@/utils/file'
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

export function TenantRoomModal({ room, onClose }: { room: Room, onClose: () => void }) {
  const isOccupied = room.status === 'OCCUPIED'
  const refreshCurrentRooms = useRoomStore(state => state.refreshCurrentRooms)

  const {
    tenants,
    isLoading: isDeleting,
    isFetching,
    showAddForm,
    editingTenantId,
    fetchTenants,
    handleDeleteTenant,
    setShowAddForm,
    setEditingTenantId,
    setIsFetching
  } = useTenantList(room.id, !isOccupied)

  useEffect(() => {
    if (isOccupied) {
      fetchTenants(room.id)
    } else {
      setIsFetching(false)
    }
  }, [isOccupied, room.id, fetchTenants, setIsFetching])

  const [selectedImageUrl, setSelectedImageUrl] = useState<string | null>(null)

  // File states
  const [cccdFiles, setCccdFiles] = useState<File[]>([])
  const [contractFiles, setContractFiles] = useState<File[]>([])
  const [existingCccdPaths, setExistingCccdPaths] = useState<string[]>([])
  const [existingContractPaths, setExistingContractPaths] = useState<string[]>([])

  const [isSubmitting, setIsSubmitting] = useState(false)

  const { register, handleSubmit, reset, formState: { errors } } = useForm<TenantFormValues>({
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
      
      if (editingTenantId) {
        formData.append('kept_cccd_paths', existingCccdPaths.join(','))
        formData.append('kept_cccd_paths_empty', existingCccdPaths.length === 0 ? 'true' : 'false')
        formData.append('kept_contract_paths', existingContractPaths.join(','))
        formData.append('kept_contract_paths_empty', existingContractPaths.length === 0 ? 'true' : 'false')
      }

      if (editingTenantId) {
        await updateTenant(editingTenantId, formData)
      } else {
        await createTenant(formData)
      }

      setEditingTenantId(null)
      setShowAddForm(false)
      await fetchTenants(room.id)
      await refreshCurrentRooms()
      resetForm()
      
      // Auto close if added successfully and no more actions needed, or keep open if they want to see list. Let's keep open on list.
    } catch {
      alert("Lỗi khi xử lý thao tác người thuê!")
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleDelete = async (tenantId: string) => {
    if (!confirm("Bạn có chắc chắn muốn xóa người thuê này?")) return
    try {
      await handleDeleteTenant(tenantId, room.id)
      await refreshCurrentRooms()
    } catch {
      // Error handled inside hook
    }
  }

  const handleEditClick = (tenant: Tenant) => {
    reset({
      fullName: tenant.full_name,
      phone: tenant.phone,
      email: tenant.email || '',
      identityCard: tenant.identity_card,
      startDate: tenant.start_date.split('T')[0]
    })
    setCccdFiles([])
    setContractFiles([])
    setExistingCccdPaths(tenant.cccd_path ? tenant.cccd_path.split(',').filter(Boolean) : [])
    setExistingContractPaths(tenant.contract_path ? tenant.contract_path.split(',').filter(Boolean) : [])
    setEditingTenantId(tenant.id)
    setShowAddForm(true)
  }

  const handleFileClick = (path: string) => {
    const fullUrl = `${import.meta.env.VITE_API_BASE_URL || ''}${path}`
    if (isImagePath(path)) {
      setSelectedImageUrl(fullUrl)
    } else {
      window.open(fullUrl, '_blank')
    }
  }

  const resetForm = () => {
    reset({
      fullName: '',
      phone: '',
      email: '',
      identityCard: '',
      startDate: new Date().toISOString().split('T')[0]
    })
    setCccdFiles([])
    setContractFiles([])
    setExistingCccdPaths([])
    setExistingContractPaths([])
    setEditingTenantId(null)
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
              {isOccupied ? <User className="w-5 h-5" /> : <UserPlus className="w-5 h-5" />}
            </div>
            <div>
              <h2 className="text-xl font-bold text-slate-800">
                {showAddForm ? (editingTenantId ? 'Sửa người thuê' : 'Thêm người thuê mới') : 'Danh sách người thuê'}
              </h2>
              <p className="text-sm text-slate-500">Phòng {room.name} {(!showAddForm) && `(${tenants.length}/${room.max_tenants})`}</p>
            </div>
            <button onClick={onClose} className="ml-auto p-2 text-slate-400 hover:text-slate-600 rounded-full hover:bg-slate-100 transition-colors">
              <X className="w-5 h-5" />
            </button>
          </div>

          {isFetching ? (
            <div className="p-10 text-center text-slate-500">Đang tải thông tin...</div>
          ) : (
            <div className="p-6">
              {(!showAddForm) ? (
                <div className="space-y-6">
                  <div className="space-y-4">
                    {tenants.map((t, idx) => (
                      <div key={t.id} className="p-4 rounded-xl border border-slate-100 bg-slate-50/50 group">
                        <div className="flex justify-between items-start mb-3">
                          <div className="flex items-center gap-2">
                            <div className="w-8 h-8 rounded-full bg-blue-100 text-blue-600 flex items-center justify-center text-xs font-bold">
                              {idx + 1}
                            </div>
                            <h4 className="font-bold text-slate-800">{t.full_name}</h4>
                          </div>
                          <div className="flex items-center gap-2">
                            <button onClick={() => handleEditClick(t)} className="p-1.5 text-slate-400 hover:text-blue-600 bg-white shadow-sm border border-slate-200 rounded-md transition-colors cursor-pointer" title="Sửa">
                              <Pencil className="w-3.5 h-3.5" />
                            </button>
                            <button disabled={isDeleting} onClick={() => handleDelete(t.id)} className="p-1.5 text-slate-400 hover:text-red-600 bg-white shadow-sm border border-slate-200 rounded-md transition-colors cursor-pointer disabled:opacity-50" title="Xóa">
                              <Trash2 className="w-3.5 h-3.5" />
                            </button>
                            <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-[10px] font-bold bg-green-100 text-green-700 uppercase tracking-wider ml-1">
                              <CheckCircle2 className="w-3 h-3" />
                              Đang ở
                            </span>
                          </div>
                        </div>
                        <div className="grid grid-cols-2 gap-y-3 gap-x-4">
                          <div className="flex flex-col">
                            <span className="text-[10px] uppercase font-bold text-slate-400 tracking-wider">Số điện thoại</span>
                            <span className="text-sm font-semibold text-slate-700">{t.phone}</span>
                          </div>
                          <div className="flex flex-col">
                            <span className="text-[10px] uppercase font-bold text-slate-400 tracking-wider">Email</span>
                            <span className="text-sm font-semibold text-slate-700">{t.email || 'N/A'}</span>
                          </div>
                          <div className="flex flex-col">
                            <span className="text-[10px] uppercase font-bold text-slate-400 tracking-wider">CCCD</span>
                            <span className="text-sm font-semibold text-slate-700">{t.identity_card}</span>
                          </div>
                          <div className="flex flex-col">
                            <span className="text-[10px] uppercase font-bold text-slate-400 tracking-wider">Ngày bắt đầu</span>
                            <span className="text-sm font-semibold text-slate-700">{new Date(t.start_date).toLocaleDateString('vi-VN')}</span>
                          </div>

                          {(t.cccd_path || t.contract_path) && (
                            <div className="col-span-2 grid grid-cols-2 gap-2 mt-2 max-h-80 overflow-y-auto pr-1">
                              {t.cccd_path && t.cccd_path.split(',').filter(Boolean).map((path, fileIdx) => (
                                <div key={`cccd-${fileIdx}`} className="flex items-center justify-between p-2 rounded-lg bg-white border border-slate-200 group/file hover:border-blue-300 transition-colors">
                                  <div className="flex items-center gap-2 overflow-hidden">
                                    <div className="w-6 h-6 rounded bg-blue-100 text-blue-600 flex items-center justify-center flex-shrink-0">
                                      <FileIcon className="w-3 h-3" />
                                    </div>
                                    <div className="flex flex-col overflow-hidden">
                                      <span className="text-[9px] font-bold text-slate-400 uppercase">Ảnh CCCD {fileIdx > 0 ? fileIdx + 1 : ''}</span>
                                      <span className="text-[10px] font-semibold text-slate-600 truncate">{getFileName(path)}</span>
                                    </div>
                                  </div>
                                  <div className="flex items-center gap-1">
                                    <button
                                      onClick={() => handleFileClick(path)}
                                      className="p-1.5 text-slate-400 hover:text-blue-600 hover:bg-blue-50 rounded-md transition-colors cursor-pointer"
                                      title={isImagePath(path) ? "Xem ảnh" : "Tải về"}
                                    >
                                      {isImagePath(path) ? <ZoomIn className="w-3.5 h-3.5" /> : <Download className="w-3.5 h-3.5 cursor-pointer" />}
                                    </button>
                                    <a
                                      href={`${import.meta.env.VITE_API_BASE_URL || ''}${path}`}
                                      target="_blank"
                                      rel="noopener noreferrer"
                                      download
                                      className="p-1.5 text-slate-400 hover:text-blue-600 hover:bg-blue-50 rounded-md transition-colors"
                                    >
                                      <ExternalLink className="w-3.5 h-3.5" />
                                    </a>
                                  </div>
                                </div>
                              ))}
                              {t.contract_path && t.contract_path.split(',').filter(Boolean).map((path, fileIdx) => (
                                <div key={`contract-${fileIdx}`} className="flex items-center justify-between p-2 rounded-lg bg-white border border-slate-200 group/file hover:border-blue-300 transition-colors">
                                  <div className="flex items-center gap-2 overflow-hidden">
                                    <div className="w-6 h-6 rounded bg-blue-100 text-blue-600 flex items-center justify-center flex-shrink-0">
                                      <FileIcon className="w-3 h-3" />
                                    </div>
                                    <div className="flex flex-col overflow-hidden">
                                      <span className="text-[9px] font-bold text-slate-400 uppercase">Hợp đồng {fileIdx > 0 ? fileIdx + 1 : ''}</span>
                                      <span className="text-[10px] font-semibold text-slate-600 truncate">{getFileName(path)}</span>
                                    </div>
                                  </div>
                                  <div className="flex items-center gap-1">
                                    <button
                                      onClick={() => handleFileClick(path)}
                                      className="p-1.5 text-slate-400 hover:text-blue-600 hover:bg-blue-50 rounded-md transition-colors"
                                      title={isImagePath(path) ? "Xem ảnh" : "Tải về"}
                                    >
                                      {isImagePath(path) ? <ZoomIn className="w-3.5 h-3.5" /> : <Download className="w-3.5 h-3.5" />}
                                    </button>
                                    <a
                                      href={`${import.meta.env.VITE_API_BASE_URL || ''}${path}`}
                                      target="_blank"
                                      rel="noopener noreferrer"
                                      download
                                      className="p-1.5 text-slate-400 hover:text-blue-600 hover:bg-blue-50 rounded-md transition-colors"
                                    >
                                      <ExternalLink className="w-3.5 h-3.5" />
                                    </a>
                                  </div>
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>

                  <div className="flex justify-between items-center pt-4 border-t border-slate-100">
                    <div className="flex gap-2">
                      {tenants.length < room.max_tenants && (
                        <Button onClick={() => { resetForm(); setShowAddForm(true) }} variant="default" className="bg-blue-600 hover:bg-blue-700 font-bold gap-2 cursor-pointer">
                          <Plus className="w-4 h-4" /> Thêm người ở
                        </Button>
                      )}
                    </div>
                    <Button onClick={onClose} variant="outline" className="font-bold border-slate-200 cursor-pointer">Đóng</Button>
                  </div>
                </div>
              ) : (
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
                          {existingCccdPaths.length === 0 && cccdFiles.length === 0 ? (
                            <label className="flex flex-col gap-2 items-center justify-center h-24 rounded-xl border-2 border-dashed border-slate-300 bg-slate-50 cursor-pointer hover:bg-slate-100 hover:border-blue-300 transition-colors">
                              <Upload className="w-5 h-5 text-slate-400" />
                              <span className="text-[10px] text-slate-500 font-semibold px-4 text-center">Tải lên ảnh CCCD</span>
                              <input type="file" accept="image/*" multiple className="hidden" onChange={e => { if (e.target.files) setCccdFiles(prev => [...prev, ...Array.from(e.target.files!)]) }} />
                            </label>
                          ) : (
                            <div className="flex flex-col gap-2">
                              {existingCccdPaths.map((path, idx) => (
                                <div key={`exist-cccd-${idx}`} className="flex flex-col gap-2">
                                  <div className="flex items-center justify-between h-10 px-3 rounded-lg border border-blue-200 bg-blue-50/50">
                                    <div className="flex items-center gap-2 overflow-hidden truncate">
                                      <CheckCircle2 className="w-3.5 h-3.5 text-green-500 flex-shrink-0" />
                                      <span className="text-[10px] font-semibold text-blue-700 truncate">{getFileName(path)}</span>
                                    </div>
                                    <div className="flex items-center gap-1">
                                      <button type="button" onClick={() => setSelectedImageUrl(`${import.meta.env.VITE_API_BASE_URL || ''}${path}`)} className="p-1 hover:bg-blue-100 rounded-md text-blue-600" title="Xem trước">
                                        <Eye className="w-4 h-4" />
                                      </button>
                                      <button type="button" onClick={() => setExistingCccdPaths(prev => prev.filter((_, i) => i !== idx))} className="p-1 hover:bg-blue-100 rounded-md text-slate-500" title="Xóa">
                                        <X className="w-3.5 h-3.5" />
                                      </button>
                                    </div>
                                  </div>
                                  {isImagePath(path) ? (
                                    <div
                                      className="relative rounded-lg overflow-hidden border border-slate-200 aspect-video bg-slate-100 cursor-pointer group"
                                      onClick={() => setSelectedImageUrl(`${import.meta.env.VITE_API_BASE_URL || ''}${path}`)}
                                    >
                                      <img src={`${import.meta.env.VITE_API_BASE_URL || ''}${path}`} className="w-full h-full object-cover" />
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
                              {cccdFiles.map((file, idx) => (
                                <div key={`new-cccd-${idx}`} className="flex flex-col gap-2">
                                  <div className="flex items-center justify-between h-10 px-3 rounded-lg border border-blue-200 bg-blue-50/50">
                                    <div className="flex items-center gap-2 overflow-hidden truncate">
                                      <CheckCircle2 className="w-3.5 h-3.5 text-green-500 flex-shrink-0" />
                                      <span className="text-[10px] font-semibold text-blue-700 truncate">{file.name}</span>
                                    </div>
                                    <div className="flex items-center gap-1">
                                      <button type="button" onClick={() => setSelectedImageUrl(URL.createObjectURL(file))} className="p-1 hover:bg-blue-100 rounded-md text-blue-600" title="Xem trước">
                                        <Eye className="w-4 h-4" />
                                      </button>
                                      <button type="button" onClick={() => setCccdFiles(prev => prev.filter((_, i) => i !== idx))} className="p-1 hover:bg-blue-100 rounded-md text-slate-500" title="Xóa">
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
                          {existingContractPaths.length === 0 && contractFiles.length === 0 ? (
                            <label className="flex flex-col gap-2 items-center justify-center h-24 rounded-xl border-2 border-dashed border-slate-300 bg-slate-50 cursor-pointer hover:bg-slate-100 hover:border-blue-300 transition-colors">
                              <Upload className="w-5 h-5 text-slate-400" />
                              <span className="text-[10px] text-slate-500 font-semibold px-4 text-center">Tải lên hợp đồng</span>
                              <input type="file" accept=".pdf,.doc,.docx,image/*" multiple className="hidden" onChange={e => { if (e.target.files) setContractFiles(prev => [...prev, ...Array.from(e.target.files!)]) }} />
                            </label>
                          ) : (
                            <div className="flex flex-col gap-2">
                              {existingContractPaths.map((path, idx) => (
                                <div key={`exist-contract-${idx}`} className="flex flex-col gap-2">
                                  <div className="flex items-center justify-between h-10 px-3 rounded-lg border border-blue-200 bg-blue-50/50">
                                    <div className="flex items-center gap-2 overflow-hidden truncate">
                                      <CheckCircle2 className="w-3.5 h-3.5 text-green-500 flex-shrink-0" />
                                      <span className="text-[10px] font-semibold text-blue-700 truncate">{getFileName(path)}</span>
                                    </div>
                                    <div className="flex items-center gap-1">
                                      {isImagePath(path) && (
                                        <button type="button" onClick={() => setSelectedImageUrl(`${import.meta.env.VITE_API_BASE_URL || ''}${path}`)} className="p-1 hover:bg-blue-100 rounded-md text-blue-600" title="Xem trước">
                                          <Eye className="w-4 h-4" />
                                        </button>
                                      )}
                                      <button type="button" onClick={() => setExistingContractPaths(prev => prev.filter((_, i) => i !== idx))} className="p-1 hover:bg-blue-100 rounded-md text-slate-500" title="Xóa">
                                        <X className="w-3.5 h-3.5" />
                                      </button>
                                    </div>
                                  </div>
                                  {isImagePath(path) ? (
                                    <div
                                      className="relative rounded-lg overflow-hidden border border-slate-200 aspect-video bg-slate-100 cursor-pointer group"
                                      onClick={() => setSelectedImageUrl(`${import.meta.env.VITE_API_BASE_URL || ''}${path}`)}
                                    >
                                      <img src={`${import.meta.env.VITE_API_BASE_URL || ''}${path}`} className="w-full h-full object-cover" />
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
                              {contractFiles.map((file, idx) => (
                                <div key={`new-contract-${idx}`} className="flex flex-col gap-2">
                                  <div className="flex items-center justify-between h-10 px-3 rounded-lg border border-blue-200 bg-blue-50/50">
                                    <div className="flex items-center gap-2 overflow-hidden truncate">
                                      <CheckCircle2 className="w-3.5 h-3.5 text-green-500 flex-shrink-0" />
                                      <span className="text-[10px] font-semibold text-blue-700 truncate">{file.name}</span>
                                    </div>
                                    <div className="flex items-center gap-1">
                                      {file.type.startsWith('image/') && (
                                        <button type="button" onClick={() => setSelectedImageUrl(URL.createObjectURL(file))} className="p-1 hover:bg-blue-100 rounded-md text-blue-600" title="Xem trước">
                                          <Eye className="w-4 h-4" />
                                        </button>
                                      )}
                                      <button type="button" onClick={() => setContractFiles(prev => prev.filter((_, i) => i !== idx))} className="p-1 hover:bg-blue-100 rounded-md text-slate-500" title="Xóa">
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
                    <Button type="button" variant="outline" onClick={() => {
                      if (tenants.length > 0) {
                        resetForm()
                        setShowAddForm(false)
                        setEditingTenantId(null)
                      } else {
                        onClose()
                      }
                    }} className="border-slate-200 text-slate-600 font-bold">Hủy</Button>
                    <Button type="submit" disabled={isSubmitting} className="bg-blue-600 hover:bg-blue-700 text-white font-bold shadow-md shadow-blue-200 transition-all">
                      {isSubmitting ? 'Đang lưu...' : (editingTenantId ? 'Lưu thay đổi' : 'Xác nhận')}
                    </Button>
                  </div>
                </form>
              )}
            </div>
          )}
        </Card>
      </div>

      {/* Lightbox / Gallery Modal */}
      {selectedImageUrl && (
        <div
          className="fixed inset-0 z-[100] flex items-center justify-center bg-black/90 backdrop-blur-md animate-in fade-in duration-200 p-4"
          onClick={() => setSelectedImageUrl(null)}
        >
          <button className="absolute top-6 right-6 p-2 rounded-full bg-white/10 text-white hover:bg-white/20 transition-colors z-[101]">
            <X className="w-6 h-6" />
          </button>
          <div className="max-w-5xl max-h-[90vh] w-full flex items-center justify-center relative">
            {selectedImageUrl.startsWith('blob:') || selectedImageUrl.match(/\.(jpg|jpeg|png|gif|webp)$/i) || selectedImageUrl.includes('/files/') ? (
              <img
                src={selectedImageUrl}
                alt="Preview"
                className="max-w-full max-h-[90vh] object-contain shadow-2xl rounded-sm animate-in zoom-in-95 duration-300"
                onClick={(e) => e.stopPropagation()}
              />
            ) : (
              <div className="bg-white p-8 rounded-2xl flex flex-col items-center gap-4 text-center" onClick={(e) => e.stopPropagation()}>
                <FileIcon className="w-16 h-16 text-blue-500" />
                <div>
                  <h3 className="font-bold text-slate-800 text-xl">Định dạng file đặc biệt</h3>
                  <p className="text-slate-500">File này không thể xem trước trực tiếp.</p>
                </div>
                <Button onClick={() => window.open(selectedImageUrl, '_blank')} className="bg-blue-600">Tải về hoặc Mở tab mới</Button>
              </div>
            )}
          </div>
        </div>
      )}
    </>
  )
}
