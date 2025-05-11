package main

import (
	"log"
	"os"
	"ussd-wrapper/docs"
	"ussd-wrapper/router"
)

// @title USSD Wrapper Service API
// @version 3.0
// @description This API documents exposes all the available API endpoints for USSD Wrapper service
// @termsOfService https://corvus-tech.com/terms

// @contact.name API Support
// @contact.url https://corvus-tech.com/contact-us
// @contact.email tech@corvus-tech.com

// @license.name Apache 2.0
// @license.url https://www.apache.org/licenses/LICENSE-2.0.html

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name api-key

func main() {

	docs.SwaggerInfo.Version = "3.0"
	docs.SwaggerInfo.Host = os.Getenv("base_url")
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Schemes = []string{os.Getenv("scheme")}

	if err := router.Init(); err != nil {
		log.Fatalf("Startup failed: %v", err)
	}
}
