package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/labstack/echo/v4"
	"net/http"
	"strings"
	"time"
	"ussd-wrapper/library"
	"ussd-wrapper/library/logger"
	"ussd-wrapper/models"
)

func (ctl *Controller) HandleUSSD1(c echo.Context) error {
	ctx := c.Request().Context()

	// Extract USSD data (assuming JSON for now)
	var payload map[string]interface{}
	if err := c.Bind(&payload); err != nil {
		library.LogWithTrace(ctx).Errorf("Failed to parse payload: %v", err)
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}

	library.LogWithTrace(ctx).Infof("Received USSD request: %+v", payload)

	// Process...
	return c.String(http.StatusOK, "CON Welcome to your USSD service")
}

func (ctl *Controller) HandleUSSD(c echo.Context) error {
	ctx := c.Request().Context()

	// Redis example
	traceID := library.GetTraceID(ctx)
	_ = library.SetRedisKeyWithExpiry(ctx, ctl.redis, "trace:"+traceID, "some_value", 0)

	return c.String(http.StatusOK, "CON USSD ready with context.")
}

// HandleUSSD processes all USSD requests
func (ctl *Controller) HandleUSSD3(c echo.Context) error {
	sessionID := c.FormValue("sessionID")
	serviceCode := c.FormValue("serviceCode")
	phoneNumber := c.FormValue("phoneNumber")
	text := c.FormValue("text")

	ctx := c.Request().Context()
	ctx, span := ctl.tracer.Start(ctx, "ussd_session")
	defer span.End()

	// ✅ Use trace-aware logger
	logger.WithCtx(ctx).Infof("USSD request received - sessionID: %s,serviceCode: %s, phoneNumber: %s, text: %s", sessionID, serviceCode, phoneNumber, text)

	// Determine session
	session, isNew, err := ctl.getOrCreateSession(ctx, sessionID, phoneNumber)
	if err != nil {
		logger.WithCtx(ctx).Errorf("Session error: %v", err)
		return ctl.handleError(c, err)
	}

	response, err := ctl.processUSSDInput(ctx, session, text, isNew)
	if err != nil {
		logger.WithCtx(ctx).Errorf("Processing error: %v", err)
		return ctl.handleError(c, err)
	}

	logger.WithCtx(ctx).Infof("Responding with: %s", response)
	return c.String(http.StatusOK, response)
}

// processUSSDInput handles the USSD menu navigation logic
func (ctl *Controller) processUSSDInput(ctx context.Context, session *models.USSDSession,
	text string, isNew bool) (string, error) {
	if isNew {
		// New session, display main menu
		return ctl.getMainMenu()
	}

	// Parse the input sequence to determine current navigation state
	inputs := strings.Split(text, "*")
	currentInput := ""
	if len(inputs) > 0 {
		currentInput = inputs[len(inputs)-1]
	}

	// Route to appropriate menu handler based on session state
	switch session.CurrentMenu {
	case "main":
		return ctl.handleMainMenu(ctx, session, currentInput)
	case "balance":
		return ctl.handleBalanceMenu(ctx, session, currentInput)
	case "transfer":
		return ctl.handleTransferMenu(ctx, session, currentInput, inputs)
	case "deposit":
		return ctl.handleDepositMenu(ctx, session, currentInput, inputs)
	case "withdraw":
		return ctl.handleWithdrawMenu(ctx, session, currentInput, inputs)
	default:
		return ctl.getMainMenu()
	}
}

// getOrCreateSession retrieves an existing session or creates a new one
func (ctl *Controller) getOrCreateSession(ctx context.Context, sessionID, phoneNumber string) (*models.USSDSession, bool, error) {
	// Try to get existing session from Redis
	sessionKey := fmt.Sprintf("ussd:session:%s", sessionID)
	sessionData, err := library.GetRedisKey(ctl.redis, sessionKey)

	if err != nil && err != redis.Nil {
		logger.WithCtx(ctx).Errorf("Redis error while fetching session: %v", err)
		return nil, false, errors.New("service temporarily unavailable")
	}

	// Session exists, unmarshal and return
	if err == nil {
		var session models.USSDSession
		if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
			logger.WithCtx(ctx).Errorf("Failed to unmarshal session: %v", err)
			return nil, false, errors.New("session corruption detected")
		}
		return &session, false, nil
	}

	// Create new session
	id := time.Now().Nanosecond()

	session := &models.USSDSession{
		ID:          int64(id),
		SessionID:   sessionID,
		PhoneNumber: phoneNumber,
		CurrentMenu: "main",
		Data:        make(map[string]interface{}),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save to Redis
	if err := ctl.saveSession(ctx, session); err != nil {
		return nil, false, err
	}

	return session, true, nil
}

// saveSession persists session data to Redis
func (ctl *Controller) saveSession(ctx context.Context, session *models.USSDSession) error {
	sessionKey := fmt.Sprintf("ussd:session:%s", session.ID)
	session.UpdatedAt = time.Now()

	// Serialize session
	sessionJSON, err := json.Marshal(session)
	if err != nil {
		logger.WithCtx(ctx).Errorf("Failed to marshal session: %v", err)
		return errors.New("failed to save session")
	}

	// Store with expiry (30 minutes is typical for USSD sessions)
	err = library.SetRedisKeyWithExpiry(ctx, ctl.redis, sessionKey, string(sessionJSON), 30*60)
	if err != nil {
		logger.WithCtx(ctx).Errorf("Redis error while saving session: %v", err)
		return errors.New("failed to save session state")
	}

	return nil
}

// handleError returns an appropriate error response to the user
func (ctl *Controller) handleError(c echo.Context, err error) error {
	errorMsg := "END An error occurred. Please try again later."
	if err != nil && err.Error() != "" {
		errorMsg = fmt.Sprintf("END Error: %s", err.Error())
	}
	return c.String(200, errorMsg)
}

// getMainMenu returns the main USSD menu
func (ctl *Controller) getMainMenu() (string, error) {
	menu := `CON Welcome to Mobile Banking
1. Check Balance
2. Transfer Money
3. Deposit
4. Withdraw
5. Account Info
6. Exit`
	return menu, nil
}

// handleMainMenu processes input from the main menu
func (ctl *Controller) handleMainMenu(ctx context.Context, session *models.USSDSession, input string) (string, error) {
	switch input {
	case "1":
		session.CurrentMenu = "balance"
		if err := ctl.saveSession(ctx, session); err != nil {
			return "", err
		}
		return ctl.handleBalanceMenu(ctx, session, "")

	case "2":
		session.CurrentMenu = "transfer"
		session.Data["transfer_step"] = "recipient"
		if err := ctl.saveSession(ctx, session); err != nil {
			return "", err
		}
		return "CON Enter recipient's phone number:", nil

	case "3":
		session.CurrentMenu = "deposit"
		session.Data["deposit_step"] = "amount"
		if err := ctl.saveSession(ctx, session); err != nil {
			return "", err
		}
		return "CON Enter amount to deposit:", nil

	case "4":
		session.CurrentMenu = "withdraw"
		session.Data["withdraw_step"] = "amount"
		if err := ctl.saveSession(ctx, session); err != nil {
			return "", err
		}
		return "CON Enter amount to withdraw:", nil

	case "5":
		// Get account info
		accountInfo, err := ctl.getAccountInfo(ctx, session.PhoneNumber)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("END Account Information\nName: %s\nAccount: %s\nStatus: %s",
			accountInfo.Name, accountInfo.AccountNumber, accountInfo.Status), nil

	case "6":
		return "END Thank you for using our service.", nil

	default:
		return "CON Invalid option. Please try again.\n", nil
	}
}
