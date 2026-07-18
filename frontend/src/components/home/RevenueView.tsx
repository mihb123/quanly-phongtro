import { useState, useEffect, useMemo, useCallback, useRef } from 'react';
import { useHouseCostStore } from '@/data/houseCostData';
import { useHouseStore } from '@/data/houseData';
import { useSelectedStore, type TabType } from '@/data/selectedData';
import { useDirtyConfirm } from '@/hooks/useDirtyConfirm';
import { useHouseCostEditor } from '@/hooks/useHouseCostEditor';
import type { HouseCost } from '@/api/houseCost';
import { Wallet, TrendingUp, TrendingDown, DollarSign, Plus, Save, Trash2, Calendar, AlertCircle } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Table, TableHeader, TableBody, TableFooter, TableRow, TableHead, TableCell } from '@/components/ui/table';
import { Bar, BarChart, CartesianGrid, XAxis } from 'recharts';
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from '@/components/ui/chart';
import { useRevenueTrend } from '@/hooks/useRevenueTrend';
import { PageHeader } from '@/components/shared/PageHeader';
import { StatCard } from '@/components/shared/StatCard';
import { SectionCard } from '@/components/shared/SectionCard';
import { EmptyState } from '@/components/shared/EmptyState';
import { formatCurrency } from '@/utils/format';

import { HouseSelectDropdown } from './HouseSelectDropdown';
import { CurrencyInput } from '@/components/ui/currency-input';
import { EditNoteModal } from './modals/EditNoteModal';

// Cấu hình series cho chart xu hướng — màu lấy từ token chart trong index.css.
const TREND_CHART_CONFIG = {
  revenue: { label: 'Doanh thu', color: 'var(--chart-2)' },
  cost: { label: 'Chi phí', color: 'var(--chart-1)' },
} satisfies ChartConfig

// Đổi 'yyyy-mm' thành nhãn 'T<m>' gọn cho trục hoành của chart.
function formatPeriodLabel(period: string): string {
  const month = Number(period.split('-')[1])
  return Number.isNaN(month) ? period : `T${month}`
}

// View doanh thu: lọc theo kỳ + nhà trọ, hiển thị thống kê tổng hợp, xu hướng và bảng chi tiết chi phí vận hành.
export function RevenueView() {
  const { houses, fetchHouses } = useHouseStore();
  const {
    costs,
    summaries,
    isLoadingCosts,
    isLoadingSummaries,
    selectedHouseIds,
    period,
    setSelectedHouseIds,
    setPeriod,
    fetchCosts,
    fetchSummaries
  } = useHouseCostStore();
  const setActiveTab = useSelectedStore(s => s.setActiveTab);
  const setTabChangeInterceptor = useSelectedStore(s => s.setTabChangeInterceptor);

  const {
    isSaving,
    hasAnyEdits,
    clearEdits,
    hasEdits,
    handleCreateMonthCost,
    handleFieldChange,
    handleExtraCostChange,
    handleAddExtraCost,
    handleRemoveExtraCost,
    handleSaveCost,
    getActiveCostValue,
    getActiveExtraCosts,
  } = useHouseCostEditor();

  const [isHouseSelectOpen, setIsHouseSelectOpen] = useState(false);
  const [pendingAction, setPendingAction] = useState<(() => void) | null>(null);
  const [editingNoteFor, setEditingNoteFor] = useState<{houseId: string, index: number} | null>(null);

  const handleDiscard = useCallback(() => {
    clearEdits();
    if (pendingAction) {
      pendingAction();
      setPendingAction(null);
    }
  }, [pendingAction, clearEdits]);

  const { handleClose: triggerConfirm, confirmModal } = useDirtyConfirm(
    hasAnyEdits,
    handleDiscard
  );

  const interceptorRef = useRef<((nextTab: TabType) => boolean)>(() => true);

  useEffect(() => {
    interceptorRef.current = (nextTab: TabType) => {
      if (hasAnyEdits) {
        setPendingAction(() => () => {
          setTabChangeInterceptor(null);
          setActiveTab(nextTab);
        });
        triggerConfirm();
        return false;
      }
      return true;
    };
  }, [hasAnyEdits, setActiveTab, setTabChangeInterceptor, triggerConfirm]);

  useEffect(() => {
    const handler = (nextTab: TabType) => interceptorRef.current(nextTab);
    setTabChangeInterceptor(handler);
    return () => {
      setTabChangeInterceptor(null);
    };
  }, [setTabChangeInterceptor]);

  useEffect(() => {
    fetchHouses();
  }, [fetchHouses]);

  // Fetch initial data if there are persisted selected houses
  useEffect(() => {
    if (selectedHouseIds.length > 0) {
      fetchCosts();
      fetchSummaries();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (houses.length === 0) return;

    const availableHouseIds = new Set(houses.map(house => house.id));
    const availableSelectedHouseIds = selectedHouseIds.filter(id => availableHouseIds.has(id));

    if (availableSelectedHouseIds.length === 0) {
      // Mặc định chọn 1 house mới nhất (house đầu tiên)
      setSelectedHouseIds([houses[0].id]);
      return;
    }

    if (availableSelectedHouseIds.length !== selectedHouseIds.length) {
      setSelectedHouseIds(availableSelectedHouseIds);
    }
  }, [houses, selectedHouseIds, setSelectedHouseIds]);

  const { points: trendPoints, isLoading: isLoadingTrend } = useRevenueTrend(selectedHouseIds, period)

  const aggregatedSummary = useMemo(() => {
    let totalRev = 0;
    let totalCost = 0;
    (summaries || []).forEach(s => {
      totalRev += s.total_revenue;
      totalCost += s.total_cost;
    });
    return {
      totalRevenue: totalRev,
      totalCost: totalCost,
      profit: totalRev - totalCost
    };
  }, [summaries]);

  const handleHouseToggleRequest = (houseId: string) => {
    if (hasAnyEdits) {
      setPendingAction(() => () => handleHouseToggle(houseId));
      triggerConfirm();
    } else {
      handleHouseToggle(houseId);
    }
  };

  const handleHouseSelectChangeRequest = (houseIds: string[]) => {
    if (hasAnyEdits) {
      setPendingAction(() => () => setSelectedHouseIds(houseIds));
      triggerConfirm();
    } else {
      setSelectedHouseIds(houseIds);
    }
  };

  const handleSelectAllRequest = () => {
    const action = () => {
      if (selectedHouseIds.length === houses.length) {
        setSelectedHouseIds([]);
      } else {
        setSelectedHouseIds(houses.map(h => h.id));
      }
    };
    if (hasAnyEdits) {
      setPendingAction(() => action);
      triggerConfirm();
    } else {
      action();
    }
  };

  const handlePeriodChange = (newPeriod: string) => {
    if (hasAnyEdits) {
      setPendingAction(() => () => setPeriod(newPeriod));
      triggerConfirm();
    } else {
      setPeriod(newPeriod);
    }
  };

  const handleHouseToggle = (houseId: string) => {
    if (selectedHouseIds.includes(houseId)) {
      setSelectedHouseIds(selectedHouseIds.filter(id => id !== houseId));
    } else {
      setSelectedHouseIds([...selectedHouseIds, houseId]);
    }
  };

  return (
    <div className="space-y-8 safe-fade-in pb-12">
      {/* Header & Filters */}
      <PageHeader
        className="border-b border-border/40 pb-6"
        title="Doanh thu"
        description="Theo dõi dòng tiền, chi phí vận hành và lợi nhuận ròng."
      />

      {/* Summary Cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <StatCard
          label="Tổng Doanh Thu"
          value={isLoadingSummaries ? '...' : formatCurrency(aggregatedSummary.totalRevenue)}
          icon={TrendingUp}
          tone="info"
        />
        <StatCard
          label="Tổng Chi Phí"
          value={isLoadingSummaries || isLoadingCosts ? '...' : formatCurrency(aggregatedSummary.totalCost)}
          icon={TrendingDown}
          tone="warning"
        />
        <StatCard
          label="Lợi Nhuận Ròng"
          value={isLoadingSummaries ? '...' : formatCurrency(aggregatedSummary.profit)}
          icon={DollarSign}
          tone={aggregatedSummary.profit >= 0 ? 'positive' : 'negative'}
        />
      </div>

      {/* Trend Chart: 6 kỳ gần nhất của các nhà đang chọn */}
      {selectedHouseIds.length > 0 && (
        <SectionCard
          icon={TrendingUp}
          title="Xu hướng 6 kỳ gần nhất"
          bodyClassName="p-4 sm:p-6"
        >
          {isLoadingTrend && trendPoints.length === 0 ? (
            <div className="flex h-56 items-center justify-center text-sm text-muted-foreground">Đang tải dữ liệu xu hướng...</div>
          ) : (
            <ChartContainer config={TREND_CHART_CONFIG} className="h-56 w-full">
              <BarChart accessibilityLayer data={trendPoints}>
                <CartesianGrid vertical={false} />
                <XAxis
                  dataKey="period"
                  tickLine={false}
                  tickMargin={8}
                  axisLine={false}
                  tickFormatter={formatPeriodLabel}
                />
                <ChartTooltip
                  cursor={false}
                  content={<ChartTooltipContent formatter={(value, name) => (
                    <div className="flex w-full items-center justify-between gap-4">
                      <span className="text-muted-foreground">{TREND_CHART_CONFIG[name as keyof typeof TREND_CHART_CONFIG]?.label ?? name}</span>
                      <span className="font-medium tabular-nums">{formatCurrency(Number(value))}</span>
                    </div>
                  )} />}
                />
                <Bar dataKey="revenue" fill="var(--color-revenue)" radius={4} />
                <Bar dataKey="cost" fill="var(--color-cost)" radius={4} />
              </BarChart>
            </ChartContainer>
          )}
        </SectionCard>
      )}

      {/* Detail Costs Section */}
      <SectionCard
        title={
          selectedHouseIds.length === 1 
            ? `Chi tiết chi phí vận hành - ${houses.find(h => h.id === selectedHouseIds[0])?.name ?? ''}`
            : 'Chi tiết chi phí vận hành'
        }
        action={
          <div className="flex w-full flex-wrap items-center justify-end gap-2 sm:w-auto md:gap-3">
            <div className="min-w-0 flex-1 sm:flex-none">
              <HouseSelectDropdown
                houses={houses}
                selectedHouseIds={selectedHouseIds}
                isOpen={isHouseSelectOpen}
                onOpenChange={setIsHouseSelectOpen}
                onToggleHouse={handleHouseToggleRequest}
                onSelectAll={handleSelectAllRequest}
                onSelectChange={handleHouseSelectChangeRequest}
              />
            </div>
            <div className="flex h-9 min-w-0 flex-1 items-center gap-2 overflow-hidden rounded-md border border-input bg-background px-2 text-foreground shadow-xs transition-colors focus-within:border-ring focus-within:ring-2 focus-within:ring-ring/30 sm:flex-none md:px-3">
              <Calendar className="hidden size-4 shrink-0 text-muted-foreground sm:block" />
              <Input
                type="month"
                value={period}
                onChange={(e) => handlePeriodChange(e.target.value)}
                className="h-full w-full min-w-0 border-none bg-transparent p-0 text-center text-sm font-medium shadow-none focus-visible:ring-0 sm:text-left md:w-[120px]"
              />
            </div>
            {selectedHouseIds.length === 1 && costs[selectedHouseIds[0]] && hasEdits(selectedHouseIds[0]) && (
              <Button
                onClick={() => handleSaveCost(selectedHouseIds[0])}
                disabled={isSaving[selectedHouseIds[0]]}
                
              >
                {isSaving[selectedHouseIds[0]] ? 'Đang lưu...' : <><Save className="size-4" /> Lưu thay đổi</>}
              </Button>
            )}
          </div>
        }
        bodyClassName="p-6"
      >
        {selectedHouseIds.length === 1 ? (
          isLoadingCosts ? (
            <div className="py-12 text-center text-muted-foreground">Đang tải dữ liệu chi phí...</div>
          ) : !costs[selectedHouseIds[0]] ? (
            <EmptyState
              icon={AlertCircle}
              className="rounded-lg border border-dashed py-16"
              title={`Chưa có chi phí cho kỳ ${period}`}
              description="Tạo bản ghi chi phí vận hành cho tháng này. Hệ thống sẽ tự động sao chép các chi phí cố định từ tháng trước nếu có."
              action={
                <Button onClick={() => handleCreateMonthCost(selectedHouseIds[0])}>
                  <Plus data-icon="inline-start" /> Tạo chi phí tháng mới
                </Button>
              }
            />
          ) : (
            <div className="overflow-hidden rounded-lg border">
              <Table>
                <TableHeader className="bg-muted/50">
                  <TableRow>
                    <TableHead className="w-[25%] min-w-[100px] text-muted-foreground">Loại chi phí</TableHead>
                    <TableHead className="w-[15%] min-w-[100px] text-muted-foreground">Phân loại</TableHead>
                    <TableHead className="w-[20%] min-w-[120px] text-muted-foreground">Số tiền (VND)</TableHead>
                    <TableHead className="w-[25%] min-w-[200px] text-muted-foreground">Ghi chú</TableHead>
                    <TableHead className="w-[5%] min-w-[60px] text-right"></TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {/* Fixed Layout Costs */}
                  {[
                    { key: 'rent', label: 'Tiền thuê nhà', type: 'Cố định' },
                    { key: 'electricity', label: 'Tiền điện', type: 'Biến đổi' },
                    { key: 'water', label: 'Tiền nước', type: 'Biến đổi' },
                    { key: 'wifi', label: 'Tiền Internet', type: 'Cố định' },
                    { key: 'cleaning', label: 'Tiền vệ sinh, rác', type: 'Cố định' },
                  ].map((item) => (
                    <TableRow key={item.key} className="group">
                      <TableCell className="font-semibold text-foreground">{item.label}</TableCell>
                      <TableCell>
                        <Badge variant={item.type === 'Cố định' ? 'secondary' : 'outline'}>
                          {item.type}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <CurrencyInput
                          value={getActiveCostValue(selectedHouseIds[0], item.key as keyof HouseCost) as number}
                          onChange={(val) => handleFieldChange(selectedHouseIds[0], item.key as keyof HouseCost, val)}
                          className="h-8 w-40 tabular-nums"
                        />
                      </TableCell>
                      <TableCell></TableCell>
                      <TableCell></TableCell>
                    </TableRow>
                  ))}

                  {/* Extra Costs */}
                  {getActiveExtraCosts(selectedHouseIds[0]).map((extra, index) => (
                    <TableRow key={`extra-${index}`} className="group">
                      <TableCell>
                        <Input
                          value={extra.name}
                          onChange={(e) => handleExtraCostChange(selectedHouseIds[0], index, 'name', e.target.value)}
                          placeholder="Tên chi phí..."
                          className="h-8 w-full min-w-[150px]"
                        />
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline">Tùy chỉnh</Badge>
                      </TableCell>
                      <TableCell>
                        <CurrencyInput
                          value={extra.amount}
                          onChange={(val) => handleExtraCostChange(selectedHouseIds[0], index, 'amount', val)}
                          className="h-8 w-40 tabular-nums"
                        />
                      </TableCell>
                      <TableCell>
                        <div
                          onClick={() => setEditingNoteFor({ houseId: selectedHouseIds[0], index })}
                          className={`min-h-8 w-full min-w-[250px] cursor-pointer whitespace-pre-wrap rounded-md p-2 text-sm leading-relaxed transition-colors hover:bg-secondary/50 ${extra.note ? 'text-foreground' : 'italic text-muted-foreground'}`}
                        >
                          {extra.note || 'Bấm để thêm ghi chú...'}
                        </div>
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleRemoveExtraCost(selectedHouseIds[0], index)}
                          className="text-muted-foreground hover:text-destructive"
                        >
                          <Trash2 className="size-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                  <TableRow className="hover:bg-transparent">
                    <TableCell colSpan={5}>
                      <Button
                        variant="ghost"
                        onClick={() => handleAddExtraCost(selectedHouseIds[0])}
                        className="-ml-2 text-muted-foreground"
                      >
                        <Plus data-icon="inline-start" /> Thêm chi phí khác
                      </Button>
                    </TableCell>
                  </TableRow>
                </TableBody>
                <TableFooter className="bg-muted/50">
                  <TableRow className="hover:bg-transparent">
                    <TableCell colSpan={2} className="py-4 text-right font-medium text-muted-foreground">Tổng cộng chi phí:</TableCell>
                    <TableCell colSpan={3} className="py-4 text-lg font-semibold tabular-nums text-foreground">
                      {formatCurrency(
                        (getActiveCostValue(selectedHouseIds[0], 'rent') as number) +
                        (getActiveCostValue(selectedHouseIds[0], 'electricity') as number) +
                        (getActiveCostValue(selectedHouseIds[0], 'water') as number) +
                        (getActiveCostValue(selectedHouseIds[0], 'wifi') as number) +
                        (getActiveCostValue(selectedHouseIds[0], 'cleaning') as number) +
                        getActiveExtraCosts(selectedHouseIds[0]).reduce((acc, curr) => acc + curr.amount, 0)
                      )}
                    </TableCell>
                  </TableRow>
                </TableFooter>
              </Table>
            </div>
          )
        ) : selectedHouseIds.length > 1 ? (
          <EmptyState
            icon={Wallet}
            className="rounded-lg border border-dashed py-8"
            title="Đang xem tổng hợp nhiều nhà"
            description="Bảng chi tiết chỉ hiển thị khi bạn chọn 1 nhà duy nhất. Hãy bỏ chọn các nhà khác để xem và chỉnh sửa chi tiết."
          />
        ) : houses.length > 0 ? (
          <EmptyState
            className="rounded-lg border border-dashed py-8"
            title="Vui lòng chọn nhà trọ"
            description="Bạn cần chọn ít nhất 1 nhà trọ để xem thống kê doanh thu."
          />
        ) : null}
      </SectionCard>

      {confirmModal}

      {/* Note Editing Modal */}
      {editingNoteFor && (
        <EditNoteModal
          initialNote={getActiveExtraCosts(editingNoteFor.houseId)[editingNoteFor.index]?.note || ''}
          onSave={(note) => {
            handleExtraCostChange(editingNoteFor.houseId, editingNoteFor.index, 'note', note);
            setEditingNoteFor(null);
          }}
          onClose={() => setEditingNoteFor(null)}
        />
      )}
    </div>
  );
}
