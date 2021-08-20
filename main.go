package main

import (
	"github.com/apex/log"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/script-development/RT-CV/controller"
	"github.com/script-development/RT-CV/db/mongo"
	"github.com/script-development/RT-CV/models"
)

func main() {
	// Loading the .env if available
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %s", err.Error())
	}

	// Connect to the database using the env variables
	dbConn := mongo.ConnectToDB()
	dbConn.RegisterEntries(
		&models.ApiKey{},
		&models.Profile{},
		&models.Secret{},
	)

	// Create a new fiber instance (http server)
	// do not use fiber Prefork!, this app is not written to support it
	app := fiber.New()

	// Setup the app routes
	controller.Routes(app, dbConn)

	// Start the webserver
	log.Fatal(app.Listen(":3000").Error())
}
