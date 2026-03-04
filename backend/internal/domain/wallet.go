package domain

import "time"

// Transaction type constants.
const (
	TxTypeRecharge   = "recharge"
	TxTypePurchase   = "purchase"
	TxTypeRefund     = "refund"
	TxTypeAdjustment = "adjustment"
)

// Wallet represents a user's JPY wallet.
type Wallet struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Balance   int64     `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WalletTransaction represents a single wallet ledger entry.
type WalletTransaction struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	Type         string    `json:"type"`
	Amount       int64     `json:"amount"`
	BalanceAfter int64     `json:"balance_after"`
	Description  string    `json:"description"`
	RelatedOrder *string   `json:"related_order,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}
