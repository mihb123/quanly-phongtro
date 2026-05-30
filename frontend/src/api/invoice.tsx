import { apiClient as api } from './client';

export interface Invoice {
  id: string;
  room_id: string;
  house_id: string;
  room_name: string;
  period: string;
  room_fee: number;
  old_electricity_index: number;
  new_electricity_index: number;
  electricity_fee: number;
  old_water_index: number;
  new_water_index: number;
  water_fee: number;
  wifi_fee: number;
  parking_fee: number;
  service_fee: number;
  other_fee: number;
  discount: number;
  tenant_count: number;
  vehicle_count: number;
  extra_person_fee: number;
  extra_person_threshold: number;
  extra_person_fee_unit: number;
  extra_vehicle_fee: number;
  extra_vehicle_threshold: number;
  extra_vehicle_fee_unit: number;
  total_amount: number;
  status: string;
  created_at: string;
}

export interface CreateInvoicePayload {
  room_id: string;
  period: string;
  new_electricity_index: number;
  new_water_index: number;
  other_fee?: number;
  discount?: number;
  vehicle_count: number;
}

export interface InvoiceFilter {
  house_id?: string;
  room_id?: string;
  period?: string;
  status?: string;
  page?: number;
  limit?: number;
}

export const getInvoices = async (filter?: InvoiceFilter) => {
  const response = await api.get<Invoice[]>('/invoice', { params: filter });
  return response.data;
};

export const getInvoiceById = async (id: string) => {
  const response = await api.get<Invoice>(`/invoice/${id}`);
  return response.data;
};

export const createInvoice = async (payload: CreateInvoicePayload) => {
  const response = await api.post<Invoice>('/invoice/', payload);
  return response.data;
};

export const payInvoice = async (id: string) => {
  const response = await api.patch<Invoice>(`/invoice/${id}/pay`);
  return response.data;
};

export const unpayInvoice = async (id: string) => {
  const response = await api.patch<Invoice>(`/invoice/${id}/unpay`);
  return response.data;
};
