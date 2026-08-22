import { useState, useEffect } from 'react'
import { AppModal } from '@/components/shared/AppModal'
import { Button } from '@/components/ui/button'
import { Plus, Trash2, Pencil, FileIcon, CheckCircle2 } from '@/components/icons'
import type { Tenant } from '@/api/tenant'
import type { Room } from '@/api/room'
import { getFileName } from '@/utils/file'
import { ImageLightboxModal, type LightboxImageItem } from '@/components/shared/ImageLightboxModal'
import { VerifiedBadge } from '@/components/shared/VerifiedBadge'

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

  const [gallery, setGallery] = useState<{ items: LightboxImageItem[]; initialIndex: number } | null>(null)

  const handleOpenGallery = (items: LightboxImageItem[], initialIndex: number) => {
    setGallery({ items, initialIndex })
  }

  const handleCloseGallery = () => {
    setGallery(null)
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
            {tenants.map((t, idx) => {
              const cccdList: LightboxImageItem[] = t.cccd_path
                ? t.cccd_path.split(',').filter(Boolean).map((p, pIdx) => ({
                    path: p,
                    title: `Ảnh CCCD ${pIdx + 1} (${t.full_name})`,
                    filename: getFileName(p),
                  }))
                : []

              return (
                <div key={t.id} className="p-4 rounded-lg border border-border/50 bg-muted/30 group">
                  <div className="flex justify-between items-start mb-3 gap-2">
                    <div className="flex items-center gap-2 min-w-0 flex-1">
                      <div className="w-8 h-8 rounded-full bg-primary/10 text-primary flex items-center justify-center text-xs font-medium shrink-0">
                        {idx + 1}
                      </div>
                      <h4 className="font-medium text-foreground break-words min-w-0">{t.full_name}</h4>
                      {cccdList.length > 0 && <VerifiedBadge title="Đã tải ảnh CCCD" />}
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

                    {cccdList.length > 0 && (
                      <div className="col-span-2 grid grid-cols-1 sm:grid-cols-2 gap-2 mt-2 max-h-80 overflow-y-auto pr-1">
                        {cccdList.map((item, fileIdx) => {
                          const label = `Ảnh CCCD ${fileIdx > 0 ? fileIdx + 1 : ''}`.trim()
                          return (
                            <div
                              key={`cccd-${fileIdx}`}
                              className="flex items-center justify-between p-2 rounded-lg bg-background border border-border/50 group/file hover:border-primary/50 hover:bg-primary/5 transition-all cursor-pointer min-w-0"
                              onClick={() => handleOpenGallery(cccdList, fileIdx)}
                              title="Nhấn để xem ảnh CCCD"
                            >
                              <div className="flex items-center gap-2 overflow-hidden min-w-0 flex-1">
                                <div className="w-6 h-6 rounded bg-primary/10 text-primary flex items-center justify-center shrink-0">
                                  <FileIcon className="w-3 h-3" />
                                </div>
                                <div className="flex flex-col overflow-hidden min-w-0">
                                  <span className="text-xs font-medium text-muted-foreground">{label}</span>
                                  <span className="text-xs font-semibold text-foreground truncate">{item.filename}</span>
                                </div>
                              </div>
                            </div>
                          )
                        })}
                      </div>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </AppModal>

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
