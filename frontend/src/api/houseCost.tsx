import { apiClient } from './client';

export interface ExtraCost {
  name: string;
  amount: number;
  note?: string;
}

export interface HouseCost {
  id: string;
  house_id: string;
  period: string;
  rent: number;
  electricity: number;
  water: number;
  wifi: number;
  cleaning: number;
  maintenance: number;
  extra_costs: ExtraCost[];
  note: string;
  total_cost: number;
  created_at: string;
  updated_at: string;
}

export interface RevenueSummary {
  id: string;
  house_id: string;
  period: string;
  total_revenue: number;
  total_cost: number;
  profit: number;
  updated_at: string;
}

export const createMonthlyCost = async (houseId: string, period: string): Promise<HouseCost> => {
  const response = await apiClient.post('/house-cost', {
    house_id: houseId,
    period: period,
  });
  return response.data.data;
};

export const getMonthlyCost = async (houseId: string, period: string): Promise<HouseCost | null> => {
  try {
    const response = await apiClient.get('/house-cost', {
      params: {
        house_id: houseId,
        period: period,
      },
    });
    return response.data.data;
  } catch (error) {
    const err = error as { response?: { status: number } };
    if (err.response && err.response.status === 404) {
      return null;
    }
    throw error;
  }
};

export const updateMonthlyCost = async (id: string, payload: Partial<HouseCost>): Promise<void> => {
  await apiClient.patch(`/house-cost/${id}`, payload);
};

export const getRevenueSummaries = async (houseIds: string[], period: string): Promise<RevenueSummary[]> => {
  if (houseIds.length === 0) return [];
  const response = await apiClient.get('/revenue-summary', {
    params: {
      house_ids: houseIds.join(','),
      period: period,
    },
  });
  return response.data.data;
};
