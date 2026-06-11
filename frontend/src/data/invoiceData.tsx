import { create } from 'zustand';
import {
  type Invoice,
  type InvoiceFilter,
  getInvoices,
  createInvoice,
  type CreateInvoicePayload,
  payInvoice,
  unpayInvoice,
  deleteInvoice,
} from '../api/invoice';


interface InvoiceDataState {
  invoices: Invoice[];
  isLoading: boolean;
  invoiceFilter: InvoiceFilter;
  setInvoiceFilter: (filter: Partial<InvoiceFilter>) => void;
  fetchInvoices: () => Promise<void>;
  createNewInvoice: (payload: CreateInvoicePayload) => Promise<{ success: boolean; message?: string }>;
  updateInvoice: (payload: CreateInvoicePayload) => Promise<{ success: boolean; message?: string }>;
  payInvoice: (id: string) => Promise<{ success: boolean; message?: string }>;
  unpayInvoice: (id: string) => Promise<{ success: boolean; message?: string }>;
  deleteInvoice: (id: string) => Promise<{ success: boolean; message?: string }>;
}

const currentMonth = `${new Date().getFullYear()}-${String(new Date().getMonth() + 1).padStart(2, '0')}`;

export const useInvoiceStore = create<InvoiceDataState>((set, get) => ({
  invoices: [],
  isLoading: false,
  invoiceFilter: {
    page: 1,
    limit: 20,
    period: currentMonth,
  },

  setInvoiceFilter: (filter) => {
    set((state) => ({
      invoiceFilter: { ...state.invoiceFilter, ...filter },
    }));
    // Automatically fetch when filters change
    get().fetchInvoices();
  },

  fetchInvoices: async () => {
    set({ isLoading: true });
    try {
      const data = await getInvoices(get().invoiceFilter);
      set({ invoices: data || [] });
    } catch (error: unknown) {
      const err = error as { response?: { data?: { message?: string } } };
      console.error(err.response?.data?.message || 'Lỗi khi tải danh sách hóa đơn');
    } finally {
      set({ isLoading: false });
    }
  },

  createNewInvoice: async (payload: CreateInvoicePayload) => {
    try {
      await createInvoice(payload);
      await get().fetchInvoices();
      return { success: true, message: 'Tạo hóa đơn thành công' };
    } catch (error: unknown) {
      const err = error as { response?: { data?: { message?: string } } };
      return { success: false, message: err.response?.data?.message || 'Lỗi khi tạo hóa đơn' };
    }
  },

  updateInvoice: async (payload: CreateInvoicePayload) => {
    try {
      await createInvoice(payload);
      await get().fetchInvoices();
      return { success: true, message: 'Cập nhật hóa đơn thành công' };
    } catch (error: unknown) {
      const err = error as { response?: { data?: { message?: string } } };
      return { success: false, message: err.response?.data?.message || 'Lỗi khi cập nhật hóa đơn' };
    }
  },

  payInvoice: async (id: string) => {
    const { invoices } = get();
    const originalInvoices = [...invoices];

    // Optimistic update
    set({
      invoices: invoices.map((inv) =>
        inv.id === id ? { ...inv, status: 'PAID' } : inv
      ),
    });

    try {
      await payInvoice(id);
      return { success: true, message: 'Thanh toán hóa đơn thành công' };
    } catch (error: unknown) {
      const err = error as { response?: { data?: { message?: string } } };
      // Rollback
      set({ invoices: originalInvoices });
      return { success: false, message: err.response?.data?.message || 'Lỗi khi thanh toán' };
    }
  },

  unpayInvoice: async (id: string) => {
    const { invoices } = get();
    const originalInvoices = [...invoices];

    // Optimistic update
    set({
      invoices: invoices.map((inv) =>
        inv.id === id ? { ...inv, status: 'UNPAID' } : inv
      ),
    });

    try {
      await unpayInvoice(id);
      return { success: true, message: 'Đã hoàn tác thanh toán' };
    } catch (error: unknown) {
      const err = error as { response?: { data?: { message?: string } } };
      // Rollback
      set({ invoices: originalInvoices });
      return { success: false, message: err.response?.data?.message || 'Lỗi khi hoàn tác thanh toán' };
    }
  },

  deleteInvoice: async (id: string) => {
    try {
      await deleteInvoice(id);
      await get().fetchInvoices();
      return { success: true, message: 'Xóa hóa đơn thành công' };
    } catch (error: unknown) {
      const err = error as { response?: { data?: { message?: string } } };
      return { success: false, message: err.response?.data?.message || 'Lỗi khi xóa hóa đơn' };
    }
  },
}));
