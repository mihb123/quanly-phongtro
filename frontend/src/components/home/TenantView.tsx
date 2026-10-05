import { memo, useCallback, useEffect, useMemo, useState } from 'react'
import { AlertTriangle, Building, DoorOpen, FilterX, Plus, Shield, Users } from '@/components/icons'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { PageHeader } from '@/components/shared/PageHeader'
import { EmptyState } from '@/components/shared/EmptyState'
import { StatCard } from '@/components/shared/StatCard'
import { SearchInput } from '@/components/shared/SearchInput'
import { DataPagination } from '@/components/shared/DataPagination'
import { VerifiedBadge } from '@/components/shared/VerifiedBadge'
import { useHouseStore } from '@/data/houseData'
import { useRoomStore } from '@/data/roomData'
import { refreshTenantDependents } from '@/data/tenantData'
import { TenantRoomModal } from '@/components/home/modals/TenantRoomModal'
import { SelectRoomModal } from '@/components/home/modals/SelectRoomModal'
import { type Room } from '@/api/room'
import { searchTenants, type Tenant } from '@/api/tenant'
import { useQuery, queryKey } from '@/lib/queryCache'
import { useStableCallback } from '@/hooks/useStableCallback'
import { FILTER_TTL, readPageSize, readStorage, removeStorage, writeStorage } from '@/lib/storage'
import { cn } from '@/lib/utils'
import { toast } from 'sonner'
import { StayingBadge, TenantMobileCard } from './tenant/TenantMobileCard'

const FILTERS_KEY = 'tenants:filters'
const PAGE_SIZE_KEY = 'tenants:pageSize'

interface TenantFilters {
  house_id: string
  q: string
}

const DEFAULT_FILTERS: TenantFilters = { house_id: '', q: '' }
const EMPTY_TENANTS: Tenant[] = []

function restoreFilters(): TenantFilters {
  const saved = readStorage<Partial<TenantFilters>>(FILTERS_KEY, { ttl: FILTER_TTL }) || {}
  return {
    house_id: typeof saved.house_id === 'string' ? saved.house_id : '',
    q: typeof saved.q === 'string' ? saved.q : '',
  }
}

const formatCount = (value?: number) => (value === undefined ? '—' : value.toLocaleString('vi-VN'))

// View khách thuê: một danh sách chung, lọc theo nhà + tìm kiếm, phân trang ở backend.
export function TenantsView() {
  const { houses } = useHouseStore()
  const getRoomsByHouse = useRoomStore(state => state.getRoomsByHouse)

  const [filters, setFilters] = useState<TenantFilters>(restoreFilters)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(() => readPageSize(PAGE_SIZE_KEY))

  const houseId = houses.some(h => h.id === filters.house_id) ? filters.house_id : ''
  const params = { house_id: houseId || undefined, q: filters.q || undefined, page, limit: pageSize }
  const query = useQuery(queryKey('tenants:list', params), () => searchTenants(params))
  const tenants = query.data?.items ?? EMPTY_TENANTS
  const summary = query.data?.summary
  const total = query.data?.total ?? 0
  const activeFilterCount = [houseId, filters.q].filter(Boolean).length
  const showHouse = !houseId && houses.length > 1

  useEffect(() => {
    if (!query.data || query.isPlaceholder || query.error !== undefined) return
    const lastPage = Math.max(1, Math.ceil(query.data.total / pageSize))
    if (page > lastPage) setPage(lastPage)
  }, [query.data, query.isPlaceholder, query.error, page, pageSize])

  const updateFilters = (patch: Partial<TenantFilters>) => {
    setFilters(prev => {
      const next = { ...prev, ...patch }
      writeStorage(FILTERS_KEY, next)
      return next
    })
    setPage(1)
  }
  const handleSearch = useCallback((q: string) => {
    setFilters(prev => {
      const next = { ...prev, q }
      writeStorage(FILTERS_KEY, next)
      return next
    })
    setPage(1)
  }, [])
  const clearFilters = () => {
    removeStorage(FILTERS_KEY)
    setFilters(DEFAULT_FILTERS)
    setPage(1)
  }
  const changePageSize = (size: number) => {
    writeStorage(PAGE_SIZE_KEY, size)
    setPageSize(size)
    setPage(1)
  }

  const [addHouseId, setAddHouseId] = useState<string | null>(null)
  const [selectedRoom, setSelectedRoom] = useState<Room | null>(null)
  const [modalView, setModalView] = useState<'list' | 'add' | 'edit'>('list')
  const [editingTenant, setEditingTenant] = useState<Tenant | null>(null)
  const [isFetchingRoom, setIsFetchingRoom] = useState(false)

  const openRoomFor = async (tenant: Tenant, view: 'list' | 'edit') => {
    const tenantHouseId = tenant.house_id || houseId
    if (!tenant.room_id || !tenantHouseId) return
    setIsFetchingRoom(true)
    try {
      const rooms = await getRoomsByHouse(tenantHouseId)
      const room = rooms.find(r => r.id === tenant.room_id)
      if (room) {
        setSelectedRoom(room)
        setModalView(view)
        setEditingTenant(view === 'edit' ? tenant : null)
      }
    } finally {
      setIsFetchingRoom(false)
    }
  }

  const handleTenantClick = useStableCallback((tenant: Tenant) => openRoomFor(tenant, 'edit'))
  const handleRoomClick = useStableCallback((tenant: Tenant) => openRoomFor(tenant, 'list'))
  const handlePhoneClick = useCallback((tenant: Tenant) => {
    const phone = tenant.phone?.trim()
    if (!phone) return
    navigator.clipboard.writeText(phone).then(() => {
      toast.success('Sao chép thành công', { description: phone })
    })
  }, [])

  const addButton = useMemo(() => {
    if (houses.length === 0) return null
    if (houseId || houses.length === 1) {
      const target = houseId || houses[0].id
      return (
        <Button size="lg" onClick={() => setAddHouseId(target)}>
          <Plus data-icon="inline-start" /> Thêm khách thuê
        </Button>
      )
    }
    return (
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button size="lg">
              <Plus data-icon="inline-start" /> Thêm khách thuê
            </Button>
          }
        />
        <DropdownMenuContent side="bottom" align="end">
          <DropdownMenuGroup>
            {houses.map(h => (
              <DropdownMenuItem key={h.id} onClick={() => setAddHouseId(h.id)}>
                <Building /> {h.name}
              </DropdownMenuItem>
            ))}
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    )
  }, [houses, houseId])

  if (houses.length === 0) {
    return (
      <div className="flex flex-col gap-6 safe-fade-in">
        <PageHeader title="Danh sách Khách thuê" description="Thông tin khách thuê đang ở." />
        <Card className="p-8">
          <EmptyState icon={Building} title="Bạn chưa có nhà trọ nào để quản lý khách thuê." />
        </Card>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-4 safe-fade-in sm:gap-6">
      <PageHeader
        title="Danh sách Khách thuê"
        description={houseId ? `Khách đang ở tại ${houses.find(h => h.id === houseId)?.name}.` : 'Khách đang ở tại tất cả nhà trọ.'}
        action={addButton}
      />

      <div className="grid grid-cols-[repeat(auto-fit,minmax(140px,1fr))] gap-3">
        <StatCard label="Khách đang ở" value={formatCount(summary?.tenants)} icon={Users} tone="info" />
        <StatCard label="Phòng có khách" value={formatCount(summary?.rooms)} icon={DoorOpen} tone="positive" />
        <StatCard
          label="Đã có CCCD"
          value={formatCount(summary?.verified)}
          icon={Shield}
          sub={summary && summary.tenants > 0 ? `${Math.round((summary.verified / summary.tenants) * 100)}% khách` : undefined}
        />
        <StatCard label="Nhà trọ" value={formatCount(summary?.houses)} icon={Building} />
      </div>

      <Card className="relative gap-3 p-3 sm:gap-4 sm:p-4">
        {isFetchingRoom && (
          <div className="absolute inset-0 z-10 flex items-center justify-center rounded-xl bg-background/50 backdrop-blur-sm">
            <div className="font-medium text-muted-foreground">Đang tải thông tin phòng...</div>
          </div>
        )}

        <div className="grid grid-cols-2 gap-2 sm:flex sm:flex-wrap sm:items-center sm:gap-3">
          <SearchInput
            label="Tìm khách thuê"
            placeholder="Tên, SĐT, CCCD, phòng..."
            value={filters.q}
            onSearch={handleSearch}
            className="col-span-2 sm:w-72"
          />
          <label htmlFor="tenant-filter-house" className="sr-only">Lọc theo nhà trọ</label>
          <select
            id="tenant-filter-house"
            value={houseId}
            onChange={e => updateFilters({ house_id: e.target.value })}
            className="h-11 w-full cursor-pointer rounded-md border border-input bg-background px-3 text-base outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30 sm:w-56 md:h-9 md:text-sm"
          >
            <option value="">Tất cả nhà trọ</option>
            {houses.map(h => <option key={h.id} value={h.id}>{h.name}</option>)}
          </select>
          <Button variant="ghost" onClick={clearFilters} disabled={activeFilterCount === 0} className="h-11 text-muted-foreground md:h-9">
            <FilterX data-icon="inline-start" />
            Xóa lọc{activeFilterCount > 0 ? ` (${activeFilterCount})` : ''}
          </Button>
        </div>

        {query.error !== undefined && (
          <div role="alert" className="flex items-center justify-between gap-3 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
            <span className="flex items-center gap-2"><AlertTriangle className="size-4 shrink-0" /> Không tải được danh sách khách thuê.</span>
            <Button variant="outline" size="sm" onClick={() => query.refetch()}>Thử lại</Button>
          </div>
        )}

        {query.isLoading ? (
          <div className="flex flex-col gap-2" aria-busy="true">
            {Array.from({ length: 5 }, (_, i) => <Skeleton key={i} className="h-16 rounded-lg" />)}
          </div>
        ) : tenants.length === 0 ? (
          <EmptyState
            icon={Users}
            title={activeFilterCount > 0 ? 'Không có khách thuê nào khớp bộ lọc.' : 'Chưa có khách thuê nào.'}
          />
        ) : (
          <div className={cn('flex flex-col gap-3 transition-opacity', query.isPlaceholder && 'opacity-60')}>
            <ul className="flex flex-col gap-2 md:hidden">
              {tenants.map(tenant => (
                <TenantMobileCard
                  key={tenant.id}
                  tenant={tenant}
                  showHouse={showHouse}
                  onEdit={handleTenantClick}
                  onOpenRoom={handleRoomClick}
                  onCopyPhone={handlePhoneClick}
                />
              ))}
            </ul>

            <div className="hidden overflow-x-auto rounded-lg border md:block">
              <Table className="whitespace-nowrap">
                <TableHeader>
                  <TableRow className="bg-secondary/30">
                    <TableHead>Tên khách thuê</TableHead>
                    {showHouse && <TableHead>Nhà trọ</TableHead>}
                    <TableHead>Phòng đang ở</TableHead>
                    <TableHead>Số điện thoại</TableHead>
                    <TableHead>Ngày bắt đầu ở</TableHead>
                    <TableHead>Tình trạng</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {tenants.map(t => (
                    <TenantRow
                      key={t.id}
                      tenant={t}
                      showHouse={showHouse}
                      onEdit={handleTenantClick}
                      onOpenRoom={handleRoomClick}
                      onCopyPhone={handlePhoneClick}
                    />
                  ))}
                </TableBody>
              </Table>
            </div>

            <DataPagination
              page={page}
              pageSize={pageSize}
              total={total}
              onPageChange={setPage}
              onPageSizeChange={changePageSize}
            />
          </div>
        )}
      </Card>

      {addHouseId && (
        <SelectRoomModal
          houseId={addHouseId}
          onSelect={(r) => {
            setSelectedRoom(r)
            setModalView('add')
            setEditingTenant(null)
            setAddHouseId(null)
          }}
          onClose={() => setAddHouseId(null)}
        />
      )}

      {selectedRoom && (
        <TenantRoomModal
          room={selectedRoom}
          initialView={modalView}
          initialEditingTenant={editingTenant}
          onClose={(changed?: boolean) => {
            setSelectedRoom(null)
            setEditingTenant(null)
            if (changed) refreshTenantDependents()
          }}
        />
      )}
    </div>
  )
}

interface TenantRowProps {
  tenant: Tenant
  showHouse: boolean
  onEdit: (tenant: Tenant) => void
  onOpenRoom: (tenant: Tenant) => void
  onCopyPhone: (tenant: Tenant) => void
}

const TenantRow = memo(function TenantRow({ tenant: t, showHouse, onEdit, onOpenRoom, onCopyPhone }: TenantRowProps) {
  const phone = t.phone?.trim()
  return (
    <TableRow>
      <TableCell className="font-semibold text-foreground">
        <div className="flex items-center gap-1.5">
          <button type="button" onClick={() => onEdit(t)} className="cursor-pointer underline-offset-4 hover:underline" title="Sửa người thuê">
            {t.full_name}
          </button>
          {t.cccd_path && <VerifiedBadge title="Đã tải ảnh CCCD" />}
        </div>
      </TableCell>
      {showHouse && <TableCell className="text-muted-foreground">{t.house_name}</TableCell>}
      <TableCell className="font-medium text-foreground">
        <button type="button" onClick={() => onOpenRoom(t)} className="cursor-pointer underline-offset-4 hover:underline" title="Xem danh sách người thuê phòng">
          {t.room_name || 'N/A'}
        </button>
      </TableCell>
      <TableCell className="text-muted-foreground">
        {phone ? (
          <button type="button" onClick={() => onCopyPhone(t)} className="cursor-pointer tabular-nums underline-offset-4 hover:underline" title="Sao chép số điện thoại">
            {phone}
          </button>
        ) : (
          <span className="italic">Chưa cập nhật</span>
        )}
      </TableCell>
      <TableCell className="text-muted-foreground">{new Date(t.start_date).toLocaleDateString('vi-VN')}</TableCell>
      <TableCell><StayingBadge /></TableCell>
    </TableRow>
  )
})
