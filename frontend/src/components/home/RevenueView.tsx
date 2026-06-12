import { useState, useEffect, useMemo } from 'react';
import { useHouseCostStore } from '@/data/houseCostData';
import { useHouseStore } from '@/data/houseData';
import type { HouseCost, ExtraCost } from '@/api/houseCost';
import { Wallet, TrendingUp, TrendingDown, DollarSign, Plus, Save, Trash2, Calendar, AlertCircle } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';

import { HouseSelectDropdown } from './HouseSelectDropdown';
import { CurrencyInput } from '@/components/ui/currency-input';

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

  const [editState, setEditState] = useState<Record<string, Partial<HouseCost>>>({});
  const [isSaving, setIsSaving] = useState<Record<string, boolean>>({});
  const [isHouseSelectOpen, setIsHouseSelectOpen] = useState(false);

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
    if (selectedHouseIds.length === 0 && houses.length > 0) {
      // Mặc định chọn 1 house mới nhất (house đầu tiên)
      setSelectedHouseIds([houses[0].id]);
    }
  }, [houses, selectedHouseIds.length, setSelectedHouseIds]);

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

  const handleHouseToggle = (houseId: string) => {
    if (selectedHouseIds.includes(houseId)) {
      setSelectedHouseIds(selectedHouseIds.filter(id => id !== houseId));
    } else {
      setSelectedHouseIds([...selectedHouseIds, houseId]);
    }
  };

  const handleCreateMonthCost = async (houseId: string) => {
    try {
      await createCostForHouse(houseId);
      toast.success('Tạo chi phí tháng mới thành công');
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
    } else {
      newExtras[index] = { ...newExtras[index], name: value as string };
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
        extra_costs: [...currentEdits, { name: 'Chi phí mới', amount: 0 }]
      }
    }));
  };

  const handleRemoveExtraCost = (houseId: string, index: number) => {
    const cost = costs[houseId];
    if (!cost) return;
    
    const currentEdits = editState[houseId]?.extra_costs || cost.extra_costs || [];
    const newExtras = currentEdits.filter((_, i) => i !== index);
    
    setEditState(prev => ({
      ...prev,
      [houseId]: {
        ...prev[houseId],
        extra_costs: newExtras
      }
    }));
  };

  const handleSaveCost = async (houseId: string) => {
    const edits = editState[houseId];
    if (!edits) return;
    
    const cost = costs[houseId];
    if (!cost) return;

    setIsSaving(prev => ({ ...prev, [houseId]: true }));
    try {
      await updateCost(cost.id, houseId, edits);
      const newEditState = { ...editState };
      delete newEditState[houseId];
      setEditState(newEditState);
      toast.success('Lưu chi phí thành công');
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
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-6 border-b border-border/40 pb-6">
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight text-foreground flex items-center gap-3">
            <Wallet className="w-8 h-8 text-primary" />
            Doanh thu
          </h1>
          <p className="text-muted-foreground mt-2 font-medium">
            Theo dõi dòng tiền, chi phí vận hành và lợi nhuận ròng.
          </p>
        </div>
        <div className="shrink-0 flex items-center gap-3 bg-secondary/30 p-1.5 rounded-xl border border-border/40">
          <Calendar className="w-4 h-4 text-muted-foreground ml-2" />
          <Input 
            type="month" 
            value={period}
            onChange={(e) => setPeriod(e.target.value)}
            className="border-none bg-transparent shadow-none focus-visible:ring-0 w-36 font-semibold"
          />
        </div>
      </div>

      {/* House Filter */}
      <div className="flex items-center gap-3">
        <HouseSelectDropdown
          houses={houses}
          selectedHouseIds={selectedHouseIds}
          isOpen={isHouseSelectOpen}
          onOpenChange={setIsHouseSelectOpen}
          onToggleHouse={handleHouseToggle}
          onSelectAll={() => {
            if (selectedHouseIds.length === houses.length) {
              setSelectedHouseIds([]);
            } else {
              setSelectedHouseIds(houses.map(h => h.id));
            }
          }}
        />
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-card border border-border/60 rounded-2xl p-6 shadow-sm relative overflow-hidden">
          <div className="absolute top-0 right-0 p-4 opacity-10">
            <TrendingUp className="w-24 h-24 text-emerald-500" />
          </div>
          <div className="flex items-center gap-3 mb-4 relative z-10">
            <div className="w-10 h-10 rounded-full bg-emerald-100 flex items-center justify-center shrink-0">
              <TrendingUp className="w-5 h-5 text-emerald-600" />
            </div>
            <h3 className="text-sm font-bold text-muted-foreground uppercase tracking-wider">Tổng Doanh Thu</h3>
          </div>
          <p className="text-3xl font-extrabold text-foreground relative z-10">
            {isLoadingSummaries ? '...' : formatCurrency(aggregatedSummary.totalRevenue)}
          </p>
          <p className="text-xs font-medium text-muted-foreground mt-2">Từ hóa đơn đã thanh toán</p>
        </div>

        <div className="bg-card border border-border/60 rounded-2xl p-6 shadow-sm relative overflow-hidden">
          <div className="absolute top-0 right-0 p-4 opacity-10">
            <TrendingDown className="w-24 h-24 text-rose-500" />
          </div>
          <div className="flex items-center gap-3 mb-4 relative z-10">
            <div className="w-10 h-10 rounded-full bg-rose-100 flex items-center justify-center shrink-0">
              <TrendingDown className="w-5 h-5 text-rose-600" />
            </div>
            <h3 className="text-sm font-bold text-muted-foreground uppercase tracking-wider">Tổng Chi Phí</h3>
          </div>
          <p className="text-3xl font-extrabold text-foreground relative z-10">
            {isLoadingSummaries || isLoadingCosts ? '...' : formatCurrency(aggregatedSummary.totalCost)}
          </p>
          <p className="text-xs font-medium text-muted-foreground mt-2">Chi phí vận hành các nhà</p>
        </div>

        <div className={`bg-gradient-to-br ${aggregatedSummary.profit >= 0 ? 'from-primary/10 to-primary/5 border-primary/20' : 'from-rose-500/10 to-rose-500/5 border-rose-500/20'} border rounded-2xl p-6 shadow-sm relative overflow-hidden`}>
          <div className="absolute top-0 right-0 p-4 opacity-10">
            <DollarSign className={`w-24 h-24 ${aggregatedSummary.profit >= 0 ? 'text-primary' : 'text-rose-500'}`} />
          </div>
          <div className="flex items-center gap-3 mb-4 relative z-10">
            <div className={`w-10 h-10 rounded-full ${aggregatedSummary.profit >= 0 ? 'bg-primary/20' : 'bg-rose-500/20'} flex items-center justify-center shrink-0`}>
              <DollarSign className={`w-5 h-5 ${aggregatedSummary.profit >= 0 ? 'text-primary' : 'text-rose-600'}`} />
            </div>
            <h3 className="text-sm font-bold text-muted-foreground uppercase tracking-wider">Lợi Nhuận Ròng</h3>
          </div>
          <p className={`text-3xl font-extrabold relative z-10 ${aggregatedSummary.profit >= 0 ? 'text-primary' : 'text-rose-600'}`}>
            {isLoadingSummaries ? '...' : formatCurrency(aggregatedSummary.profit)}
          </p>
          <p className="text-xs font-medium text-muted-foreground mt-2">Tổng thu trừ đi tổng chi</p>
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
                        <th className="px-6 py-4 font-bold">Loại chi phí</th>
                        <th className="px-6 py-4 font-bold">Phân loại</th>
                        <th className="px-6 py-4 font-bold">Số tiền (VND)</th>
                        <th className="px-6 py-4 font-bold text-right w-16"></th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border/30">
                      {/* Fixed Layout Costs */}
                      {[
                        { key: 'rent', label: 'Tiền thuê nhà (khoán)', type: 'Cố định' },
                        { key: 'electricity', label: 'Tiền điện', type: 'Biến đổi' },
                        { key: 'water', label: 'Tiền nước', type: 'Biến đổi' },
                        { key: 'wifi', label: 'Tiền WiFi/Internet', type: 'Cố định' },
                        { key: 'cleaning', label: 'Tiền vệ sinh, rác', type: 'Cố định' },
                        { key: 'maintenance', label: 'Tiền bảo trì, sửa chữa', type: 'Biến đổi' },
                      ].map((item) => (
                        <tr key={item.key} className="hover:bg-secondary/20 transition-colors group">
                          <td className="px-6 py-4 font-semibold text-foreground">{item.label}</td>
                          <td className="px-6 py-4">
                            <span className={`px-2.5 py-1 text-[10px] font-bold rounded-full ${item.type === 'Cố định' ? 'bg-blue-100 text-blue-700' : 'bg-orange-100 text-orange-700'}`}>
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
                              className="font-semibold focus-visible:ring-primary h-9 rounded-lg"
                            />
                          </td>
                          <td className="px-6 py-4">
                            <span className="px-2.5 py-1 text-[10px] font-bold rounded-full bg-purple-100 text-purple-700">Tùy chỉnh</span>
                          </td>
                          <td className="px-6 py-3">
                            <CurrencyInput
                              value={extra.amount}
                              onChange={(val) => handleExtraCostChange(selectedHouseIds[0], index, 'amount', val)}
                              className="w-40 font-semibold focus-visible:ring-primary h-9 rounded-lg"
                            />
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
                    </tbody>
                    <tfoot className="bg-primary/5 border-t-2 border-primary/20">
                      <tr>
                        <td colSpan={2} className="px-6 py-5 font-extrabold text-foreground text-right uppercase tracking-wider">Tổng cộng chi phí:</td>
                        <td colSpan={2} className="px-6 py-5 font-extrabold text-rose-600 text-xl">
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

                <div className="flex justify-between items-center">
                  <Button 
                    variant="outline" 
                    onClick={() => handleAddExtraCost(selectedHouseIds[0])}
                    className="rounded-xl border-dashed border-2 hover:bg-secondary font-semibold"
                  >
                    <Plus className="w-4 h-4 mr-2" /> Thêm chi phí khác
                  </Button>
                  
                  <div className="flex items-center gap-3">
                    <span className="text-sm font-semibold text-muted-foreground">Ghi chú chung:</span>
                    <Input
                      value={getActiveCostValue(selectedHouseIds[0], 'note') as string}
                      onChange={(e) => handleFieldChange(selectedHouseIds[0], 'note', e.target.value)}
                      placeholder="Ghi chú thêm cho tháng này..."
                      className="w-64 font-medium focus-visible:ring-primary rounded-lg"
                    />
                  </div>
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
    </div>
  );
}
