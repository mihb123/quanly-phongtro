import { create } from 'zustand';
import {
  type Invoice,
  type InvoiceFilter,
  type Paged,
  createInvoice,
  type CreateInvoicePayload,
  payInvoice,
  unpayInvoice,
  deleteInvoice,
} from '../api/invoice';
import { invalidateQueries, queryKey, updateQueries } from '@/lib/queryCache';
import { FILTER_TTL, readPageSize, readStorage, removeStorage, writeStorage } from '@/lib/storage';

export const INVOICE_FILTERS_KEY = 'invoices:filters';
export const INVOICE_PAGE_SIZE_KEY = 'invoices:pageSize';
export const UNPAID_INVOICE_STATUSES = 'UNPAID,PARTIALLY_PAID,PENDING_VERIFICATION';

export type InvoiceListFilter = Required<Pick<InvoiceFilter, 'house_id' | 'room_id' | 'period' | 'status' | 'page' | 'limit'>>;

type PersistedFilter = Pick<InvoiceListFilter, 'house_id' | 'room_id' | 'period' | 'status'>;

export const currentPeriod = () => {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
};

const defaultFilter = (): InvoiceListFilter => ({
  house_id: '',
  room_id: '',
  period: currentPeriod(),
  status: '',
  page: 1,
  limit: readPageSize(INVOICE_PAGE_SIZE_KEY),
});

function restoreFilter(): InvoiceListFilter {
  const saved = readStorage<Partial<PersistedFilter>>(INVOICE_FILTERS_KEY, { ttl: FILTER_TTL }) || {};
  const base = defaultFilter();
  return {
    ...base,
    house_id: typeof saved.house_id === 'string' ? saved.house_id : base.house_id,
    room_id: typeof saved.room_id === 'string' ? saved.room_id : base.room_id,
    period: typeof saved.period === 'string' && /^\d{4}-\d{2}$/.test(saved.period) ? saved.period : base.period,
    status: typeof saved.status === 'string' ? saved.status : base.status,
  };
}

export function countActiveInvoiceFilters(filter: InvoiceListFilter) {
  return [filter.house_id, filter.room_id, filter.status, filter.period !== currentPeriod() ? filter.period : '']
    .filter(Boolean).length;
}

export const invoiceListKey = (filter: InvoiceListFilter) => queryKey('invoices:list', filter);

export const invoiceBreakdownKey = (filter: Pick<InvoiceFilter, 'house_id' | 'room_id' | 'period' | 'status'>) =>
  queryKey('invoices:breakdown', {
    house_id: filter.house_id,
    room_id: filter.room_id,
    period: filter.period,
    status: filter.status,
  });

const refreshInvoiceQueries = () => {
  invalidateQueries('invoices:');
  invalidateQueries('revenue:');
};

const setInvoiceStatus = (id: string, status: string) =>
  updateQueries<Paged<Invoice>>('invoices:list', page => ({
    ...page,
    items: page.items.map(inv => (inv.id === id ? { ...inv, status } : inv)),
  }));

interface InvoiceDataState {
  invoiceFilter: InvoiceListFilter;
  setInvoiceFilter: (filter: Partial<InvoiceListFilter>) => void;
  clearInvoiceFilter: () => void;
  fetchInvoices: () => void;
  createNewInvoice: (payload: CreateInvoicePayload) => Promise<{ success: boolean; message?: string }>;
  updateInvoice: (payload: CreateInvoicePayload) => Promise<{ success: boolean; message?: string }>;
  payInvoice: (id: string) => Promise<{ success: boolean; message?: string }>;
  unpayInvoice: (id: string) => Promise<{ success: boolean; message?: string }>;
  deleteInvoice: (id: string) => Promise<{ success: boolean; message?: string }>;
}

const errorMessage = (error: unknown, fallback: string) =>
  (error as { response?: { data?: { message?: string } } }).response?.data?.message || fallback;

export const useInvoiceStore = create<InvoiceDataState>((set, get) => ({
  invoiceFilter: restoreFilter(),

  setInvoiceFilter: (filter) => {
    const prev = get().invoiceFilter;
    const next = { ...prev, ...filter };
    const criteriaChanged = (['house_id', 'room_id', 'period', 'status', 'limit'] as const).some(key => next[key] !== prev[key]);
    if (criteriaChanged && filter.page === undefined) next.page = 1;
    if (next.house_id !== prev.house_id && filter.room_id === undefined) next.room_id = '';
    set({ invoiceFilter: next });
    writeStorage<PersistedFilter>(INVOICE_FILTERS_KEY, {
      house_id: next.house_id,
      room_id: next.room_id,
      period: next.period,
      status: next.status,
    });
    if (next.limit !== prev.limit) writeStorage(INVOICE_PAGE_SIZE_KEY, next.limit);
  },

  clearInvoiceFilter: () => {
    removeStorage(INVOICE_FILTERS_KEY);
    set(state => ({ invoiceFilter: { ...defaultFilter(), limit: state.invoiceFilter.limit } }));
  },

  fetchInvoices: refreshInvoiceQueries,

  createNewInvoice: async (payload: CreateInvoicePayload) => {
    try {
      await createInvoice(payload);
      refreshInvoiceQueries();
      return { success: true, message: 'Tạo hóa đơn thành công' };
    } catch (error: unknown) {
      return { success: false, message: errorMessage(error, 'Lỗi khi tạo hóa đơn') };
    }
  },

  updateInvoice: async (payload: CreateInvoicePayload) => {
    try {
      await createInvoice(payload);
      refreshInvoiceQueries();
      return { success: true, message: 'Cập nhật hóa đơn thành công' };
    } catch (error: unknown) {
      return { success: false, message: errorMessage(error, 'Lỗi khi cập nhật hóa đơn') };
    }
  },

  payInvoice: async (id: string) => {
    setInvoiceStatus(id, 'PAID');
    try {
      await payInvoice(id);
      return { success: true, message: 'Thanh toán hóa đơn thành công' };
    } catch (error: unknown) {
      return { success: false, message: errorMessage(error, 'Lỗi khi thanh toán') };
    } finally {
      refreshInvoiceQueries();
    }
  },

  unpayInvoice: async (id: string) => {
    setInvoiceStatus(id, 'UNPAID');
    try {
      await unpayInvoice(id);
      return { success: true, message: 'Đã hoàn tác thanh toán' };
    } catch (error: unknown) {
      return { success: false, message: errorMessage(error, 'Lỗi khi hoàn tác thanh toán') };
    } finally {
      refreshInvoiceQueries();
    }
  },

  deleteInvoice: async (id: string) => {
    try {
      await deleteInvoice(id);
      refreshInvoiceQueries();
      return { success: true, message: 'Xóa hóa đơn thành công' };
    } catch (error: unknown) {
      return { success: false, message: errorMessage(error, 'Lỗi khi xóa hóa đơn') };
    }
  },
}));
