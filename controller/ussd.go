package controller

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"ussd-wrapper/library"
)

func (ctl *Controller) HandleUSSD(c echo.Context) error {
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

func (ctl *Controller) HandleUSSD1(c echo.Context) error {
	ctx := c.Request().Context()

	// Redis example
	traceID := library.GetTraceID(ctx)
	_ = ctl.Redis.Set(ctx, "trace:"+traceID, "some_value", 0).Err()

	return c.String(http.StatusOK, "CON USSD ready with context.")
}
