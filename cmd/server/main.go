package main

import (
	"fmt"
	"os"

	"github.com/antonrybalko/image-service-go/internal/app"
)

func main() {
	// Create a new service
	service, err := app.NewService()
	if err != nil {
		fmt.Printf("Failed to initialize service: %v\n", err)
		os.Exit(1)
	}
	defer service.Cleanup()

	// Start the service
	if err := service.Start(); err != nil {
		fmt.Printf("Failed to start service: %v\n", err)
		os.Exit(1)
	}

	// Wait for shutdown signal
	service.WaitForShutdown()
}
