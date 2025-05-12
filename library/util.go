package library

import (
	"fmt"
	"github.com/google/uuid"
	"time"
)

// ParsePhoneNumber normalizes a phone number to standard format
func ParsePhoneNumber(phone string) (string, error) {
	// This is a simplified implementation
	// In production, consider using a phone number validation library

	// Remove any non-numeric characters
	// For a real implementation, consider using libphonenumber

	if len(phone) < 12 {
		return "", fmt.Errorf("phone number too short")
	}

	// This is placeholder logic - customize based on your requirements
	// Example: If number doesn't start with country code, add it
	if phone[0] != '+' {
		// Add country code (e.g., +1 for US)
		phone = "+254" + phone
	}

	return phone, nil
}

// GenerateTransactionID creates a unique transaction ID
func GenerateTransactionID(prefix string) string {
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	randomPart, _ := uuid.NewV7()

	return fmt.Sprintf("%s%d%s", prefix, timestamp, randomPart)
}

// FormatCurrency formats a number as currency with the given symbol
func FormatCurrency(amount float64, currency string) string {
	return fmt.Sprintf("%.2f %s", amount, currency)
}
