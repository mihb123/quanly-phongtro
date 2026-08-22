import { useState, useEffect } from 'react'
import { AppModal } from '@/components/shared/AppModal'
import { Button } from '@/components/ui/button'
import { Plus, Trash2, Pencil, ExternalLink, FileIcon, ZoomIn, Download, CheckCircle2, Loader2 } from '@/components/icons'
import type { Tenant } from '@/api/tenant'
import type { Room } from '@/api/room'
import { getFileName, isImagePath } from '@/utils/file'
import { getProtectedFileObjectUrl, openProtectedFile } from '@/api/files'
import { ImageLightboxModal } from '@/components/shared/ImageLightboxModal'
import { toast } from 'sonner'

import { useTenantList } from '@/hooks/useTenantList'

interface TenantListModalProps {
  room: Room
  onClose: () => void
  onAdd: () => void
  onEdit: (tenant: Tenant) => void
  onDataChange?: () => void
}

// Modal danh sách người thuê. Vỏ dùng AppModal; giữ nguyên hook useTenantList, xóa/sửa và ImageLightboxModal xem trước file.
export function TenantListModal({ room, onClose, onAdd, onEdit, onDataChange }: TenantListModalProps) {
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
    if (onDataChange) onDataChange()
    if (tenants.length <= 1) {
      onClose()
    }
  }

  const [previewImage, setPreviewImage] = useState<{ url: string; title: string; filename: string } | null>(null)
  const [loadingFilePath, setLoadingFilePath] = useState<string | null>(null)

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

  const handlePreviewFile = async (path: string, label?: string) => {
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

  const handleOpenFileExternal = async (path: string) => {
    setLoadingFilePath(path)
    try {
      await openProtectedFile(path)
    } catch (error) {
      console.error('Không mở được file tenant:', error)
      toast.error('Không thể mở file, vui lòng thử lại!')
    } finally {
      setLoadingFilePath(null)
    }
  }

  return (
    <>
      <AppModal
        open
        onClose={onClose}
        title="Danh sách người thuê"
        description={`Phòng ${room.name} (${tenants.length}/${room.max_tenants})`}
        contentClassName="sm:max-w-xl"
        footer={
          <div className="flex w-full justify-between items-center">
            <div className="flex gap-2">
              {tenants.length < room.max_tenants && (
                <Button onClick={onAdd} variant="default">
                  <Plus className="w-4 h-4" /> Thêm người ở
                </Button>
              )}
            </div>
            <Button onClick={onClose} variant="outline">Đóng</Button>
          </div>
        }
      >
        {isFetching ? (
          <div className="p-10 text-center text-muted-foreground">Đang tải thông tin...</div>
        ) : (
          <div className="space-y-4">
            {tenants.map((t, idx) => (
              <div key={t.id} className="p-4 rounded-lg border border-border/50 bg-muted/30 group">
                <div className="flex justify-between items-start mb-3 gap-2">
                  <div className="flex items-center gap-2 min-w-0 flex-1">
                    <div className="w-8 h-8 rounded-full bg-primary/10 text-primary flex items-center justify-center text-xs font-medium shrink-0">
                      {idx + 1}
                    </div>
                    <h4 className="font-medium text-foreground break-words min-w-0">{t.full_name}</h4>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
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
                    <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-medium bg-success/10 text-success ml-1 shrink-0">
                      <CheckCircle2 className="w-3 h-3" />
                      Đang ở
                    </span>
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-y-3 gap-x-4">
                  <div className="flex flex-col min-w-0">
                    <span className="text-xs uppercase font-medium text-muted-foreground">Số điện thoại</span>
                    <span className="text-sm font-semibold text-foreground break-all">{t.phone || 'N/A'}</span>
                  </div>
                  <div className="flex flex-col min-w-0">
                    <span className="text-xs uppercase font-medium text-muted-foreground">Email</span>
                    <span className="text-sm font-semibold text-foreground break-all">{t.email || 'N/A'}</span>
                  </div>
                  <div className="flex flex-col min-w-0">
                    <span className="text-xs uppercase font-medium text-muted-foreground">CCCD</span>
                    <span className="text-sm font-semibold text-foreground break-all">{t.identity_card || 'N/A'}</span>
                  </div>
                  <div className="flex flex-col min-w-0">
                    <span className="text-xs uppercase font-medium text-muted-foreground">Ngày bắt đầu</span>
                    <span className="text-sm font-semibold text-foreground">{t.start_date ? new Date(t.start_date).toLocaleDateString('vi-VN') : 'N/A'}</span>
                  </div>

                  {(t.cccd_path || t.contract_path) && (
                    <div className="col-span-2 grid grid-cols-1 sm:grid-cols-2 gap-2 mt-2 max-h-80 overflow-y-auto pr-1">
                      {t.cccd_path && t.cccd_path.split(',').filter(Boolean).map((path, fileIdx) => {
                        const label = `Ảnh CCCD ${fileIdx > 0 ? fileIdx + 1 : ''}`.trim()
                        const isImg = isImagePath(path)
                        const isLoading = loadingFilePath === path
                        return (
                          <div
                            key={`cccd-${fileIdx}`}
                            className="flex items-center justify-between p-2 rounded-lg bg-background border border-border/50 group/file hover:border-primary/40 transition-colors cursor-pointer min-w-0"
                            onClick={() => handlePreviewFile(path, label)}
                          >
                            <div className="flex items-center gap-2 overflow-hidden min-w-0 flex-1">
                              <div className="w-6 h-6 rounded bg-primary/10 text-primary flex items-center justify-center shrink-0">
                                <FileIcon className="w-3 h-3" />
                              </div>
                              <div className="flex flex-col overflow-hidden min-w-0">
                                <span className="text-xs font-medium text-muted-foreground">{label}</span>
                                <span className="text-xs font-semibold text-foreground truncate">{getFileName(path)}</span>
                              </div>
                            </div>
                            <div className="flex items-center gap-1 shrink-0" onClick={(e) => e.stopPropagation()}>
                              <button
                                type="button"
                                onClick={() => handlePreviewFile(path, label)}
                                disabled={isLoading}
                                className="p-1.5 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-md transition-colors cursor-pointer disabled:opacity-50"
                                title={isImg ? "Xem ảnh" : "Tải về"}
                              >
                                {isLoading ? (
                                  <Loader2 className="w-3.5 h-3.5 animate-spin text-primary" />
                                ) : isImg ? (
                                  <ZoomIn className="w-3.5 h-3.5" />
                                ) : (
                                  <Download className="w-3.5 h-3.5" />
                                )}
                              </button>
                              <button
                                type="button"
                                onClick={() => handleOpenFileExternal(path)}
                                disabled={isLoading}
                                className="p-1.5 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-md transition-colors cursor-pointer disabled:opacity-50"
                                title="Mở trong tab mới"
                              >
                                <ExternalLink className="w-3.5 h-3.5" />
                              </button>
                            </div>
                          </div>
                        )
                      })}
                      {t.contract_path && t.contract_path.split(',').filter(Boolean).map((path, fileIdx) => {
                        const label = `Hợp đồng ${fileIdx > 0 ? fileIdx + 1 : ''}`.trim()
                        const isImg = isImagePath(path)
                        const isLoading = loadingFilePath === path
                        return (
                          <div
                            key={`contract-${fileIdx}`}
                            className="flex items-center justify-between p-2 rounded-lg bg-background border border-border/50 group/file hover:border-primary/40 transition-colors cursor-pointer min-w-0"
                            onClick={() => handlePreviewFile(path, label)}
                          >
                            <div className="flex items-center gap-2 overflow-hidden min-w-0 flex-1">
                              <div className="w-6 h-6 rounded bg-primary/10 text-primary flex items-center justify-center shrink-0">
                                <FileIcon className="w-3 h-3" />
                              </div>
                              <div className="flex flex-col overflow-hidden min-w-0">
                                <span className="text-xs font-medium text-muted-foreground">{label}</span>
                                <span className="text-xs font-semibold text-foreground truncate">{getFileName(path)}</span>
                              </div>
                            </div>
                            <div className="flex items-center gap-1 shrink-0" onClick={(e) => e.stopPropagation()}>
                              <button
                                type="button"
                                onClick={() => handlePreviewFile(path, label)}
                                disabled={isLoading}
                                className="p-1.5 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-md transition-colors cursor-pointer disabled:opacity-50"
                                title={isImg ? "Xem ảnh" : "Tải về"}
                              >
                                {isLoading ? (
                                  <Loader2 className="w-3.5 h-3.5 animate-spin text-primary" />
                                ) : isImg ? (
                                  <ZoomIn className="w-3.5 h-3.5" />
                                ) : (
                                  <Download className="w-3.5 h-3.5" />
                                )}
                              </button>
                              <button
                                type="button"
                                onClick={() => handleOpenFileExternal(path)}
                                disabled={isLoading}
                                className="p-1.5 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-md transition-colors cursor-pointer disabled:opacity-50"
                                title="Mở trong tab mới"
                              >
                                <ExternalLink className="w-3.5 h-3.5" />
                              </button>
                            </div>
                          </div>
                        )
                      })}
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
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
