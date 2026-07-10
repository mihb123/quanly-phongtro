import { useState, useCallback } from 'react';
import { toast } from 'sonner';
import { useHouseCostStore } from '@/data/houseCostData';
import type { HouseCost, ExtraCost } from '@/api/houseCost';

// useHouseCostEditor owns the per-house draft edits for operating costs and the create/save/delete
// operations against the house-cost store. It isolates cost-editing state orchestration from the
// RevenueView presentation; behavior mirrors the previous inline implementation.
export function useHouseCostEditor() {
  const { costs, createCostForHouse, updateCost } = useHouseCostStore();

  const [editState, setEditState] = useState<Record<string, Partial<HouseCost>>>({});
  const [isSaving, setIsSaving] = useState<Record<string, boolean>>({});

  const hasAnyEdits = Object.keys(editState).length > 0;

  const clearEdits = useCallback(() => setEditState({}), []);

  const hasEdits = (houseId: string) => {
    return editState[houseId] !== undefined && Object.keys(editState[houseId]).length > 0;
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

  return {
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
  };
}
