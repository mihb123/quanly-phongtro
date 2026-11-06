package expense

import (
	"net/http"

	"github.com/mihb123/quanly-phongtro/internal/handler/httpx"
	expensesvc "github.com/mihb123/quanly-phongtro/internal/service/expense"
)

type ExpenseHandler struct {
	service expensesvc.ExpenseService
}

func NewExpenseHandler(service expensesvc.ExpenseService) *ExpenseHandler {
	return &ExpenseHandler{service: service}
}

func (h *ExpenseHandler) Create(w http.ResponseWriter, r *http.Request) {
	managerID, ok := httpx.GetManagerID(r, w)
	if !ok {
		return
	}
	var req createExpenseRequest
	if !httpx.Bind(w, r, &req) {
		return
	}
	expense, err := h.service.Create(r.Context(), managerID, req.input())
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toExpenseResponse(expense), "Đã thêm khoản chi")
}

func (h *ExpenseHandler) List(w http.ResponseWriter, r *http.Request) {
	managerID, ok := httpx.GetManagerID(r, w)
	if !ok {
		return
	}
	var req listExpensesRequest
	if !httpx.Bind(w, r, &req) {
		return
	}
	result, err := h.service.List(r.Context(), managerID, req.filter())
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	items := make([]expenseResponse, 0, len(result.Items))
	for i := range result.Items {
		items = append(items, toExpenseResponse(&result.Items[i]))
	}
	httpx.WriteJSON(w, http.StatusOK, expenseListResponse{Items: items, Total: result.Total, Page: req.page(), Limit: req.limit()}, "")
}

func (h *ExpenseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	managerID, ok := httpx.GetManagerID(r, w)
	if !ok {
		return
	}
	var req expenseIDRequest
	if !httpx.Bind(w, r, &req) {
		return
	}
	if err := h.service.Delete(r.Context(), managerID, req.ID); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, nil, "Đã xoá khoản chi")
}
