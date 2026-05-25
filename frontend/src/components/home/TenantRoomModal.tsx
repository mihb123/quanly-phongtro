import React, { useState, useEffect } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { UserPlus, User, Upload, CheckCircle2, Plus, Trash2, Pencil, X, ExternalLink, FileIcon, ZoomIn, Eye, Download } from 'lucide-react'
import { createTenant, getTenantsByRoom, updateTenant, deleteTenant, type Tenant } from '@/api/tenant'
import type { Room } from '@/api/room'

interface TenantRoomModalProps {
  room: Room
  onClose: () => void
  onSuccess: () => void
}

export function TenantRoomModal({ room, onClose, onSuccess }: TenantRoomModalProps) {
  const [tenants, setTenants] = useState<Tenant[]>([])
  const isOccupied = room.status === 'OCCUPIED'
  const [showAddForm, setShowAddForm] = useState(!isOccupied)
  const [editingTenantId, setEditingTenantId] = useState<string | null>(null)
  const [selectedImageUrl, setSelectedImageUrl] = useState<string | null>(null)

  // Form State
  const [fullName, setFullName] = useState('')
  const [phone, setPhone] = useState('')
  const [email, setEmail] = useState('')
  const [identityCard, setIdentityCard] = useState('')
  const [startDate, setStartDate] = useState(new Date().toISOString().split('T')[0])
  const [cccdFile, setCccdFile] = useState<File | null>(null)
  const [contractFile, setContractFile] = useState<File | null>(null)

  const [isLoading, setIsLoading] = useState(false)
  const [isFetchingInfo, setIsFetchingInfo] = useState(isOccupied)

  useEffect(() => {
    if (isOccupied) {
      getTenantsByRoom(room.id)
        .then(res => {
          setTenants(res.data)
          if (res.data.length === 0) {
            setShowAddForm(true)
          }
        })
        .catch(err => console.error("Error fetching tenants", err))
        .finally(() => setIsFetchingInfo(false))
    } else {
      setIsFetchingInfo(false)
    }
  }, [isOccupied, room.id])

  const fetchTenants = () => {
    setIsLoading(true)
    getTenantsByRoom(room.id)
      .then(res => {
        setTenants(res.data)
        if (res.data.length === 0) {
          setShowAddForm(true)
        }
      })
      .catch(err => console.error("Error fetching tenants", err))
      .finally(() => setIsLoading(false))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)
    try {
      const formData = new FormData()
      formData.append('room_id', room.id)
      formData.append('full_name', fullName)
      formData.append('phone', phone)
      if (email) formData.append('email', email)
      formData.append('identity_card', identityCard)
      formData.append('start_date', startDate)
      if (cccdFile) formData.append('cccd_file', cccdFile)
      if (contractFile) formData.append('contract_file', contractFile)
      if (editingTenantId) {
        await updateTenant(editingTenantId, formData)
      } else {
        await createTenant(formData)
      }

      setEditingTenantId(null)
      setShowAddForm(false)
      fetchTenants()
      onSuccess()
      resetForm()
    } catch (error) {
      alert("Lỗi khi xử lý thao tác người thuê!")
    } finally {
      setIsLoading(false)
    }
  }

  const handleDelete = async (tenantId: string) => {
    if (!confirm("Bạn có chắc chắn muốn xóa người thuê này?")) return
    setIsLoading(true)
    try {
      await deleteTenant(tenantId)
      fetchTenants()
      onSuccess()
    } catch (error) {
      alert("Lỗi khi xóa người thuê!")
    } finally {
      setIsLoading(false)
    }
  }

  const handleEditClick = (tenant: Tenant) => {
    setFullName(tenant.full_name)
    setPhone(tenant.phone)
    setEmail(tenant.email || '')
    setIdentityCard(tenant.identity_card)
    setStartDate(tenant.start_date.split('T')[0])
    setCccdFile(null)
    setContractFile(null)
    setEditingTenantId(tenant.id)
  }

  const getFileName = (path?: string) => {
    if (!path) return ''
    const parts = path.split('/')
    return parts[parts.length - 1]
  }

  const isImagePath = (path?: string) => {
    if (!path) return false
    return /\.(jpg|jpeg|png|gif|webp)$/i.test(path)
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
    setFullName('')
    setPhone('')
    setEmail('')
    setIdentityCard('')
    setStartDate(new Date().toISOString().split('T')[0])
    setCccdFile(null)
    setContractFile(null)
    setEditingTenantId(null)
  }

  return (
    <>
      <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/40 backdrop-blur-sm px-4 overflow-y-auto py-10">
        <Card className="w-full max-w-2xl bg-white shadow-xl border-0 animate-in zoom-in-95 duration-200">
          <div className="p-6 border-b border-slate-100 flex items-center gap-3 bg-slate-50/50 rounded-t-xl">
            <div className="w-10 h-10 rounded-full flex items-center justify-center bg-blue-100 text-blue-600">
              {isOccupied ? <User className="w-5 h-5" /> : <UserPlus className="w-5 h-5" />}
            </div>
            <div>
              <h2 className="text-xl font-bold text-slate-800">
                {showAddForm ? 'Thêm người thuê mới' : editingTenantId ? 'Sửa người thuê' : 'Danh sách người thuê'}
              </h2>
              <p className="text-sm text-slate-500">Phòng {room.name} {(!showAddForm && !editingTenantId) && `(${tenants.length}/${room.max_tenants})`}</p>
            </div>
            <button onClick={onClose} className="ml-auto p-2 text-slate-400 hover:text-slate-600 rounded-full hover:bg-slate-100 transition-colors">
              <X className="w-5 h-5" />
            </button>
          </div>

          {isFetchingInfo ? (
            <div className="p-10 text-center text-slate-500">Đang tải thông tin...</div>
          ) : (
            <div className="p-6">
              {(!showAddForm && !editingTenantId) ? (
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
                            <button onClick={() => handleEditClick(t)} className="p-1.5 text-slate-400 hover:text-blue-600 bg-white shadow-sm border border-slate-200 rounded-md transition-colors" title="Sửa">
                              <Pencil className="w-3.5 h-3.5" />
                            </button>
                            <button onClick={() => handleDelete(t.id)} className="p-1.5 text-slate-400 hover:text-red-600 bg-white shadow-sm border border-slate-200 rounded-md transition-colors" title="Xóa">
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
                            <div className="col-span-2 grid grid-cols-1 gap-2 mt-2">
                              {t.cccd_path && (
                                <div className="flex items-center justify-between p-2 rounded-lg bg-white border border-slate-200 group/file hover:border-blue-300 transition-colors">
                                  <div className="flex items-center gap-2 overflow-hidden">
                                    <div className="w-6 h-6 rounded bg-blue-100 text-blue-600 flex items-center justify-center flex-shrink-0">
                                      <FileIcon className="w-3 h-3" />
                                    </div>
                                    <div className="flex flex-col overflow-hidden">
                                      <span className="text-[9px] font-bold text-slate-400 uppercase">Ảnh CCCD</span>
                                      <span className="text-[10px] font-semibold text-slate-600 truncate">{getFileName(t.cccd_path)}</span>
                                    </div>
                                  </div>
                                  <div className="flex items-center gap-1">
                                    <button
                                      onClick={() => handleFileClick(t.cccd_path!)}
                                      className="p-1.5 text-slate-400 hover:text-blue-600 hover:bg-blue-50 rounded-md transition-colors"
                                      title={isImagePath(t.cccd_path) ? "Xem ảnh" : "Tải về"}
                                    >
                                      {isImagePath(t.cccd_path) ? <ZoomIn className="w-3.5 h-3.5" /> : <Download className="w-3.5 h-3.5" />}
                                    </button>
                                    <a
                                      href={`${import.meta.env.VITE_API_BASE_URL || ''}${t.cccd_path}`}
                                      target="_blank"
                                      rel="noopener noreferrer"
                                      download
                                      className="p-1.5 text-slate-400 hover:text-blue-600 hover:bg-blue-50 rounded-md transition-colors"
                                    >
                                      <ExternalLink className="w-3.5 h-3.5" />
                                    </a>
                                  </div>
                                </div>
                              )}
                              {t.contract_path && (
                                <div className="flex items-center justify-between p-2 rounded-lg bg-white border border-slate-200 group/file hover:border-blue-300 transition-colors">
                                  <div className="flex items-center gap-2 overflow-hidden">
                                    <div className="w-6 h-6 rounded bg-blue-100 text-blue-600 flex items-center justify-center flex-shrink-0">
                                      <FileIcon className="w-3 h-3" />
                                    </div>
                                    <div className="flex flex-col overflow-hidden">
                                      <span className="text-[9px] font-bold text-slate-400 uppercase">Hợp đồng</span>
                                      <span className="text-[10px] font-semibold text-slate-600 truncate">{getFileName(t.contract_path)}</span>
                                    </div>
                                  </div>
                                  <div className="flex items-center gap-1">
                                    <button
                                      onClick={() => handleFileClick(t.contract_path!)}
                                      className="p-1.5 text-slate-400 hover:text-blue-600 hover:bg-blue-50 rounded-md transition-colors"
                                      title={isImagePath(t.contract_path) ? "Xem ảnh" : "Tải về"}
                                    >
                                      {isImagePath(t.contract_path) ? <ZoomIn className="w-3.5 h-3.5" /> : <Download className="w-3.5 h-3.5" />}
                                    </button>
                                    <a
                                      href={`${import.meta.env.VITE_API_BASE_URL || ''}${t.contract_path}`}
                                      target="_blank"
                                      rel="noopener noreferrer"
                                      download
                                      className="p-1.5 text-slate-400 hover:text-blue-600 hover:bg-blue-50 rounded-md transition-colors"
                                    >
                                      <ExternalLink className="w-3.5 h-3.5" />
                                    </a>
                                  </div>
                                </div>
                              )}
                            </div>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>

                  <div className="flex justify-between items-center pt-4 border-t border-slate-100">
                    <div className="flex gap-2">
                      {tenants.length < room.max_tenants && (
                        <Button onClick={() => { resetForm(); setShowAddForm(true) }} variant="default" className="bg-blue-600 hover:bg-blue-700 font-bold gap-2">
                          <Plus className="w-4 h-4" /> Thêm người ở
                        </Button>
                      )}
                    </div>
                    <Button onClick={onClose} variant="outline" className="font-bold border-slate-200">Đóng</Button>
                  </div>
                </div>
              ) : (
                <form onSubmit={handleSubmit} className="space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2 col-span-2 md:col-span-1">
                      <Label>Họ và tên <span className="text-red-500">*</span></Label>
                      <Input required value={fullName} onChange={e => setFullName(e.target.value)} placeholder="Nguyễn Văn A" className="border-slate-200" />
                    </div>
                    <div className="space-y-2 col-span-2 md:col-span-1">
                      <Label>Số điện thoại <span className="text-red-500">*</span></Label>
                      <Input required value={phone} onChange={e => setPhone(e.target.value)} placeholder="09..." className="border-slate-200" />
                    </div>
                    <div className="space-y-2 col-span-2 md:col-span-1">
                      <Label>Email (Tùy chọn)</Label>
                      <Input type="email" value={email} onChange={e => setEmail(e.target.value)} placeholder="abc@gmail.com" className="border-slate-200" />
                    </div>
                    <div className="space-y-2 col-span-2 md:col-span-1">
                      <Label>Căn cước công dân <span className="text-red-500">*</span></Label>
                      <Input required value={identityCard} onChange={e => setIdentityCard(e.target.value)} placeholder="12 số CCCD" className="border-slate-200" />
                    </div>
                    <div className="space-y-2 col-span-2 md:col-span-1">
                      <Label>Ngày bắt đầu thuê</Label>
                      <Input type="date" required value={startDate} onChange={e => setStartDate(e.target.value)} className="border-slate-200" />
                    </div>

                    <div className="space-y-4 col-span-2 mt-4 p-4 rounded-xl border border-slate-200 bg-white">
                      <h3 className="text-sm font-bold text-slate-800 mb-2">Tài liệu đính kèm</h3>
                      <div className="grid grid-cols-2 gap-4">
                        {/* CCCD Upload */}
                        <div className="space-y-2 col-span-2 md:col-span-1">
                          <Label>Ảnh CCCD</Label>
                          {!cccdFile ? (
                            <label className="flex flex-col gap-2 items-center justify-center h-24 rounded-xl border-2 border-dashed border-slate-300 bg-slate-50 cursor-pointer hover:bg-slate-100 hover:border-blue-300 transition-colors">
                              <Upload className="w-5 h-5 text-slate-400" />
                              <span className="text-[10px] text-slate-500 font-semibold px-4 text-center">Tải lên ảnh CCCD</span>
                              <input type="file" accept="image/*" className="hidden" onChange={e => setCccdFile(e.target.files?.[0] || null)} />
                            </label>
                          ) : (
                            <div className="flex flex-col gap-2">
                              <div className="flex items-center justify-between h-10 px-3 rounded-lg border border-blue-200 bg-blue-50/50">
                                <div className="flex items-center gap-2 overflow-hidden truncate">
                                  <CheckCircle2 className="w-3.5 h-3.5 text-green-500 flex-shrink-0" />
                                  <span className="text-[10px] font-semibold text-blue-700 truncate">{cccdFile.name}</span>
                                </div>
                                <div className="flex items-center gap-1">
                                  <button type="button" onClick={() => setSelectedImageUrl(URL.createObjectURL(cccdFile))} className="p-1 hover:bg-blue-100 rounded-md text-blue-600" title="Xem trước">
                                    <Eye className="w-4 h-4" />
                                  </button>
                                  <button type="button" onClick={() => setCccdFile(null)} className="p-1 hover:bg-blue-100 rounded-md text-slate-500">
                                    <X className="w-3.5 h-3.5" />
                                  </button>
                                </div>
                              </div>
                              <div
                                className="relative rounded-lg overflow-hidden border border-slate-200 aspect-video bg-slate-100 cursor-pointer group"
                                onClick={() => setSelectedImageUrl(URL.createObjectURL(cccdFile))}
                              >
                                <img src={URL.createObjectURL(cccdFile)} className="w-full h-full object-cover" />
                                <div className="absolute inset-0 bg-black/20 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center text-white">
                                  <ZoomIn className="w-5 h-5" />
                                </div>
                              </div>
                            </div>
                          )}
                        </div>

                        {/* Contract Upload */}
                        <div className="space-y-2 col-span-2 md:col-span-1">
                          <Label>Hợp đồng</Label>
                          {!contractFile ? (
                            <label className="flex flex-col gap-2 items-center justify-center h-24 rounded-xl border-2 border-dashed border-slate-300 bg-slate-50 cursor-pointer hover:bg-slate-100 hover:border-blue-300 transition-colors">
                              <Upload className="w-5 h-5 text-slate-400" />
                              <span className="text-[10px] text-slate-500 font-semibold px-4 text-center">Tải lên hợp đồng</span>
                              <input type="file" accept=".pdf,.doc,.docx,image/*" className="hidden" onChange={e => setContractFile(e.target.files?.[0] || null)} />
                            </label>
                          ) : (
                            <div className="flex flex-col gap-2">
                              <div className="flex items-center justify-between h-10 px-3 rounded-lg border border-blue-200 bg-blue-50/50">
                                <div className="flex items-center gap-2 overflow-hidden truncate">
                                  <CheckCircle2 className="w-3.5 h-3.5 text-green-500 flex-shrink-0" />
                                  <span className="text-[10px] font-semibold text-blue-700 truncate">{contractFile.name}</span>
                                </div>
                                <div className="flex items-center gap-1">
                                  {contractFile.type.startsWith('image/') && (
                                    <button type="button" onClick={() => setSelectedImageUrl(URL.createObjectURL(contractFile))} className="p-1 hover:bg-blue-100 rounded-md text-blue-600" title="Xem trước">
                                      <Eye className="w-4 h-4" />
                                    </button>
                                  )}
                                  <button type="button" onClick={() => setContractFile(null)} className="p-1 hover:bg-blue-100 rounded-md text-slate-500">
                                    <X className="w-3.5 h-3.5" />
                                  </button>
                                </div>
                              </div>
                              {contractFile.type.startsWith('image/') ? (
                                <div
                                  className="relative rounded-lg overflow-hidden border border-slate-200 aspect-video bg-slate-100 cursor-pointer group"
                                  onClick={() => setSelectedImageUrl(URL.createObjectURL(contractFile))}
                                >
                                  <img src={URL.createObjectURL(contractFile)} className="w-full h-full object-cover" />
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
                    <Button type="submit" disabled={isLoading} className="bg-blue-600 hover:bg-blue-700 text-white font-bold shadow-md shadow-blue-200 transition-all">
                      {isLoading ? 'Đang lưu...' : (editingTenantId ? 'Lưu thay đổi' : 'Xác nhận')}
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
