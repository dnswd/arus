package arus

type Statement struct {
	ID     string `json:"id,omitempty"`
	BankID string `json:"bank_id,omitempty"`
	Amount int64  `json:"amount,omitempty"`
}
