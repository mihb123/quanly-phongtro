package expense

import (
	"cmp"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/handler/httpx"
	"github.com/mihb123/quanly-phongtro/internal/model"
	expensesvc "github.com/mihb123/quanly-phongtro/internal/service/expense"
)

var businessTZ = time.FixedZone("ICT", 7*60*60)

type createExpenseRequest struct {
	HouseID   string `json:"house_id" label:"Nhà" validate:"required,uuid"`
	ReceiptNo string `json:"receipt_no" label:"Mã chứng từ" validate:"omitempty,max=50"`
	Amount    int64  `json:"amount" label:"Số tiền" validate:"required,gt=0,lte=10000000000"`
	Note      string `json:"note" label:"Ghi chú" validate:"max=255"`
	SpentAt   string `json:"spent_at" label:"Thời điểm chi" validate:"required"`

	spentAt time.Time
}

type listExpensesRequest struct {
	HouseID string `query:"house_id" json:"-" label:"Nhà" validate:"omitempty,uuid"`
	From    string `query:"from" json:"-" label:"Từ ngày" validate:"omitempty,datetime=2006-01-02"`
	To      string `query:"to" json:"-" label:"Đến ngày" validate:"omitempty,datetime=2006-01-02"`
	Page    int    `query:"page" json:"-" label:"Trang" validate:"omitempty,min=1"`
	Limit   int    `query:"limit" json:"-" label:"Số dòng mỗi trang" validate:"omitempty,oneof=25 50 100"`
}

type expenseIDRequest struct {
	ID string `path:"id" json:"-" label:"Khoản chi" validate:"required,uuid"`
}

type expenseResponse struct {
	ID        string    `json:"id"`
	HouseID   string    `json:"house_id"`
	ReceiptNo string    `json:"receipt_no"`
	Amount    int64     `json:"amount"`
	Note      string    `json:"note"`
	SpentAt   time.Time `json:"spent_at"`
}

type expenseListResponse struct {
	Items []expenseResponse `json:"items"`
	Total int               `json:"total"`
	Page  int               `json:"page"`
	Limit int               `json:"limit"`
}

func (q *createExpenseRequest) Validate(errs *httpx.FieldErrors) {
	spentAt, err := time.Parse(time.RFC3339, q.SpentAt)
	if err != nil {
		errs.Add("spent_at", "Thời điểm chi không đúng định dạng")
		return
	}
	if spentAt.After(time.Now().Add(24 * time.Hour)) {
		errs.Add("spent_at", "Thời điểm chi không được ở tương lai")
		return
	}
	q.spentAt = spentAt.UTC()
}

func (q *createExpenseRequest) input() expensesvc.CreateInput {
	return expensesvc.CreateInput{
		HouseID:   q.HouseID,
		ReceiptNo: q.ReceiptNo,
		Amount:    q.Amount,
		Note:      q.Note,
		SpentAt:   q.spentAt,
	}
}

func (q *listExpensesRequest) Validate(errs *httpx.FieldErrors) {
	if q.From != "" && q.To != "" && q.To < q.From {
		errs.Add("to", "Đến ngày không được trước Từ ngày")
	}
}

func (q *listExpensesRequest) page() int  { return max(q.Page, 1) }
func (q *listExpensesRequest) limit() int { return cmp.Or(q.Limit, 25) }

func (q *listExpensesRequest) filter() model.ExpenseFilter {
	from, to := httpx.DayRange(q.From, q.To, businessTZ)
	return model.ExpenseFilter{
		HouseID: q.HouseID,
		From:    from,
		To:      to,
		Limit:   q.limit(),
		Offset:  (q.page() - 1) * q.limit(),
	}
}

func toExpenseResponse(e *model.Expense) expenseResponse {
	return expenseResponse{
		ID:        e.ID,
		HouseID:   e.HouseID,
		ReceiptNo: e.ReceiptNo,
		Amount:    e.Amount,
		Note:      e.Note,
		SpentAt:   e.SpentAt,
	}
}
