import { useState, useEffect, useMemo, useCallback, useRef } from 'react';
import { useHouseCostStore } from '@/data/houseCostData';
import { useHouseStore } from '@/data/houseData';
import { useSelectedStore, type TabType } from '@/data/selectedData';
import { useDirtyConfirm } from '@/hooks/useDirtyConfirm';
import type { HouseCost, ExtraCost } from '@/api/houseCost';
import { Wallet, TrendingUp, TrendingDown, DollarSign, Plus, Save, Trash2, Calendar, AlertCircle } from '@/components/icons';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Table, TableHeader, TableBody, TableFooter, TableRow, TableHead, TableCell } from '@/components/ui/table';
import { PageHeader } from '@/components/shared/PageHeader';
import { StatCard } from '@/components/shared/StatCard';
import { SectionCard } from '@/components/shared/SectionCard';
import { EmptyState } from '@/components/shared/EmptyState';
import { formatCurrency } from '@/utils/format';

import { HouseSelectDropdown } from './HouseSelectDropdown';
import { CurrencyInput } from '@/components/ui/currency-input';
import { EditNoteModal } from './modals/EditNoteModal';

// View doanh thu: lọc theo kỳ + nhà trọ, hiển thị thống kê tổng hợp và bảng chi tiết chi phí vận hành.
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
    createCostForHouse,
    updateCost,
    fetchCosts,
    fetchSummaries
  } = useHouseCostStore();
  const setActiveTab = useSelectedStore(s => s.setActiveTab);
  const setTabChangeInterceptor = useSelectedStore(s => s.setTabChangeInterceptor);

  const [editState, setEditState] = useState<Record<string, Partial<HouseCost>>>({});
  const [isSaving, setIsSaving] = useState<Record<string, boolean>>({});
  const [isHouseSelectOpen, setIsHouseSelectOpen] = useState(false);
  const [pendingAction, setPendingAction] = useState<(() => void) | null>(null);
  const [editingNoteFor, setEditingNoteFor] = useState<{houseId: string, index: number} | null>(null);

  const hasAnyEdits = Object.keys(editState).length > 0;

  const handleDiscard = useCallback(() => {
    setEditState({});
    if (pendingAction) {
      pendingAction();
      setPendingAction(null);
    }
  }, [pendingAction]);

  const { handleClose: triggerConfirm, confirmModal } = useDirtyConfirm(
    hasAnyEdits,
    handleDiscard
  );

  const interceptorRef = useRef<((nextTab: TabType) => boolean)>(() => true);

  useEffect(() => {
    interceptorRef.current = (nextTab: TabType) => {
      if (Object.keys(editState).length > 0) {
        setPendingAction(() => () => {
          setTabChangeInterceptor(null);
          setActiveTab(nextTab);
        });
        triggerConfirm();
        return false;
      }
      return true;
    };
  }, [editState, setActiveTab, setTabChangeInterceptor, triggerConfirm]);

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

  const handleCreateMonthCost = async (houseId: string) => {
    try {
      const res = await createCostForHouse(houseId);
      if (res.success) {
        toast.success('Tạo chi phí tháng mới thành công');
      } else {
        toast.error(res.error || 'Không thể tạo chi phí hoặc đã tồn tại chi phí cho kỳ này.');
      }
    } catch {
      toast.error('Không thể tạo chi phí hoặc đã tồn tại chi phí cho kỳ này.');
    }
  };

  const handleFieldChange = (houseId: string, field: keyof HouseCost, value: string | number) => {
    const numValue = typeof value === 'string' ? (parseFloat(value) || 0) : value;
    setEditState(prev => ({
      ...prev,
      [houseId]: {
        ...prev[houseId],
        [field]: typeof value === 'string' && field === 'note' ? value : numValue
      }
    }));
  };

  const handleExtraCostChange = (houseId: string, index: number, field: keyof ExtraCost, value: string | number) => {
    const cost = costs[houseId];
    if (!cost) return;
    
    const currentEdits = editState[houseId]?.extra_costs || cost.extra_costs || [];
    const newExtras = [...currentEdits];
    
    if (field === 'amount') {
      newExtras[index] = { ...newExtras[index], amount: typeof value === 'string' ? (parseFloat(value) || 0) : value };
    } else if (field === 'name') {
      newExtras[index] = { ...newExtras[index], name: value as string };
    } else {
      newExtras[index] = { ...newExtras[index], note: value as string };
    }

    setEditState(prev => ({
      ...prev,
      [houseId]: {
        ...prev[houseId],
        extra_costs: newExtras
      }
    }));
  };

  const handleAddExtraCost = (houseId: string) => {
    const cost = costs[houseId];
    if (!cost) return;
    
    const currentEdits = editState[houseId]?.extra_costs || cost.extra_costs || [];
    setEditState(prev => ({
      ...prev,
      [houseId]: {
        ...prev[houseId],
        extra_costs: [...currentEdits, { name: 'Chi phí mới', amount: 0, note: '' }]
      }
    }));
  };

  const handleRemoveExtraCost = async (houseId: string, index: number) => {
    const cost = costs[houseId];
    if (!cost) return;
    
    const currentEdits = editState[houseId]?.extra_costs || cost.extra_costs || [];
    const newExtras = currentEdits.filter((_, i) => i !== index);
    
    // Auto-save immediately upon deletion
    const editsToSave = { ...(editState[houseId] || {}), extra_costs: newExtras };

    setIsSaving(prev => ({ ...prev, [houseId]: true }));
    try {
      const res = await updateCost(cost.id, houseId, editsToSave);
      if (res.success) {
        const newEditState = { ...editState };
        delete newEditState[houseId];
        setEditState(newEditState);
        toast.success('Đã xóa chi phí');
      } else {
        toast.error(res.error || 'Không thể xóa chi phí.');
      }
    } catch {
      toast.error('Không thể xóa chi phí.');
    } finally {
      setIsSaving(prev => ({ ...prev, [houseId]: false }));
    }
  };

  const handleSaveCost = async (houseId: string) => {
    const edits = editState[houseId];
    if (!edits) return;
    
    const cost = costs[houseId];
    if (!cost) return;

    setIsSaving(prev => ({ ...prev, [houseId]: true }));
    try {
      const res = await updateCost(cost.id, houseId, edits);
      if (res.success) {
        const newEditState = { ...editState };
        delete newEditState[houseId];
        setEditState(newEditState);
        toast.success('Lưu chi phí thành công');
      } else {
        toast.error(res.error || 'Không thể lưu thay đổi.');
      }
    } catch {
      toast.error('Không thể lưu thay đổi.');
    } finally {
      setIsSaving(prev => ({ ...prev, [houseId]: false }));
    }
  };

  const getActiveCostValue = (houseId: string, field: keyof HouseCost) => {
    if (editState[houseId] && editState[houseId][field] !== undefined) {
      return editState[houseId][field];
    }
    return costs[houseId]?.[field] ?? 0;
  };

  const getActiveExtraCosts = (houseId: string) => {
    if (editState[houseId] && editState[houseId].extra_costs !== undefined) {
      return editState[houseId].extra_costs!;
    }
    return costs[houseId]?.extra_costs || [];
  };

  const hasEdits = (houseId: string) => {
    return editState[houseId] !== undefined && Object.keys(editState[houseId]).length > 0;
  };

  return (
    <div className="space-y-8 safe-fade-in pb-12">
      {/* Header & Filters */}
      <PageHeader
        className="border-b border-border/40 pb-6"
        title={
          <span className="flex items-center gap-3">
            <Wallet className="size-6 text-primary md:size-7" />
            Doanh thu
          </span>
        }
        description="Theo dõi dòng tiền, chi phí vận hành và lợi nhuận ròng."
        action={
          <div className="flex w-full items-center gap-2 sm:w-auto md:gap-3">
            <div className="flex h-10 min-w-0 flex-1 items-center gap-2 overflow-hidden rounded-md border border-input bg-background px-2 py-2 text-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground focus-within:ring-1 focus-within:ring-ring sm:flex-none md:px-4">
              <Calendar className="hidden size-4 shrink-0 text-muted-foreground sm:block" />
              <Input
                type="month"
                value={period}
                onChange={(e) => handlePeriodChange(e.target.value)}
                className="h-full w-full min-w-0 border-none bg-transparent p-0 text-center text-sm font-medium shadow-none focus-visible:ring-0 sm:text-left md:w-[120px]"
              />
            </div>
            <div className="min-w-0 flex-1 sm:flex-none">
              <HouseSelectDropdown
                houses={houses}
                selectedHouseIds={selectedHouseIds}
                isOpen={isHouseSelectOpen}
                onOpenChange={setIsHouseSelectOpen}
                onToggleHouse={handleHouseToggleRequest}
                onSelectAll={handleSelectAllRequest}
              />
            </div>
          </div>
        }
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

      {/* Detail Costs Section - Only show when 1 house is selected to allow inline editing safely */}
      {selectedHouseIds.length === 1 && (
        <SectionCard
          title={`Chi tiết chi phí vận hành - ${houses.find(h => h.id === selectedHouseIds[0])?.name ?? ''}`}
          action={
            costs[selectedHouseIds[0]] && hasEdits(selectedHouseIds[0]) ? (
              <Button
                onClick={() => handleSaveCost(selectedHouseIds[0])}
                disabled={isSaving[selectedHouseIds[0]]}
                className="gap-2 font-bold"
              >
                {isSaving[selectedHouseIds[0]] ? 'Đang lưu...' : <><Save className="size-4" /> Lưu thay đổi</>}
              </Button>
            ) : undefined
          }
          bodyClassName="p-6"
        >
          {isLoadingCosts ? (
            <div className="py-12 text-center text-muted-foreground">Đang tải dữ liệu chi phí...</div>
          ) : !costs[selectedHouseIds[0]] ? (
            <EmptyState
              icon={AlertCircle}
              className="rounded-2xl border-2 border-dashed border-border/60 bg-secondary/20 py-16"
              title={`Chưa có chi phí cho kỳ ${period}`}
              description="Tạo bản ghi chi phí vận hành cho tháng này. Hệ thống sẽ tự động sao chép các chi phí cố định từ tháng trước nếu có."
              action={
                <Button onClick={() => handleCreateMonthCost(selectedHouseIds[0])} className="font-bold">
                  <Plus className="mr-2 size-4" /> Tạo chi phí tháng mới
                </Button>
              }
            />
          ) : (
            <div className="overflow-hidden rounded-xl border border-border/50">
              <Table>
                <TableHeader className="bg-secondary/50">
                  <TableRow>
                    <TableHead className="w-[25%] min-w-[100px] font-bold uppercase text-muted-foreground">Loại chi phí</TableHead>
                    <TableHead className="w-[15%] min-w-[100px] font-bold uppercase text-muted-foreground">Phân loại</TableHead>
                    <TableHead className="w-[20%] min-w-[120px] font-bold uppercase text-muted-foreground">Số tiền (VND)</TableHead>
                    <TableHead className="w-[25%] min-w-[200px] font-bold uppercase text-muted-foreground">Ghi chú</TableHead>
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
                    { key: 'maintenance', label: 'Tiền bảo trì, sửa chữa', type: 'Biến đổi' },
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
                          className="h-9 w-40 rounded-lg font-semibold focus-visible:ring-primary"
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
                          className="h-9 w-full min-w-[150px] rounded-lg font-semibold focus-visible:ring-primary"
                        />
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline">Tùy chỉnh</Badge>
                      </TableCell>
                      <TableCell>
                        <CurrencyInput
                          value={extra.amount}
                          onChange={(val) => handleExtraCostChange(selectedHouseIds[0], index, 'amount', val)}
                          className="h-9 w-40 rounded-lg font-semibold focus-visible:ring-primary"
                        />
                      </TableCell>
                      <TableCell>
                        <div
                          onClick={() => setEditingNoteFor({ houseId: selectedHouseIds[0], index })}
                          className={`min-h-[36px] w-full min-w-[250px] cursor-pointer whitespace-pre-wrap rounded-lg p-2.5 text-sm font-medium leading-relaxed transition-colors hover:bg-secondary/50 ${extra.note ? 'text-foreground' : 'italic text-muted-foreground'}`}
                        >
                          {extra.note || 'Bấm để thêm ghi chú...'}
                        </div>
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleRemoveExtraCost(selectedHouseIds[0], index)}
                          className="size-8 rounded-lg text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
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
                        className="-ml-2 font-bold text-primary hover:bg-primary/10 hover:text-primary/80"
                      >
                        <Plus className="mr-2 size-4" /> Thêm chi phí khác
                      </Button>
                    </TableCell>
                  </TableRow>
                </TableBody>
                <TableFooter className="border-t-2 border-primary/20 bg-primary/5">
                  <TableRow className="hover:bg-transparent">
                    <TableCell colSpan={2} className="py-5 text-right font-extrabold uppercase tracking-wider text-foreground">Tổng cộng chi phí:</TableCell>
                    <TableCell colSpan={3} className="py-5 text-xl font-extrabold text-warning">
                      {formatCurrency(
                        (getActiveCostValue(selectedHouseIds[0], 'rent') as number) +
                        (getActiveCostValue(selectedHouseIds[0], 'electricity') as number) +
                        (getActiveCostValue(selectedHouseIds[0], 'water') as number) +
                        (getActiveCostValue(selectedHouseIds[0], 'wifi') as number) +
                        (getActiveCostValue(selectedHouseIds[0], 'cleaning') as number) +
                        (getActiveCostValue(selectedHouseIds[0], 'maintenance') as number) +
                        getActiveExtraCosts(selectedHouseIds[0]).reduce((acc, curr) => acc + curr.amount, 0)
                      )}
                    </TableCell>
                  </TableRow>
                </TableFooter>
              </Table>
            </div>
          )}
        </SectionCard>
      )}

      {selectedHouseIds.length > 1 && (
        <EmptyState
          icon={Wallet}
          className="rounded-2xl border border-border/50 bg-secondary/30 py-8"
          title="Đang xem tổng hợp nhiều nhà"
          description="Bảng chi tiết chỉ hiển thị khi bạn chọn 1 nhà duy nhất. Hãy bỏ chọn các nhà khác để xem và chỉnh sửa chi tiết."
        />
      )}

      {selectedHouseIds.length === 0 && houses.length > 0 && (
        <EmptyState
          className="rounded-2xl border border-border/50 bg-secondary/30 py-8"
          title="Vui lòng chọn nhà trọ"
          description="Bạn cần chọn ít nhất 1 nhà trọ để xem thống kê doanh thu."
        />
      )}
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
