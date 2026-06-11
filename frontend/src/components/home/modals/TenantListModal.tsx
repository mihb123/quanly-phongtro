import { useState, useEffect } from 'react'
import { createPortal } from 'react-dom'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { User, Plus, Trash2, Pencil, X, ExternalLink, FileIcon, ZoomIn, Download, CheckCircle2 } from 'lucide-react'
import type { Tenant } from '@/api/tenant'
import type { Room } from '@/api/room'
import { getFileName, isImagePath } from '@/utils/file'

import { useTenantList } from '@/hooks/useTenantList'

interface TenantListModalProps {
  room: Room
  onClose: () => void
  onAdd: () => void
  onEdit: (tenant: Tenant) => void
}

export function TenantListModal({ room, onClose, onAdd, onEdit }: TenantListModalProps) {
  const {
    tenants,
    isLoading: isDeleting,
    isFetching,
    fetchTenants,
    handleDeleteTenant
  } = useTenantList(room.id)

  useEffect(() => {
    fetchTenants(room.id)
  }, [room.id, fetchTenants])

  const onDelete = async (tenantId: string) => {
    await handleDeleteTenant(tenantId, room.id)
    if (tenants.length <= 1) {
      onClose()
    }
  }
  const [selectedImageUrl, setSelectedImageUrl] = useState<string | null>(null)

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

  const handleFileClick = (path: string) => {
    const fullUrl = `${import.meta.env.VITE_API_BASE_URL || ''}${path}`
    if (isImagePath(path)) {
      setSelectedImageUrl(fullUrl)
    } else {
      window.open(fullUrl, '_blank')
    }
  }

  return createPortal(
    <>
      <div 
        className="fixed inset-0 z-[60] flex items-center justify-center bg-background/80 backdrop-blur-sm p-4"
        onClick={onClose}
      >
        <Card 
          className="w-full max-w-2xl bg-card text-card-foreground shadow-xl border border-border/40 safe-fade-in max-h-[95vh] flex flex-col overflow-y-auto"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="p-6 border-b border-border/40 flex items-center gap-3 bg-muted/30 rounded-t-xl shrink-0">
            <div className="w-10 h-10 rounded-full flex items-center justify-center bg-primary/10 text-primary">
              <User className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-xl font-bold text-foreground">Danh sách người thuê</h2>
              <p className="text-sm text-muted-foreground">Phòng {room.name} ({tenants.length}/{room.max_tenants})</p>
            </div>
            <button onClick={onClose} className="ml-auto p-2 text-muted-foreground hover:text-foreground rounded-full hover:bg-secondary transition-colors cursor-pointer">
              <X className="w-5 h-5" />
            </button>
          </div>

          {isFetching ? (
            <div className="p-10 text-center text-muted-foreground flex-1">Đang tải thông tin...</div>
          ) : (
            <div className="p-6 overflow-y-auto flex-1">
              <div className="space-y-6">
                <div className="space-y-4">
                  {tenants.map((t, idx) => (
                    <div key={t.id} className="p-4 rounded-xl border border-border/50 bg-muted/30 group">
                      <div className="flex justify-between items-start mb-3">
                        <div className="flex items-center gap-2">
                          <div className="w-8 h-8 rounded-full bg-primary/10 text-primary flex items-center justify-center text-xs font-bold">
                            {idx + 1}
                          </div>
                          <h4 className="font-bold text-foreground">{t.full_name}</h4>
                        </div>
                        <div className="flex items-center gap-2">
                          <button onClick={() => onEdit(t)} className="p-1.5 text-muted-foreground hover:text-primary bg-background shadow-sm border border-border/50 rounded-md transition-colors cursor-pointer" title="Sửa">
                            <Pencil className="w-3.5 h-3.5" />
                          </button>
                          <button disabled={isDeleting} onClick={() => {
                            if (confirm("Bạn có chắc chắn muốn xóa người thuê này?")) {
                              onDelete(t.id)
                            }
                          }} className="p-1.5 text-muted-foreground hover:text-destructive bg-background shadow-sm border border-border/50 rounded-md transition-colors cursor-pointer disabled:opacity-50" title="Xóa">
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
                          <span className="text-[10px] uppercase font-bold text-muted-foreground tracking-wider">Số điện thoại</span>
                          <span className="text-sm font-semibold text-foreground">{t.phone}</span>
                        </div>
                        <div className="flex flex-col">
                          <span className="text-[10px] uppercase font-bold text-muted-foreground tracking-wider">Email</span>
                          <span className="text-sm font-semibold text-foreground">{t.email || 'N/A'}</span>
                        </div>
                        <div className="flex flex-col">
                          <span className="text-[10px] uppercase font-bold text-muted-foreground tracking-wider">CCCD</span>
                          <span className="text-sm font-semibold text-foreground">{t.identity_card}</span>
                        </div>
                        <div className="flex flex-col">
                          <span className="text-[10px] uppercase font-bold text-muted-foreground tracking-wider">Ngày bắt đầu</span>
                          <span className="text-sm font-semibold text-foreground">{new Date(t.start_date).toLocaleDateString('vi-VN')}</span>
                        </div>

                        {(t.cccd_path || t.contract_path) && (
                          <div className="col-span-2 grid grid-cols-2 gap-2 mt-2 max-h-80 overflow-y-auto pr-1">
                            {t.cccd_path && t.cccd_path.split(',').filter(Boolean).map((path, fileIdx) => (
                              <div key={`cccd-${fileIdx}`} className="flex items-center justify-between p-2 rounded-lg bg-background border border-border/50 group/file hover:border-primary/40 transition-colors">
                                <div className="flex items-center gap-2 overflow-hidden">
                                  <div className="w-6 h-6 rounded bg-primary/10 text-primary flex items-center justify-center flex-shrink-0">
                                    <FileIcon className="w-3 h-3" />
                                  </div>
                                  <div className="flex flex-col overflow-hidden">
                                    <span className="text-[9px] font-bold text-muted-foreground uppercase">Ảnh CCCD {fileIdx > 0 ? fileIdx + 1 : ''}</span>
                                    <span className="text-[10px] font-semibold text-foreground truncate">{getFileName(path)}</span>
                                  </div>
                                </div>
                                <div className="flex items-center gap-1">
                                  <button
                                    onClick={() => handleFileClick(path)}
                                    className="p-1.5 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-md transition-colors cursor-pointer"
                                    title={isImagePath(path) ? "Xem ảnh" : "Tải về"}
                                  >
                                    {isImagePath(path) ? <ZoomIn className="w-3.5 h-3.5" /> : <Download className="w-3.5 h-3.5 cursor-pointer" />}
                                  </button>
                                  <a
                                    href={`${import.meta.env.VITE_API_BASE_URL || ''}${path}`}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    download
                                    className="p-1.5 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-md transition-colors"
                                  >
                                    <ExternalLink className="w-3.5 h-3.5" />
                                  </a>
                                </div>
                              </div>
                            ))}
                            {t.contract_path && t.contract_path.split(',').filter(Boolean).map((path, fileIdx) => (
                              <div key={`contract-${fileIdx}`} className="flex items-center justify-between p-2 rounded-lg bg-background border border-border/50 group/file hover:border-primary/40 transition-colors">
                                <div className="flex items-center gap-2 overflow-hidden">
                                  <div className="w-6 h-6 rounded bg-primary/10 text-primary flex items-center justify-center flex-shrink-0">
                                    <FileIcon className="w-3 h-3" />
                                  </div>
                                  <div className="flex flex-col overflow-hidden">
                                    <span className="text-[9px] font-bold text-muted-foreground uppercase">Hợp đồng {fileIdx > 0 ? fileIdx + 1 : ''}</span>
                                    <span className="text-[10px] font-semibold text-foreground truncate">{getFileName(path)}</span>
                                  </div>
                                </div>
                                <div className="flex items-center gap-1">
                                  <button
                                    onClick={() => handleFileClick(path)}
                                    className="p-1.5 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-md transition-colors cursor-pointer"
                                    title={isImagePath(path) ? "Xem ảnh" : "Tải về"}
                                  >
                                    {isImagePath(path) ? <ZoomIn className="w-3.5 h-3.5" /> : <Download className="w-3.5 h-3.5" />}
                                  </button>
                                  <a
                                    href={`${import.meta.env.VITE_API_BASE_URL || ''}${path}`}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    download
                                    className="p-1.5 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-md transition-colors"
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

                <div className="flex justify-between items-center pt-4 border-t border-border/40">
                  <div className="flex gap-2">
                    {tenants.length < room.max_tenants && (
                      <Button onClick={onAdd} variant="default" className="shadow-sm font-bold gap-2 cursor-pointer">
                        <Plus className="w-4 h-4" /> Thêm người ở
                      </Button>
                    )}
                  </div>
                  <Button onClick={onClose} variant="outline" className="font-bold cursor-pointer">Đóng</Button>
                </div>
              </div>
            </div>
          )}
        </Card>
      </div>

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
    </>,
    document.body
  )
}
