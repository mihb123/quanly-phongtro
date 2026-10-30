package main

import (
	"cmp"
	"regexp"
	"slices"
	"time"

	"github.com/mihb123/quanly-phongtro/plugin-sdk/qlpt"
)

const (
	settingMinAmount       = "min_amount"
	settingRemindAfterDays = "remind_after_days"
	storageReminders       = "reminders"
)

var periodPattern = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

type debtItem struct {
	qlpt.Invoice
	RemindedAt *time.Time `json:"reminded_at"`
	NeedRemind bool       `json:"need_remind"`
}

type summary struct {
	Items       []debtItem `json:"items"`
	Count       int        `json:"count"`
	NeedRemind  int        `json:"need_remind"`
	TotalAmount float64    `json:"total_amount"`
}

func main() {
	app := qlpt.New()
	app.Get("/summary", getSummary)
	app.Post("/reminders/{invoice_id}", markReminded)
	app.Delete("/reminders/{invoice_id}", clearReminder)
	app.Run()
}

func getSummary(c *qlpt.Context) error {
	period := c.Query("period")
	if period != "" && !periodPattern.MatchString(period) {
		return qlpt.BadRequest("Kỳ phải có dạng YYYY-MM")
	}
	settings, err := c.Settings()
	if err != nil {
		return err
	}
	var periods []string
	if period != "" {
		periods = append(periods, period)
	}
	invoices, err := c.Invoices().ListOutstanding(periods...)
	if err != nil {
		return err
	}
	reminders, err := loadReminders(c)
	if err != nil {
		return err
	}

	minAmount := settings.Number(settingMinAmount)
	remindAfter := time.Duration(settings.Number(settingRemindAfterDays)*24) * time.Hour
	now := time.Now()
	res := summary{Items: []debtItem{}}
	for _, invoice := range invoices {
		if invoice.TotalAmount < minAmount {
			continue
		}
		item := debtItem{Invoice: invoice, NeedRemind: true}
		if at, ok := reminders[invoice.ID]; ok {
			item.RemindedAt = &at
			item.NeedRemind = now.Sub(at) >= remindAfter
		}
		if item.NeedRemind {
			res.NeedRemind++
		}
		res.TotalAmount += invoice.TotalAmount
		res.Items = append(res.Items, item)
	}
	res.Count = len(res.Items)
	slices.SortFunc(res.Items, func(a, b debtItem) int {
		if a.NeedRemind != b.NeedRemind {
			if a.NeedRemind {
				return -1
			}
			return 1
		}
		return cmp.Or(cmp.Compare(a.Period, b.Period), cmp.Compare(a.HouseName, b.HouseName), cmp.Compare(a.RoomName, b.RoomName))
	})
	return c.JSON(200, res)
}

func markReminded(c *qlpt.Context) error {
	invoiceID := c.Param("invoice_id")
	invoices, err := c.Invoices().ListOutstanding()
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(invoices, func(i qlpt.Invoice) bool { return i.ID == invoiceID }) {
		return qlpt.NotFound("Không tìm thấy hóa đơn chưa thanh toán")
	}
	reminders, err := loadReminders(c)
	if err != nil {
		return err
	}
	for id := range reminders {
		if !slices.ContainsFunc(invoices, func(i qlpt.Invoice) bool { return i.ID == id }) {
			delete(reminders, id)
		}
	}
	now := time.Now().UTC()
	reminders[invoiceID] = now
	if err := c.Storage().Set(storageReminders, reminders); err != nil {
		return err
	}
	return c.Message(200, "Đã đánh dấu đã nhắc", map[string]any{"invoice_id": invoiceID, "reminded_at": now})
}

func clearReminder(c *qlpt.Context) error {
	reminders, err := loadReminders(c)
	if err != nil {
		return err
	}
	delete(reminders, c.Param("invoice_id"))
	if err := c.Storage().Set(storageReminders, reminders); err != nil {
		return err
	}
	return c.Message(200, "Đã bỏ đánh dấu", nil)
}

func loadReminders(c *qlpt.Context) (map[string]time.Time, error) {
	reminders := map[string]time.Time{}
	if _, err := c.Storage().Get(storageReminders, &reminders); err != nil {
		return nil, err
	}
	return reminders, nil
}
