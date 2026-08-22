import { useState } from 'react'
import { toast } from 'sonner'
import type { House } from '@/api/house'
import { clearProtectedFileCache } from '@/api/files'
import { useHouseStore } from '@/data/houseData'
import { useSelectedStore } from '@/data/selectedData'
import { getFileName } from '@/utils/file'
import { compressImagesForUpload } from '@/services/imageCompression'
import { ImageLightboxModal, type LightboxImageItem } from '@/components/shared/ImageLightboxModal'
import { FormSubGroup } from './FormSection'
import { DocumentUploadGrid } from './DocumentUploadGrid'
import { ConfirmModal } from './ConfirmModal'

type DocumentKind = 'cccd' | 'contract'

const FIELD_BY_KIND: Record<DocumentKind, { file: string; kept: string; label: string }> = {
  cccd: { file: 'owner_cccd_file', kept: 'kept_owner_cccd_paths', label: 'CCCD chủ nhà' },
  contract: { file: 'owner_contract_file', kept: 'kept_owner_contract_paths', label: 'Hợp đồng thuê nhà' },
}

const splitPaths = (raw?: string) => (raw ? raw.split(',').filter(Boolean) : [])

// Hồ sơ thuê nguyên căn của nhà: CCCD chủ nhà + hợp đồng thuê. File lưu ngay khi tải lên hoặc xóa.
// Render dưới dạng các nhóm nhỏ để nằm gọn trong khối "Thông tin thuê nhà" của form cha.
export function HouseDocumentsSection({ house }: { house: House }) {
  const updateHouseDocuments = useHouseStore(state => state.updateHouseDocuments)
  const { selectedHouse, selectHouse } = useSelectedStore()

  const [cccdPaths, setCccdPaths] = useState<string[]>(() => splitPaths(house.owner_cccd_path))
  const [contractPaths, setContractPaths] = useState<string[]>(() => splitPaths(house.owner_contract_path))
  const [busyKind, setBusyKind] = useState<DocumentKind | null>(null)
  const [pendingDelete, setPendingDelete] = useState<{ kind: DocumentKind; path: string } | null>(null)
  const [isDeleting, setIsDeleting] = useState(false)
  const [gallery, setGallery] = useState<{ items: LightboxImageItem[]; initialIndex: number } | null>(null)

  const applyResult = (updated: House) => {
    setCccdPaths(splitPaths(updated.owner_cccd_path))
    setContractPaths(splitPaths(updated.owner_contract_path))
    if (selectedHouse?.id === updated.id) selectHouse(updated)
  }

  // buildFormData luôn gửi cả hai nhóm để backend biết chính xác file nào được giữ lại.
  const buildFormData = (nextCccd: string[], nextContract: string[]) => {
    const formData = new FormData()
    formData.append(FIELD_BY_KIND.cccd.kept, nextCccd.join(','))
    formData.append(`${FIELD_BY_KIND.cccd.kept}_empty`, nextCccd.length === 0 ? 'true' : 'false')
    formData.append(FIELD_BY_KIND.contract.kept, nextContract.join(','))
    formData.append(`${FIELD_BY_KIND.contract.kept}_empty`, nextContract.length === 0 ? 'true' : 'false')
    return formData
  }

  const handleUpload = async (kind: DocumentKind, files: FileList | null) => {
    const incoming = files ? Array.from(files) : []
    if (incoming.length === 0) return

    setBusyKind(kind)
    try {
      const images = incoming.filter(f => f.type.startsWith('image/'))
      const others = incoming.filter(f => !f.type.startsWith('image/'))
      const compressed = images.length > 0 ? (await compressImagesForUpload(images)).map(c => c.file) : []

      const formData = buildFormData(cccdPaths, contractPaths)
      ;[...compressed, ...others].forEach(f => formData.append(FIELD_BY_KIND[kind].file, f))

      const res = await updateHouseDocuments(house.id, formData)
      if (res.success && res.house) {
        applyResult(res.house)
        toast.success(`Đã lưu ${FIELD_BY_KIND[kind].label.toLowerCase()}!`)
      } else {
        toast.error(res.error || 'Lỗi khi lưu hồ sơ thuê nhà!')
      }
    } catch (error) {
      console.error('Lỗi khi tải hồ sơ thuê nhà:', error)
      toast.error('Lỗi khi tải lên file!')
    } finally {
      setBusyKind(null)
    }
  }

  const handleConfirmDelete = async () => {
    if (!pendingDelete) return
    const { kind, path } = pendingDelete

    setIsDeleting(true)
    try {
      const nextCccd = kind === 'cccd' ? cccdPaths.filter(p => p !== path) : cccdPaths
      const nextContract = kind === 'contract' ? contractPaths.filter(p => p !== path) : contractPaths

      const res = await updateHouseDocuments(house.id, buildFormData(nextCccd, nextContract))
      if (res.success && res.house) {
        applyResult(res.house)
        clearProtectedFileCache(path)
        toast.success('Đã xóa file thành công!')
        setPendingDelete(null)
      } else {
        toast.error(res.error || 'Lỗi khi xóa file!')
      }
    } catch (error) {
      console.error('Lỗi khi xóa hồ sơ thuê nhà:', error)
      toast.error('Lỗi khi xóa file!')
    } finally {
      setIsDeleting(false)
    }
  }

  const isBusy = (kind: DocumentKind) => busyKind === kind || (isDeleting && pendingDelete?.kind === kind)

  return (
    <>
      <FormSubGroup label="CCCD chủ nhà">
        <DocumentUploadGrid
          itemLabel="CCCD chủ nhà"
          paths={cccdPaths}
          isBusy={isBusy('cccd')}
          emptyActionLabel="Tải CCCD"
          onAddFiles={files => void handleUpload('cccd', files)}
          onRequestDelete={path => setPendingDelete({ kind: 'cccd', path })}
          onPreview={(items, initialIndex) => setGallery({ items, initialIndex })}
        />
      </FormSubGroup>

      <FormSubGroup label="Hợp đồng thuê nhà">
        <DocumentUploadGrid
          itemLabel="Hợp đồng thuê nhà"
          paths={contractPaths}
          isBusy={isBusy('contract')}
          emptyActionLabel="Tải hợp đồng"
          onAddFiles={files => void handleUpload('contract', files)}
          onRequestDelete={path => setPendingDelete({ kind: 'contract', path })}
          onPreview={(items, initialIndex) => setGallery({ items, initialIndex })}
        />
      </FormSubGroup>

      {pendingDelete && (
        <ConfirmModal
          title="Xác nhận xóa file"
          message={`Bạn có chắc chắn muốn xóa file "${getFileName(pendingDelete.path)}"? Thay đổi sẽ được lưu ngay lập tức.`}
          confirmText="Xóa file"
          cancelText="Hủy"
          isLoading={isDeleting}
          onConfirm={handleConfirmDelete}
          onCancel={() => {
            if (!isDeleting) setPendingDelete(null)
          }}
        />
      )}

      <ImageLightboxModal
        isOpen={Boolean(gallery)}
        images={gallery?.items}
        initialIndex={gallery?.initialIndex ?? 0}
        onClose={() => setGallery(null)}
      />
    </>
  )
}
