import { useState, useEffect, useMemo, useCallback, useRef } from 'react';
import { useHouseCostStore } from '@/data/houseCostData';
import { useHouseStore } from '@/data/houseData';
import { useSelectedStore, type TabType } from '@/data/selectedData';
import { useDirtyConfirm } from '@/hooks/useDirtyConfirm';
import type { HouseCost, ExtraCost } from '@/api/houseCost';
import { Wallet, TrendingUp, TrendingDown, DollarSign, Plus, Save, Trash2, Calendar, AlertCircle } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';

import { HouseSelectDropdown } from './HouseSelectDropdown';
import { CurrencyInput } from '@/components/ui/currency-input';
import { EditNoteModal } from './modals/EditNoteModal';

const formatCurrency = (amount: number) => {
  return new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND' }).format(amount);
};

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
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-border/40 pb-6">
        <div>
          <h1 className="text-2xl md:text-3xl font-extrabold tracking-tight text-foreground flex items-center gap-3">
            <Wallet className="w-7 h-7 md:w-8 md:h-8 text-primary" />
            Doanh thu
          </h1>
          <p className="text-sm md:text-base text-muted-foreground mt-1 md:mt-2 font-medium">
            Theo dõi dòng tiền, chi phí vận hành và lợi nhuận ròng.
          </p>
        </div>
        <div className="flex w-full md:w-auto items-center gap-2 md:gap-3">
          <div className="flex-1 md:flex-none flex items-center gap-2 bg-background hover:bg-accent hover:text-accent-foreground border border-input text-foreground h-10 px-2 md:px-4 py-2 rounded-md shadow-sm transition-colors focus-within:ring-1 focus-within:ring-ring overflow-hidden min-w-0">
            <Calendar className="w-4 h-4 text-muted-foreground shrink-0 hidden sm:block" />
            <Input 
              type="month" 
              value={period}
              onChange={(e) => handlePeriodChange(e.target.value)}
              className="border-none bg-transparent shadow-none focus-visible:ring-0 w-full md:w-[120px] min-w-0 font-medium h-full p-0 text-sm text-center sm:text-left"
            />
          </div>
          <div className="flex-1 md:flex-none min-w-0">
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
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div className="bg-card border border-border/60 rounded-xl p-4 shadow-sm">
          <div className="flex items-center gap-3 mb-2">
            <div className="w-8 h-8 rounded-full bg-emerald-100 flex items-center justify-center shrink-0">
              <TrendingUp className="w-4 h-4 text-emerald-600" />
            </div>
            <h3 className="text-xs font-bold text-muted-foreground uppercase tracking-wider">Tổng Doanh Thu</h3>
          </div>
          <p className="text-2xl font-extrabold text-foreground">
            {isLoadingSummaries ? '...' : formatCurrency(aggregatedSummary.totalRevenue)}
          </p>
        </div>

        <div className="bg-card border border-border/60 rounded-xl p-4 shadow-sm">
          <div className="flex items-center gap-3 mb-2">
            <div className="w-8 h-8 rounded-full bg-rose-100 flex items-center justify-center shrink-0">
              <TrendingDown className="w-4 h-4 text-rose-600" />
            </div>
            <h3 className="text-xs font-bold text-muted-foreground uppercase tracking-wider">Tổng Chi Phí</h3>
          </div>
          <p className="text-2xl font-extrabold text-foreground">
            {isLoadingSummaries || isLoadingCosts ? '...' : formatCurrency(aggregatedSummary.totalCost)}
          </p>
        </div>

        <div className={`bg-gradient-to-br ${aggregatedSummary.profit >= 0 ? 'from-primary/10 to-primary/5 border-primary/20' : 'from-rose-500/10 to-rose-500/5 border-rose-500/20'} border rounded-xl p-4 shadow-sm`}>
          <div className="flex items-center gap-3 mb-2">
            <div className={`w-8 h-8 rounded-full ${aggregatedSummary.profit >= 0 ? 'bg-primary/20' : 'bg-rose-500/20'} flex items-center justify-center shrink-0`}>
              <DollarSign className={`w-4 h-4 ${aggregatedSummary.profit >= 0 ? 'text-primary' : 'text-rose-600'}`} />
            </div>
            <h3 className="text-xs font-bold text-muted-foreground uppercase tracking-wider">Lợi Nhuận Ròng</h3>
          </div>
          <p className={`text-2xl font-extrabold ${aggregatedSummary.profit >= 0 ? 'text-primary' : 'text-rose-600'}`}>
            {isLoadingSummaries ? '...' : formatCurrency(aggregatedSummary.profit)}
          </p>
        </div>
      </div>

      {/* Detail Costs Section - Only show when 1 house is selected to allow inline editing safely */}
      {selectedHouseIds.length === 1 && (
        <div className="bg-card border border-border/60 rounded-2xl shadow-sm overflow-hidden">
          <div className="px-6 py-5 border-b border-border/40 flex justify-between items-center bg-secondary/10">
            <h3 className="font-bold text-foreground text-lg flex items-center gap-2">
              Chi tiết chi phí vận hành - {houses.find(h => h.id === selectedHouseIds[0])?.name}
            </h3>
            {costs[selectedHouseIds[0]] && hasEdits(selectedHouseIds[0]) && (
              <Button 
                onClick={() => handleSaveCost(selectedHouseIds[0])}
                disabled={isSaving[selectedHouseIds[0]]}
                className="rounded-xl shadow-sm font-bold gap-2"
              >
                {isSaving[selectedHouseIds[0]] ? 'Đang lưu...' : <><Save className="w-4 h-4"/> Lưu thay đổi</>}
              </Button>
            )}
          </div>
          
          <div className="p-6">
            {isLoadingCosts ? (
              <div className="text-center py-12 text-muted-foreground">Đang tải dữ liệu chi phí...</div>
            ) : !costs[selectedHouseIds[0]] ? (
              <div className="text-center py-16 flex flex-col items-center justify-center border-2 border-dashed border-border/60 rounded-2xl bg-secondary/20">
                <div className="w-16 h-16 bg-background rounded-full shadow-sm border border-border/40 flex items-center justify-center mb-4">
                  <AlertCircle className="w-8 h-8 text-muted-foreground" />
                </div>
                <h3 className="text-lg font-bold text-foreground mb-2">Chưa có chi phí cho kỳ {period}</h3>
                <p className="text-muted-foreground mb-6 max-w-sm">
                  Tạo bản ghi chi phí vận hành cho tháng này. Hệ thống sẽ tự động sao chép các chi phí cố định từ tháng trước nếu có.
                </p>
                <Button 
                  onClick={() => handleCreateMonthCost(selectedHouseIds[0])}
                  className="rounded-xl shadow-sm font-bold"
                >
                  <Plus className="w-4 h-4 mr-2" /> Tạo chi phí tháng mới
                </Button>
              </div>
            ) : (
              <div className="space-y-6">
                <div className="overflow-x-auto rounded-xl border border-border/50">
                  <table className="w-full text-sm text-left">
                    <thead className="text-xs text-muted-foreground uppercase bg-secondary/50 border-b border-border/50">
                      <tr>
                        <th className="px-6 py-4 font-bold w-[25%] min-w-[100px]">Loại chi phí</th>
                        <th className="px-6 py-4 font-bold w-[15%] whitespace-nowrap min-w-[100px]">Phân loại</th>
                        <th className="px-6 py-4 font-bold w-[20%] min-w-[120px]">Số tiền (VND)</th>
                        <th className="px-6 py-4 font-bold w-[25%] min-w-[200px]">Ghi chú</th>
                        <th className="px-6 py-4 font-bold text-right w-[5%] min-w-[60px]"></th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border/30">
                      {/* Fixed Layout Costs */}
                      {[
                        { key: 'rent', label: 'Tiền thuê nhà', type: 'Cố định' },
                        { key: 'electricity', label: 'Tiền điện', type: 'Biến đổi' },
                        { key: 'water', label: 'Tiền nước', type: 'Biến đổi' },
                        { key: 'wifi', label: 'Tiền Internet', type: 'Cố định' },
                        { key: 'cleaning', label: 'Tiền vệ sinh, rác', type: 'Cố định' },
                        { key: 'maintenance', label: 'Tiền bảo trì, sửa chữa', type: 'Biến đổi' },
                      ].map((item) => (
                        <tr key={item.key} className="hover:bg-secondary/20 transition-colors group">
                          <td className="px-6 py-4 font-semibold text-foreground">{item.label}</td>
                          <td className="px-6 py-4">
                            <span className={`inline-flex whitespace-nowrap px-2.5 py-1 text-[10px] font-bold rounded-full ${item.type === 'Cố định' ? 'bg-blue-100 text-blue-700' : 'bg-orange-100 text-orange-700'}`}>
                              {item.type}
                            </span>
                          </td>
                          <td className="px-6 py-3">
                            <CurrencyInput
                              value={getActiveCostValue(selectedHouseIds[0], item.key as keyof HouseCost) as number}
                              onChange={(val) => handleFieldChange(selectedHouseIds[0], item.key as keyof HouseCost, val)}
                              className="w-40 font-semibold focus-visible:ring-primary h-9 rounded-lg"
                            />
                          </td>
                          <td className="px-6 py-4"></td>
                          <td className="px-6 py-4"></td>
                        </tr>
                      ))}

                      {/* Extra Costs */}
                      {getActiveExtraCosts(selectedHouseIds[0]).map((extra, index) => (
                        <tr key={`extra-${index}`} className="hover:bg-secondary/20 transition-colors group">
                          <td className="px-6 py-3">
                            <Input
                              value={extra.name}
                              onChange={(e) => handleExtraCostChange(selectedHouseIds[0], index, 'name', e.target.value)}
                              placeholder="Tên chi phí..."
                              className="w-full min-w-[150px] font-semibold focus-visible:ring-primary h-9 rounded-lg"
                            />
                          </td>
                          <td className="px-6 py-4">
                            <span className="inline-flex whitespace-nowrap px-2.5 py-1 text-[10px] font-bold rounded-full bg-purple-100 text-purple-700">Tùy chỉnh</span>
                          </td>
                          <td className="px-6 py-3">
                            <CurrencyInput
                              value={extra.amount}
                              onChange={(val) => handleExtraCostChange(selectedHouseIds[0], index, 'amount', val)}
                              className="w-40 font-semibold focus-visible:ring-primary h-9 rounded-lg"
                            />
                          </td>
                          <td className="px-6 py-3">
                            <div 
                              onClick={() => setEditingNoteFor({ houseId: selectedHouseIds[0], index })}
                              className={`w-full min-w-[250px] font-medium min-h-[36px] p-2.5 text-sm cursor-pointer rounded-lg hover:bg-secondary/50 transition-colors whitespace-pre-wrap leading-relaxed ${extra.note ? 'text-foreground' : 'text-muted-foreground italic'}`}
                            >
                              {extra.note || 'Bấm để thêm ghi chú...'}
                            </div>
                          </td>
                          <td className="px-6 py-3 text-right">
                            <Button 
                              variant="ghost" 
                              size="icon" 
                              onClick={() => handleRemoveExtraCost(selectedHouseIds[0], index)}
                              className="text-muted-foreground hover:text-rose-500 hover:bg-rose-50 h-8 w-8 rounded-lg"
                            >
                              <Trash2 className="w-4 h-4" />
                            </Button>
                          </td>
                        </tr>
                      ))}
                      <tr>
                        <td colSpan={5} className="px-6 py-3">
                          <Button 
                            variant="ghost" 
                            onClick={() => handleAddExtraCost(selectedHouseIds[0])}
                            className="text-primary hover:text-primary/80 hover:bg-primary/10 font-bold -ml-2"
                          >
                            <Plus className="w-4 h-4 mr-2" /> Thêm chi phí khác
                          </Button>
                        </td>
                      </tr>
                    </tbody>
                    <tfoot className="bg-primary/5 border-t-2 border-primary/20">
                      <tr>
                        <td colSpan={2} className="px-6 py-5 font-extrabold text-foreground text-right uppercase tracking-wider">Tổng cộng chi phí:</td>
                        <td colSpan={3} className="px-6 py-5 font-extrabold text-rose-600 text-xl">
                          {formatCurrency(
                            (getActiveCostValue(selectedHouseIds[0], 'rent') as number) +
                            (getActiveCostValue(selectedHouseIds[0], 'electricity') as number) +
                            (getActiveCostValue(selectedHouseIds[0], 'water') as number) +
                            (getActiveCostValue(selectedHouseIds[0], 'wifi') as number) +
                            (getActiveCostValue(selectedHouseIds[0], 'cleaning') as number) +
                            (getActiveCostValue(selectedHouseIds[0], 'maintenance') as number) +
                            getActiveExtraCosts(selectedHouseIds[0]).reduce((acc, curr) => acc + curr.amount, 0)
                          )}
                        </td>
                      </tr>
                    </tfoot>
                  </table>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {selectedHouseIds.length > 1 && (
        <div className="bg-secondary/30 border border-border/50 rounded-2xl p-8 text-center">
          <Wallet className="w-12 h-12 text-muted-foreground/30 mx-auto mb-4" />
          <h3 className="text-lg font-bold text-foreground mb-2">Đang xem tổng hợp nhiều nhà</h3>
          <p className="text-muted-foreground">
            Bảng chi tiết chỉ hiển thị khi bạn chọn 1 nhà duy nhất. Hãy bỏ chọn các nhà khác để xem và chỉnh sửa chi tiết.
          </p>
        </div>
      )}
      
      {selectedHouseIds.length === 0 && houses.length > 0 && (
        <div className="bg-secondary/30 border border-border/50 rounded-2xl p-8 text-center">
          <h3 className="text-lg font-bold text-foreground mb-2">Vui lòng chọn nhà trọ</h3>
          <p className="text-muted-foreground">
            Bạn cần chọn ít nhất 1 nhà trọ để xem thống kê doanh thu.
          </p>
        </div>
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
