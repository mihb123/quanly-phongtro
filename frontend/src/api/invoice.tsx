import { apiClient as api } from './client';

export interface InvoiceFeeItem {
  name: string;
  amount: number;
}

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
  other_fees?: InvoiceFeeItem[];
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
  status: string; // 'UNPAID', 'PAID', 'PENDING_VERIFICATION'
  transaction_image_path?: string | null;
  created_at: string;
}

export interface CreateInvoicePayload {
  room_id: string;
  period: string;
  old_electricity_index?: number;
  new_electricity_index: number;
  old_water_index?: number;
  new_water_index: number;
  other_fee?: number;
  other_fees?: InvoiceFeeItem[];
  discount?: number;
  vehicle_count: number;
  tenant_count?: number;
  exclude_room_fee?: boolean;
}

export interface InvoiceFilter {
  house_id?: string;
  room_id?: string;
  period?: string;
  status?: string;
  page?: number;
  limit?: number;
}

export interface InvoiceAmounts {
  invoice_count: number;
  room_fee: number;
  electricity_fee: number;
  water_fee: number;
  wifi_fee: number;
  parking_fee: number;
  service_fee: number;
  extra_person_fee: number;
  extra_vehicle_fee: number;
  other_fee: number;
  discount: number;
  total_amount: number;
}

export interface InvoiceBreakdown {
  expected: InvoiceAmounts;
  collected: InvoiceAmounts;
}

export interface InvoiceBreakdownFilter {
  house_id?: string;
  house_ids?: string;
  room_id?: string;
  period?: string;
  status?: string;
}

export const getInvoices = async (filter?: InvoiceFilter) => {
  const response = await api.get<Invoice[]>('/invoice', { params: filter });
  return response.data;
};

export interface Paged<T> {
  items: T[];
  total: number;
}

export const getInvoicesPage = async (filter: InvoiceFilter): Promise<Paged<Invoice>> => {
  const response = await api.get<Invoice[]>('/invoice', { params: filter });
  const items = response.data || [];
  const total = Number(response.headers['x-total-count']);
  return { items, total: Number.isFinite(total) ? total : items.length };
};

const ALL_INVOICES_PAGE = 100;

// Đi hết các trang (backend giới hạn 100 dòng/trang) để lấy đủ hóa đơn theo bộ lọc.
export const getAllInvoices = async (filter: Omit<InvoiceFilter, 'page' | 'limit'>) => {
  const first = await getInvoicesPage({ ...filter, page: 1, limit: ALL_INVOICES_PAGE });
  const pages = Math.ceil(first.total / ALL_INVOICES_PAGE);
  if (pages <= 1) return first.items;
  const rest = await Promise.all(
    Array.from({ length: pages - 1 }, (_, i) => getInvoicesPage({ ...filter, page: i + 2, limit: ALL_INVOICES_PAGE })),
  );
  return [...first.items, ...rest.flatMap(page => page.items)];
};

export const getInvoiceBreakdown = async (filter?: InvoiceBreakdownFilter) => {
  const response = await api.get<InvoiceBreakdown>('/invoice/breakdown', { params: filter });
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

export const deleteInvoice = async (id: string) => {
  const response = await api.delete(`/invoice/${id}`);
  return response.data;
};

// getInvoiceImageBlob fetches the rendered invoice image as binary data for preview/download.
export const getInvoiceImageBlob = async (id: string) => {
  const response = await api.get(`/invoice/${id}/image`, { responseType: 'blob' });
  return response.data;
};
