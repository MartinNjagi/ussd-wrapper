package controller

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"ussd-wrapper/library/logger"
	"ussd-wrapper/models"
)

// getAccountInfo fetches account information for a phone number
func (ctl *Controller) getAccountInfo(ctx context.Context, phoneNumber string) (*models.AccountInfo, error) {
	// In production, this would call your backend API or database
	// For this example, we return mock data

	logger.WithCtx(ctx).Infof("Fetching account info for: %s", phoneNumber)

	// Mock account info
	accountInfo := &models.AccountInfo{
		PhoneNumber:   phoneNumber,
		Name:          "John Doe",
		AccountNumber: phoneNumber[len(phoneNumber)-9:],
		Balance:       1000.50,
		Currency:      "USD",
		Status:        "Active",
	}

	return accountInfo, nil
}

// handleBalanceMenu processes balance checking
func (ctl *Controller) handleBalanceMenu(ctx context.Context, session *models.USSDSession, input string) (string, error) {
	// If this is the first time in the balance menu
	if input == "" {
		// Fetch account balance
		accountInfo, err := ctl.getAccountInfo(ctx, session.PhoneNumber)
		if err != nil {
			return "", err
		}

		// Reset menu state
		session.CurrentMenu = "main"
		if err := ctl.saveSession(ctx, session); err != nil {
			return "", err
		}

		return fmt.Sprintf("END Your balance is: %.2f %s",
			accountInfo.Balance, accountInfo.Currency), nil
	}

	// Should not reach here in normal flow
	return ctl.getMainMenu()
}

// handleTransferMenu processes money transfer
func (ctl *Controller) handleTransferMenu(ctx context.Context, session *models.USSDSession, currentInput string, allInputs []string) (string, error) {
	transferStep, ok := session.Data["transfer_step"].(string)
	if !ok {
		transferStep = "recipient"
		session.Data["transfer_step"] = transferStep
	}

	switch transferStep {
	case "recipient":
		// Validate phone number
		if !ctl.isValidPhoneNumber(currentInput) {
			return "CON Invalid phone number. Please enter a valid phone number:", nil
		}

		// Store recipient
		session.Data["transfer_recipient"] = currentInput
		session.Data["transfer_step"] = "amount"
		if err := ctl.saveSession(ctx, session); err != nil {
			return "", err
		}

		return "CON Enter amount to transfer:", nil

	case "amount":
		amount, err := strconv.ParseFloat(currentInput, 64)
		if err != nil || amount <= 0 {
			return "CON Invalid amount. Please enter a valid amount:", nil
		}

		// Check if user has sufficient balance
		accountInfo, err := ctl.getAccountInfo(ctx, session.PhoneNumber)
		if err != nil {
			return "", err
		}

		if amount > accountInfo.Balance {
			return "END Insufficient funds. Your current balance is: " +
				fmt.Sprintf("%.2f %s", accountInfo.Balance, accountInfo.Currency), nil
		}

		// Store amount
		session.Data["transfer_amount"] = amount
		session.Data["transfer_step"] = "confirm"
		if err := ctl.saveSession(ctx, session); err != nil {
			return "", err
		}

		recipient := session.Data["transfer_recipient"].(string)
		return fmt.Sprintf("CON Confirm transfer of %.2f %s to %s?\n1. Confirm\n2. Cancel",
			amount, accountInfo.Currency, recipient), nil

	case "confirm":
		if currentInput == "1" {
			// Process transfer
			recipient := session.Data["transfer_recipient"].(string)
			amount := session.Data["transfer_amount"].(float64)

			// In production, call your payment API here
			logger.WithCtx(ctx).Infof("Processing transfer: %f to %s from %s",
				amount, recipient, session.PhoneNumber)

			// Get recipient name (mock)
			recipientName := "Jane Doe"

			// Reset menu state
			session.CurrentMenu = "main"
			delete(session.Data, "transfer_step")
			delete(session.Data, "transfer_recipient")
			delete(session.Data, "transfer_amount")
			if err := ctl.saveSession(ctx, session); err != nil {
				return "", err
			}

			return fmt.Sprintf("END Transfer of %.2f to %s (%s) was successful.",
				amount, recipient, recipientName), nil

		} else if currentInput == "2" {
			// Cancel transfer
			session.CurrentMenu = "main"
			delete(session.Data, "transfer_step")
			delete(session.Data, "transfer_recipient")
			delete(session.Data, "transfer_amount")
			if err := ctl.saveSession(ctx, session); err != nil {
				return "", err
			}

			return "END Transfer cancelled.", nil
		} else {
			return "CON Invalid option.\nConfirm transfer?\n1. Confirm\n2. Cancel", nil
		}
	}

	// Should not reach here in normal flow
	return ctl.getMainMenu()
}

// handleDepositMenu processes deposit requests
func (ctl *Controller) handleDepositMenu(ctx context.Context, session *models.USSDSession, currentInput string, allInputs []string) (string, error) {
	depositStep, ok := session.Data["deposit_step"].(string)
	if !ok {
		depositStep = "amount"
		session.Data["deposit_step"] = depositStep
	}

	switch depositStep {
	case "amount":
		amount, err := strconv.ParseFloat(currentInput, 64)
		if err != nil || amount <= 0 {
			return "CON Invalid amount. Please enter a valid amount:", nil
		}

		// Store amount
		session.Data["deposit_amount"] = amount
		session.Data["deposit_step"] = "method"
		if err := ctl.saveSession(ctx, session); err != nil {
			return "", err
		}

		return `CON Select deposit method:
1. Mobile Money
2. Bank Transfer
3. Agent
4. Cancel`, nil

	case "method":
		method := ""
		switch currentInput {
		case "1":
			method = "Mobile Money"
		case "2":
			method = "Bank Transfer"
		case "3":
			method = "Agent"
		case "4":
			// Cancel operation
			session.CurrentMenu = "main"
			delete(session.Data, "deposit_step")
			delete(session.Data, "deposit_amount")
			if err := ctl.saveSession(ctx, session); err != nil {
				return "", err
			}
			return "END Deposit cancelled.", nil
		default:
			return `CON Invalid option. Select deposit method:
1. Mobile Money
2. Bank Transfer
3. Agent
4. Cancel`, nil
		}

		// Process deposit
		amount := session.Data["deposit_amount"].(float64)
		accountInfo, err := ctl.getAccountInfo(ctx, session.PhoneNumber)
		if err != nil {
			return "", err
		}

		// In production, generate actual instructions based on the method
		instructions := fmt.Sprintf("To deposit %.2f %s via %s, please follow these steps:\n",
			amount, accountInfo.Currency, method)

		switch method {
		case "Mobile Money":
			instructions += "1. Dial *100#\n2. Select Send Money\n3. Enter account: " + accountInfo.AccountNumber
		case "Bank Transfer":
			instructions += "Bank: Example Bank\nAccount: " + accountInfo.AccountNumber
		case "Agent":
			instructions += "Visit any of our agents with your ID and phone number."
		}

		// Reset menu state
		session.CurrentMenu = "main"
		delete(session.Data, "deposit_step")
		delete(session.Data, "deposit_amount")
		if err := ctl.saveSession(ctx, session); err != nil {
			return "", err
		}

		return "END " + instructions, nil
	}

	// Should not reach here in normal flow
	return ctl.getMainMenu()
}

// handleWithdrawMenu processes withdrawal requests
func (ctl *Controller) handleWithdrawMenu(ctx context.Context, session *models.USSDSession, currentInput string, allInputs []string) (string, error) {
	withdrawStep, ok := session.Data["withdraw_step"].(string)
	if !ok {
		withdrawStep = "amount"
		session.Data["withdraw_step"] = withdrawStep
	}

	switch withdrawStep {
	case "amount":
		amount, err := strconv.ParseFloat(currentInput, 64)
		if err != nil || amount <= 0 {
			return "CON Invalid amount. Please enter a valid amount:", nil
		}

		// Check balance
		accountInfo, err := ctl.getAccountInfo(ctx, session.PhoneNumber)
		if err != nil {
			return "", err
		}

		if amount > accountInfo.Balance {
			return "END Insufficient funds. Your current balance is: " +
				fmt.Sprintf("%.2f %s", accountInfo.Balance, accountInfo.Currency), nil
		}

		// Store amount
		session.Data["withdraw_amount"] = amount
		session.Data["withdraw_step"] = "method"
		if err := ctl.saveSession(ctx, session); err != nil {
			return "", err
		}

		return `CON Select withdrawal method:
1. Mobile Money
2. Agent
3. ATM
4. Cancel`, nil

	case "method":
		method := ""
		switch currentInput {
		case "1":
			method = "Mobile Money"
		case "2":
			method = "Agent"
		case "3":
			method = "ATM"
		case "4":
			// Cancel operation
			session.CurrentMenu = "main"
			delete(session.Data, "withdraw_step")
			delete(session.Data, "withdraw_amount")
			if err := ctl.saveSession(ctx, session); err != nil {
				return "", err
			}
			return "END Withdrawal cancelled.", nil
		default:
			return `CON Invalid option. Select withdrawal method:
1. Mobile Money
2. Agent
3. ATM
4. Cancel`, nil
		}

		// Process withdrawal
		amount := session.Data["withdraw_amount"].(float64)

		// In production, this would integrate with your payment system
		// For now, generate a mock transaction ID
		transactionID := fmt.Sprintf("WD%d", time.Now().Unix())

		logger.WithCtx(ctx).Infof("Processing withdrawal: %f via %s for %s, txn: %s",
			amount, method, session.PhoneNumber, transactionID)

		// Reset menu state
		session.CurrentMenu = "main"
		delete(session.Data, "withdraw_step")
		delete(session.Data, "withdraw_amount")
		if err := ctl.saveSession(ctx, session); err != nil {
			return "", err
		}

		response := fmt.Sprintf("END Withdrawal of %.2f via %s initiated.\nTransaction ID: %s",
			amount, method, transactionID)

		if method == "ATM" {
			response += "\nUse the ATM code: " + transactionID[2:]
		}

		return response, nil
	}

	// Should not reach here in normal flow
	return ctl.getMainMenu()
}

// isValidPhoneNumber validates a phone number format
func (ctl *Controller) isValidPhoneNumber(phone string) bool {
	// Basic validation - you would enhance this based on your requirements
	// Example: Must be at least 9 digits
	return len(strings.ReplaceAll(phone, " ", "")) >= 9
}
