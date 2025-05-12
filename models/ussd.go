package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// USSDSession represents a user's USSD session
type USSDSession struct {
	ID          int64     `json:"id"`
	SessionID   string    `json:"session_id"`
	PhoneNumber string    `json:"phone_number"`
	CurrentMenu string    `json:"current_menu"`
	Data        JSONMap   `json:"data"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// AccountInfo represents a user's banking account information
type AccountInfo struct {
	PhoneNumber   string  `json:"phone_number"`
	Name          string  `json:"name"`
	AccountNumber string  `json:"account_number"`
	Balance       float64 `json:"balance"`
	Currency      string  `json:"currency"`
	Status        string  `json:"status"`
}

// JSONMap is a helper type for storing JSON data
type JSONMap map[string]interface{}

// Value implements the driver.Valuer interface for JSONMap
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for JSONMap
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONMap)
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &j)
}

// TransactionType defines the type of financial transaction
type TransactionType string

const (
	TransactionTypeDeposit    TransactionType = "deposit"
	TransactionTypeWithdrawal TransactionType = "withdrawal"
	TransactionTypeTransfer   TransactionType = "transfer"
	TransactionTypeBillPay    TransactionType = "bill_payment"
	TransactionTypeAirtime    TransactionType = "airtime"
)

// TransactionStatus defines the status of a transaction
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusCompleted TransactionStatus = "completed"
	TransactionStatusFailed    TransactionStatus = "failed"
	TransactionStatusCancelled TransactionStatus = "cancelled"
)

// Transaction represents a financial transaction in the system
type Transaction struct {
	ID          int64             `json:"id"`
	ReferenceID string            `json:"reference_id"`
	Type        TransactionType   `json:"transaction_type"`
	SenderID    *int64            `json:"sender_id,omitempty"`
	RecipientID *int64            `json:"recipient_id,omitempty"`
	Amount      float64           `json:"amount"`
	Fee         float64           `json:"fee"`
	Status      TransactionStatus `json:"status"`
	Description string            `json:"description,omitempty"`
	Metadata    JSONMap           `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`

	// Additional fields from joins (not stored in DB)
	SenderPhone    string `json:"sender_phone,omitempty"`
	RecipientPhone string `json:"recipient_phone,omitempty"`
}

// User represents a system user (wallet holder)
type User struct {
	ID          int64     `json:"id"`
	PhoneNumber string    `json:"phone_number"`
	Pin         string    `json:"-"` // Never expose PIN in JSON
	FirstName   string    `json:"first_name,omitempty"`
	LastName    string    `json:"last_name,omitempty"`
	Balance     float64   `json:"balance"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AuditLog represents an audit entry for system actions
type AuditLog struct {
	ID         int64     `json:"id"`
	UserID     *int64    `json:"user_id,omitempty"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type,omitempty"`
	EntityID   string    `json:"entity_id,omitempty"`
	OldValue   JSONMap   `json:"old_value,omitempty"`
	NewValue   JSONMap   `json:"new_value,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
	UserAgent  string    `json:"user_agent,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// USSDMenu represents a configurable USSD menu
type USSDMenu struct {
	ID           int64     `json:"id"`
	MenuKey      string    `json:"menu_key"`
	Title        string    `json:"title"`
	Options      JSONMap   `json:"options"`
	ParentMenu   string    `json:"parent_menu,omitempty"`
	RequiresAuth bool      `json:"requires_auth"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
