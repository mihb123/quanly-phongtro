package qlpt

import (
	"encoding/json"
	"fmt"
)

type hostResult struct {
	OK     bool            `json:"ok"`
	Data   json.RawMessage `json:"data"`
	Error  string          `json:"error"`
	Status int             `json:"status"`
}

func call(op string, params, out any) error {
	payload, err := json.Marshal(struct {
		Op     string `json:"op"`
		Params any    `json:"params,omitempty"`
	}{op, params})
	if err != nil {
		return err
	}
	var res hostResult
	if err := json.Unmarshal(transport(payload), &res); err != nil {
		return fmt.Errorf("qlpt: host trả kết quả không hợp lệ cho %s: %w", op, err)
	}
	if !res.OK {
		return &Error{Status: res.Status, Message: res.Error}
	}
	if out == nil || len(res.Data) == 0 || string(res.Data) == "null" {
		return nil
	}
	return json.Unmarshal(res.Data, out)
}

type Values map[string]any

func (v Values) String(key string) string {
	s, _ := v[key].(string)
	return s
}

func (v Values) Bool(key string) bool {
	b, _ := v[key].(bool)
	return b
}

func (v Values) Number(key string) float64 {
	n, _ := v[key].(float64)
	return n
}

func (c *Context) Settings() (Values, error) {
	values := Values{}
	if err := call("config.get", nil, &values); err != nil {
		return nil, err
	}
	return values, nil
}

type Storage struct{}

func (c *Context) Storage() Storage { return Storage{} }

func (Storage) Get(key string, out any) (bool, error) {
	var raw json.RawMessage
	if err := call("storage.get", map[string]string{"key": key}, &raw); err != nil {
		return false, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return false, nil
	}
	return true, json.Unmarshal(raw, out)
}

func (Storage) Set(key string, value any) error {
	return call("storage.set", map[string]any{"key": key, "value": value}, nil)
}

func (Storage) Delete(key string) error {
	return call("storage.set", map[string]any{"key": key, "value": nil}, nil)
}

type Invoice struct {
	ID          string   `json:"id"`
	RoomID      string   `json:"room_id"`
	RoomName    string   `json:"room_name"`
	HouseID     string   `json:"house_id"`
	HouseName   string   `json:"house_name"`
	Period      string   `json:"period"`
	TotalAmount float64  `json:"total_amount"`
	Status      string   `json:"status"`
	TenantNames []string `json:"tenant_names"`
}

type Payment struct {
	InvoiceID      string `json:"invoice_id"`
	Amount         int    `json:"amount"`
	AccountRef     string `json:"account_ref"`
	TransactionRef string `json:"transaction_ref"`
	MatchingMethod string `json:"matching_method,omitempty"`
	PaymentMethod  string `json:"payment_method,omitempty"`
	RawPayload     string `json:"raw_payload,omitempty"`
}

type Invoices struct{}

func (c *Context) Invoices() Invoices { return Invoices{} }

func (Invoices) ListOutstanding(periods ...string) ([]Invoice, error) {
	var invoices []Invoice
	err := call("invoices.list_outstanding", map[string]any{"periods": periods}, &invoices)
	return invoices, err
}

func (Invoices) ListPaid(periods ...string) ([]Invoice, error) {
	var invoices []Invoice
	err := call("invoices.list_paid", map[string]any{"periods": periods}, &invoices)
	return invoices, err
}

func (Invoices) RecordedPayment(accountRef, transactionRef string) (invoiceID string, recorded bool, err error) {
	var res struct {
		InvoiceID string `json:"invoice_id"`
		Recorded  bool   `json:"recorded"`
	}
	err = call("invoices.recorded_payment", map[string]string{"account_ref": accountRef, "transaction_ref": transactionRef}, &res)
	return res.InvoiceID, res.Recorded, err
}

func (Invoices) MarkPaid(p Payment) error {
	return call("invoices.mark_paid", p, nil)
}
