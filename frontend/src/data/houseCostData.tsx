import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { createMonthlyCost, getMonthlyCost, updateMonthlyCost, getRevenueSummaries } from '@/api/houseCost';
import type { HouseCost, RevenueSummary } from '@/api/houseCost';

interface HouseCostState {
  costs: Record<string, HouseCost | null>; // keyed by houseId
  summaries: RevenueSummary[];
  isLoadingCosts: boolean;
  isLoadingSummaries: boolean;
  selectedHouseIds: string[];
  period: string; // yyyy-mm

  setSelectedHouseIds: (ids: string[]) => void;
  setPeriod: (period: string) => void;
  fetchCosts: () => Promise<void>;
  fetchSummaries: () => Promise<void>;
  createCostForHouse: (houseId: string) => Promise<{success: boolean, error?: string}>;
  updateCost: (id: string, houseId: string, payload: Partial<HouseCost>) => Promise<{success: boolean, error?: string}>;
}

const getCurrentPeriod = () => {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
};

export const useHouseCostStore = create<HouseCostState>()(
  persist(
    (set, get) => ({
      costs: {},
      summaries: [],
      isLoadingCosts: false,
      isLoadingSummaries: false,
      selectedHouseIds: [],
      period: getCurrentPeriod(),

      setSelectedHouseIds: (ids) => {
        set({ selectedHouseIds: ids });
        get().fetchCosts();
        get().fetchSummaries();
      },

      setPeriod: (period) => {
        set({ period });
        get().fetchCosts();
        get().fetchSummaries();
      },

  fetchCosts: async () => {
    const { selectedHouseIds, period } = get();
    if (selectedHouseIds.length === 0) {
      set({ costs: {} });
      return;
    }

    set({ isLoadingCosts: true });
    try {
      const newCosts: Record<string, HouseCost | null> = {};
      await Promise.all(
        selectedHouseIds.map(async (houseId) => {
          const cost = await getMonthlyCost(houseId, period);
          newCosts[houseId] = cost;
        })
      );
      set({ costs: newCosts });
    } catch (error) {
      console.error('Failed to fetch house costs:', error);
    } finally {
      set({ isLoadingCosts: false });
    }
  },

  fetchSummaries: async () => {
    const { selectedHouseIds, period } = get();
    if (selectedHouseIds.length === 0) {
      set({ summaries: [] });
      return;
    }

    set({ isLoadingSummaries: true });
    try {
      const summaries = await getRevenueSummaries(selectedHouseIds, period);
      set({ summaries: summaries || [] });
    } catch (error) {
      console.error('Failed to fetch revenue summaries:', error);
    } finally {
      set({ isLoadingSummaries: false });
    }
  },

  createCostForHouse: async (houseId: string) => {
    const { period } = get();
    try {
      const newCost = await createMonthlyCost(houseId, period);
      set((state) => {
        let found = false;
        const newSummaries = state.summaries.map(s => {
          if (s.house_id === houseId && s.period === period) {
            found = true;
            return {
              ...s,
              total_cost: newCost.total_cost || 0,
              profit: (s.total_revenue || 0) - (newCost.total_cost || 0)
            };
          }
          return s;
        });
        if (!found) {
          newSummaries.push({
            id: '',
            house_id: houseId,
            period: period,
            total_revenue: 0,
            total_cost: newCost.total_cost || 0,
            profit: -(newCost.total_cost || 0),
            updated_at: new Date().toISOString()
          });
        }
        return {
          costs: { ...state.costs, [houseId]: newCost },
          summaries: newSummaries
        };
      });
      return { success: true };
    } catch (error) {
      const err = error as Error & { response?: { data?: { message?: string } } };
      console.error('Failed to create monthly cost:', err);
      return { success: false, error: err?.response?.data?.message || err?.message || 'Lỗi khi tạo chi phí' };
    }
  },

  updateCost: async (id: string, houseId: string, payload: Partial<HouseCost>) => {
    try {
      const currentCost = get().costs[houseId];
      if (!currentCost) throw new Error("Cost not found in state");

      const updatePayload = {
        ...currentCost,
        house_id: houseId,
        period: currentCost.period || get().period,
        ...payload
      };
      await updateMonthlyCost(id, updatePayload);
      // Update local state optimistic/after
      set((state) => {
        const current = state.costs[houseId];
        if (!current) return state;
        
        // Calculate total cost optimistic
        let total = (payload.rent ?? current.rent) +
                    (payload.electricity ?? current.electricity) +
                    (payload.water ?? current.water) +
                    (payload.wifi ?? current.wifi) +
                    (payload.cleaning ?? current.cleaning) +
                    (payload.maintenance ?? current.maintenance);
                    
        const extras = payload.extra_costs ?? current.extra_costs ?? [];
        extras.forEach(ex => total += ex.amount);

        let found = false;
        const newSummaries = state.summaries.map(s => {
          if (s.house_id === houseId && s.period === (current.period || get().period)) {
            found = true;
            return {
              ...s,
              total_cost: total,
              profit: (s.total_revenue || 0) - total
            };
          }
          return s;
        });

        if (!found) {
          newSummaries.push({
            id: '',
            house_id: houseId,
            period: current.period || get().period,
            total_revenue: 0,
            total_cost: total,
            profit: -total,
            updated_at: new Date().toISOString()
          });
        }

        return {
          costs: {
            ...state.costs,
            [houseId]: {
              ...current,
              ...payload,
              total_cost: total
            }
          },
          summaries: newSummaries
        };
      });
      return { success: true };
      } catch (error) {
        const err = error as Error & { response?: { data?: { message?: string } } };
        console.error('Failed to update house cost:', err);
        return { success: false, error: err?.response?.data?.message || err?.message || 'Lỗi khi cập nhật chi phí' };
      }
    },
  }),
  {
    name: 'house-cost-storage',
    partialize: (state) => ({ selectedHouseIds: state.selectedHouseIds }),
  }
));
