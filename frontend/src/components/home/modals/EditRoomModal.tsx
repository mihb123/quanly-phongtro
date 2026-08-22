import { useState } from 'react'
import { AppModal } from '@/components/shared/AppModal'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Upload, CheckCircle2, X, FileIcon, ZoomIn, Eye, Loader2 } from '@/components/icons'
import type { Room } from '@/api/room'
import { useRoomStore } from '@/data/roomData'
import { useSelectedStore } from '@/data/selectedData'
import { useForm, Controller } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { getFileName, isImagePath } from '@/utils/file'
import { ProtectedFileImage } from './ProtectedFileImage'
import { ImageLightboxModal, type LightboxImageItem } from '@/components/shared/ImageLightboxModal'
import { formatNumber, parseNumber } from '@/utils/format'
import { useDirtyConfirm } from '@/hooks/useDirtyConfirm'
import { ConfirmModal } from './ConfirmModal'
import { compressImagesForUpload } from '@/services/imageCompression'
import { toast } from 'sonner'

const roomSchema = z.object({
  name: z.string().min(1, 'Tên phòng không được để trống'),
  price: z.string().min(1, 'Giá phòng không được để trống'),
  maxTenants: z.string().min(1, 'Sức chứa không được để trống'),
  electricity: z.string().optional(),
  water: z.string().optional(),
  wifi: z.string().optional(),
  parking: z.string().optional(),
  service: z.string().optional(),
  extraPersonThreshold: z.string().optional(),
  extraPersonFee: z.string().optional(),
  extraVehicleThreshold: z.string().optional(),
  extraVehicleFee: z.string().optional(),
  groupChatId: z.string().optional(),
})

type RoomFormValues = z.infer<typeof roomSchema>

const splitPaths = (raw?: string) => (raw ? raw.split(',').filter(Boolean) : [])

// Modal sửa thông tin phòng (giá, sức chứa, giá phát sinh riêng, Zalo bot, hợp đồng lưu tự động khi upload/xóa).
export function EditRoomModal({ room, onClose }: { room: Room, onClose: () => void }) {
  const house = useSelectedStore(state => state.selectedHouse)
  const updateRoomStore = useRoomStore(state => state.updateRoom)
  const updateRoomContractStore = useRoomStore(state => state.updateRoomContract)

  const [isLoading, setIsLoading] = useState(false)
  const [existingContractPaths, setExistingContractPaths] = useState<string[]>(() => splitPaths(room.contract_path))
  const [gallery, setGallery] = useState<{ items: LightboxImageItem[]; initialIndex: number } | null>(null)

  const [isUploadingContract, setIsUploadingContract] = useState(false)
  const [contractToDelete, setContractToDelete] = useState<{ path: string; index: number } | null>(null)
  const [isDeletingContract, setIsDeletingContract] = useState(false)

  const contractGallery: LightboxImageItem[] = existingContractPaths.map((path, idx) => ({
    path,
    title: `Hợp đồng ${idx + 1}`,
    filename: getFileName(path),
  }))

  const handleOpenGallery = (items: LightboxImageItem[], initialIndex: number) => setGallery({ items, initialIndex })

  const { register, handleSubmit, control, formState: { errors, isDirty } } = useForm<RoomFormValues>({
    resolver: zodResolver(roomSchema),
    defaultValues: {
      name: room.name || '',
      price: room.price?.toString() || '0',
      maxTenants: room.max_tenants?.toString() || '2',
      electricity: room.electricity_price?.toString() || house?.default_electricity_price?.toString() || '',
      water: room.water_price?.toString() || house?.default_water_price?.toString() || '',
      wifi: room.wifi_price?.toString() || house?.default_wifi_price?.toString() || '',
      parking: room.parking_price?.toString() || house?.default_parking_price?.toString() || '',
      service: room.service_price?.toString() || house?.default_service_price?.toString() || '',
      extraPersonThreshold: room.extra_person_threshold?.toString() || '',
      extraPersonFee: room.extra_person_fee?.toString() || '',
      extraVehicleThreshold: room.extra_vehicle_threshold?.toString() || '',
      extraVehicleFee: room.extra_vehicle_fee?.toString() || '',
      groupChatId: room.group_chat_id || '',
    }
  })

  const { handleClose, confirmModal } = useDirtyConfirm(isDirty, onClose, isLoading)

  if (!house) return null

  const handleUploadContract = async (files: FileList | null) => {
    const incoming = files ? Array.from(files) : []
    if (incoming.length === 0) return

    setIsUploadingContract(true)
    try {
      const imageFiles = incoming.filter(f => f.type.startsWith('image/'))
      const otherFiles = incoming.filter(f => !f.type.startsWith('image/'))

      let processedImages: File[] = []
      if (imageFiles.length > 0) {
        const compressed = await compressImagesForUpload(imageFiles)
        processedImages = compressed.map(c => c.file)
      }
      const finalFiles = [...processedImages, ...otherFiles]

      const formData = new FormData()
      formData.append('house_id', house.id)
      formData.append('kept_contract_paths', existingContractPaths.join(','))
      formData.append('kept_contract_paths_empty', existingContractPaths.length === 0 ? 'true' : 'false')
      finalFiles.forEach(f => formData.append('contract_file', f))

      const res = await updateRoomContractStore(room.id, formData)
      if (res.success && res.data) {
        const newPaths = splitPaths(res.data.contract_path)
        setExistingContractPaths(newPaths)
        toast.success('Tải lên và lưu hợp đồng thuê thành công!')
      } else if (res.success) {
        const updatedRoom = useRoomStore.getState().rooms.find(r => r.id === room.id)
        if (updatedRoom?.contract_path) {
          setExistingContractPaths(splitPaths(updatedRoom.contract_path))
        }
        toast.success('Tải lên và lưu hợp đồng thuê thành công!')
      } else {
        toast.error(res.error || 'Lỗi khi lưu hợp đồng thuê!')
      }
    } catch (error) {
      console.error('Lỗi khi tải hợp đồng thuê:', error)
      toast.error('Lỗi khi tải lên hợp đồng thuê!')
    } finally {
      setIsUploadingContract(false)
    }
  }

  const handleConfirmDeleteContract = async () => {
    if (!contractToDelete) return

    setIsDeletingContract(true)
    try {
      const newPaths = existingContractPaths.filter((_, i) => i !== contractToDelete.index)
      const formData = new FormData()
      formData.append('house_id', house.id)
      formData.append('kept_contract_paths', newPaths.join(','))
      formData.append('kept_contract_paths_empty', newPaths.length === 0 ? 'true' : 'false')

      const res = await updateRoomContractStore(room.id, formData)
      if (res.success) {
        setExistingContractPaths(newPaths)
        toast.success('Đã xóa file hợp đồng thành công!')
        setContractToDelete(null)
      } else {
        toast.error(res.error || 'Lỗi khi xóa hợp đồng thuê!')
      }
    } catch (error) {
      console.error('Lỗi khi xóa hợp đồng:', error)
      toast.error('Lỗi khi xóa file hợp đồng!')
    } finally {
      setIsDeletingContract(false)
    }
  }

  const onSubmit = async (values: RoomFormValues) => {
    setIsLoading(true)
    try {
      const res = await updateRoomStore(room.id, { 
        house_id: house.id, 
        name: values.name, 
        price: parseNumber(values.price), 
        max_tenants: Number(values.maxTenants),
        status: room.status,
        electricity_price: values.electricity ? parseNumber(values.electricity) : undefined,
        water_price: values.water ? parseNumber(values.water) : undefined,
        wifi_price: values.wifi ? parseNumber(values.wifi) : undefined,
        parking_price: values.parking ? parseNumber(values.parking) : undefined,
        service_price: values.service ? parseNumber(values.service) : undefined,
        extra_person_threshold: values.extraPersonThreshold ? Number(values.extraPersonThreshold) : undefined,
        extra_person_fee: values.extraPersonFee ? parseNumber(values.extraPersonFee) : undefined,
        extra_vehicle_threshold: values.extraVehicleThreshold ? Number(values.extraVehicleThreshold) : undefined,
        extra_vehicle_fee: values.extraVehicleFee ? parseNumber(values.extraVehicleFee) : undefined,
        group_chat_id: values.groupChatId || undefined,
      })
      if (!res.success) {
        alert(res.error || "Lỗi khi cập nhật phòng, vui lòng kiểm tra lại thông tin!")
        return
      }

      toast.success('Cập nhật thông tin phòng thành công!')
      onClose()
    } catch {
      alert("Lỗi khi cập nhật phòng, vui lòng kiểm tra lại thông tin!")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <>
    <AppModal open onClose={handleClose} title="Sửa thông tin phòng" contentClassName="sm:max-w-xl">
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2 col-span-2">
              <Label>Tên phòng</Label>
              <Input {...register('name')} className="border-border" />
              {errors.name && <span className="text-destructive text-xs">{errors.name.message}</span>}
            </div>
            <div className="space-y-2">
              <Label>Giá thuê hàng tháng (VNĐ)</Label>
              <Controller
                name="price"
                control={control}
                render={({ field }) => (
                  <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-border" />
                )}
              />
              {errors.price && <span className="text-destructive text-xs">{errors.price.message}</span>}
            </div>
            <div className="space-y-2">
              <Label>Sức chứa tối đa (người)</Label>
              <Input type="number" min="1" {...register('maxTenants')} className="border-border" />
              {errors.maxTenants && <span className="text-destructive text-xs">{errors.maxTenants.message}</span>}
            </div>
          </div>

          <div className="p-4 rounded-lg border border-border/50 bg-card">
             <h3 className="font-medium text-foreground text-sm mb-3">Đơn giá dịch vụ riêng cho phòng này</h3>
             <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Điện (VNĐ/kWh)</Label>
                  <Controller
                    name="electricity"
                    control={control}
                    render={({ field }) => (
                      <Input {...field} placeholder="Mặc định..." value={formatNumber(field.value || '')} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="h-8 border-border bg-background" />
                    )}
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Nước (VNĐ/m3 hoặc ng)</Label>
                  <Controller
                    name="water"
                    control={control}
                    render={({ field }) => (
                      <Input {...field} placeholder="Mặc định..." value={formatNumber(field.value || '')} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="h-8 border-border bg-background" />
                    )}
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Wifi (VNĐ/phòng hoặc ng)</Label>
                  <Controller
                    name="wifi"
                    control={control}
                    render={({ field }) => (
                      <Input {...field} placeholder="Mặc định..." value={formatNumber(field.value || '')} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="h-8 border-border bg-background" />
                    )}
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Giữ xe (VNĐ/xe)</Label>
                  <Controller
                    name="parking"
                    control={control}
                    render={({ field }) => (
                      <Input {...field} placeholder="Mặc định..." value={formatNumber(field.value || '')} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="h-8 border-border bg-background" />
                    )}
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Dịch vụ (VNĐ/phòng hoặc ng)</Label>
                  <Controller
                    name="service"
                    control={control}
                    render={({ field }) => (
                      <Input {...field} placeholder="Mặc định..." value={formatNumber(field.value || '')} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="h-8 border-border bg-background" />
                    )}
                  />
                </div>
             </div>
          </div>

          <div className="p-4 rounded-lg border border-border/50 bg-card">
             <h3 className="font-medium text-foreground text-sm mb-3">Quy định phụ thu phát sinh</h3>
             <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Số người miễn phí</Label>
                  <Input type="number" min="0" {...register('extraPersonThreshold')} placeholder="Mặc định..." className="h-8 border-border bg-background" />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Phí / người vượt (VNĐ)</Label>
                  <Controller
                    name="extraPersonFee"
                    control={control}
                    render={({ field }) => (
                      <Input {...field} placeholder="Mặc định..." value={formatNumber(field.value || '')} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="h-8 border-border bg-background" />
                    )}
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Số xe miễn phí</Label>
                  <Input type="number" min="0" {...register('extraVehicleThreshold')} placeholder="Mặc định..." className="h-8 border-border bg-background" />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Phí / xe vượt (VNĐ)</Label>
                  <Controller
                    name="extraVehicleFee"
                    control={control}
                    render={({ field }) => (
                      <Input {...field} placeholder="Mặc định..." value={formatNumber(field.value || '')} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="h-8 border-border bg-background" />
                    )}
                  />
                </div>
             </div>
          </div>


          <div className="p-4 rounded-lg border border-border/50 bg-card">
            <div className="flex items-center justify-between mb-1">
              <div>
                <h3 className="font-medium text-foreground text-sm">Hợp đồng thuê</h3>
                <p className="text-xs text-muted-foreground">Hợp đồng tự động lưu ngay khi tải lên hoặc xóa.</p>
              </div>
              {isUploadingContract && (
                <span className="flex items-center gap-1.5 text-xs text-primary font-medium">
                  <Loader2 className="w-3.5 h-3.5 animate-spin" /> Đang tải & lưu hợp đồng...
                </span>
              )}
            </div>

            {existingContractPaths.length === 0 && !isUploadingContract ? (
              <label className="flex flex-col gap-2 items-center justify-center h-24 mt-2 rounded-lg border-2 border-dashed border-border bg-muted/30 cursor-pointer hover:bg-secondary hover:border-primary/50 transition-colors">
                <Upload className="w-5 h-5 text-muted-foreground" />
                <span className="text-xs text-muted-foreground font-semibold px-4 text-center">Tải lên hợp đồng</span>
                <input
                  type="file"
                  accept=".pdf,image/*"
                  multiple
                  className="hidden"
                  onChange={e => {
                    void handleUploadContract(e.target.files)
                    e.target.value = ''
                  }}
                />
              </label>
            ) : (
              <div className="flex flex-col gap-2 max-h-[320px] overflow-y-auto pr-1 mt-2">
                {existingContractPaths.map((path, idx) => {
                  const isImg = isImagePath(path)
                  return (
                    <div key={`exist-contract-${idx}`} className="flex flex-col gap-2">
                      <div className="flex items-center justify-between h-10 px-3 rounded-lg border border-primary/20 bg-primary/10">
                        <div
                          className="flex items-center gap-2 overflow-hidden truncate cursor-pointer flex-1 min-w-0"
                          onClick={() => handleOpenGallery(contractGallery, idx)}
                          title="Nhấn để xem hợp đồng"
                        >
                          <CheckCircle2 className="w-3.5 h-3.5 text-success flex-shrink-0" />
                          <span className="text-xs font-semibold text-primary truncate">{getFileName(path)}</span>
                        </div>
                        <div className="flex items-center gap-1 shrink-0">
                          {isImg && (
                            <button
                              type="button"
                              onClick={() => handleOpenGallery(contractGallery, idx)}
                              className="p-1 hover:bg-primary/20 rounded-md text-primary cursor-pointer"
                              title="Xem trước"
                            >
                              <Eye className="w-4 h-4" />
                            </button>
                          )}
                          <button
                            type="button"
                            onClick={() => setContractToDelete({ path, index: idx })}
                            className="p-1 hover:bg-destructive/10 rounded-md text-muted-foreground hover:text-destructive cursor-pointer"
                            title="Xóa"
                          >
                            <X className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      </div>
                      {isImg ? (
                        <div className="relative rounded-lg overflow-hidden border border-border aspect-video bg-muted cursor-pointer group" onClick={() => handleOpenGallery(contractGallery, idx)}>
                          <ProtectedFileImage path={path} alt={getFileName(path)} className="w-full h-full object-cover" />
                          <div className="absolute inset-0 bg-black/20 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center text-white">
                            <ZoomIn className="w-5 h-5" />
                          </div>
                        </div>
                      ) : (
                        <div
                          className="rounded-lg border border-border/50 aspect-video bg-muted/30 flex flex-col items-center justify-center text-muted-foreground cursor-pointer hover:bg-muted/50 transition-colors"
                          onClick={() => handleOpenGallery(contractGallery, idx)}
                        >
                          <FileIcon className="w-8 h-8" />
                          <span className="text-xs font-medium mt-1">FILE TÀI LIỆU</span>
                        </div>
                      )}
                    </div>
                  )
                })}

                {isUploadingContract && (
                  <div className="flex items-center justify-center gap-2 p-4 rounded-lg border border-dashed border-primary/40 bg-primary/5 text-primary text-xs font-medium">
                    <Loader2 className="w-4 h-4 animate-spin" />
                    Đang tải lên & lưu hợp đồng...
                  </div>
                )}

                <label className="flex items-center justify-center gap-2 h-9 mt-1 rounded-lg border border-dashed border-border bg-muted/30 cursor-pointer hover:bg-secondary transition-colors">
                  <Upload className="w-4 h-4 text-muted-foreground" />
                  <span className="text-xs font-semibold text-foreground">Tải thêm file hợp đồng</span>
                  <input
                    type="file"
                    accept=".pdf,image/*"
                    multiple
                    disabled={isUploadingContract}
                    className="hidden"
                    onChange={e => {
                      void handleUploadContract(e.target.files)
                      e.target.value = ''
                    }}
                  />
                </label>
              </div>
            )}
          </div>

          <div className="bg-info/10 p-4 rounded-lg border border-info/30">
             <h3 className="font-medium text-info text-sm mb-1">Zalo Bot Auto-linking</h3>
             <p className="text-xs text-info/80 mb-3">Group Chat ID sẽ tự động cập nhật khi bot được thêm vào nhóm. Bạn có thể sửa thủ công nếu bị lỗi.</p>
             <div className="space-y-1">
               <Label className="text-xs text-info">Group Chat ID</Label>
               <Input {...register('groupChatId')} placeholder="Tự động cập nhật..." className="h-8 border-border bg-background" />
             </div>
          </div>

          <div className="flex justify-end gap-3 pt-4">
            <Button type="button" variant="outline" onClick={handleClose}>Hủy</Button>
            <Button type="submit" disabled={isLoading || isUploadingContract}>
              {isLoading ? 'Đang lưu...' : 'Lưu thay đổi'}
            </Button>
          </div>
      </form>
    </AppModal>

    {/* Confirmation Dialog for Deleting Contract File */}
    {contractToDelete && (
      <ConfirmModal
        title="Xác nhận xóa hợp đồng"
        message={`Bạn có chắc chắn muốn xóa file "${getFileName(contractToDelete.path)}"? Thay đổi sẽ được lưu ngay lập tức.`}
        confirmText="Xóa file"
        cancelText="Hủy"
        isLoading={isDeletingContract}
        onConfirm={handleConfirmDeleteContract}
        onCancel={() => {
          if (!isDeletingContract) setContractToDelete(null)
        }}
      />
    )}

    {/* Lightbox / Gallery Modal */}
    <ImageLightboxModal
      isOpen={Boolean(gallery)}
      images={gallery?.items}
      initialIndex={gallery?.initialIndex ?? 0}
      onClose={() => setGallery(null)}
    />
    {confirmModal}
    </>
  )
}
