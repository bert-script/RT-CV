package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"syscall"

	"github.com/apex/log"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
	"github.com/script-development/RT-CV/controller"
	"github.com/script-development/RT-CV/db"
	"github.com/script-development/RT-CV/db/mongo"
	"github.com/script-development/RT-CV/db/mongo/backup"
	"github.com/script-development/RT-CV/helpers/emailservice"
	"github.com/script-development/RT-CV/helpers/random"
	"github.com/script-development/RT-CV/mock"
	"github.com/script-development/RT-CV/models"
)

// AppVersion is used for the X-App-Version header and in the /health route
// This variable can be set by:
//   go build -ldflags "-X main.AppVersion=1.0.0"
var AppVersion = "LOCAL"

func main() {
	doProfile := false
	flag.BoolVar(&doProfile, "profile", false, "start profiling")
	flag.Parse()

	// Seed the random package so generated values are "actually" random
	random.Seed()

	if doProfile {
		f, err := os.Create("cpu.profile")
		if err != nil {
			log.WithField("error", err).Fatal("could not create cpu profile")
		}

		err = pprof.StartCPUProfile(f)
		if err != nil {
			log.WithField("error", err).Fatal("could not start CPU profile")
		}

		exitSignal := make(chan os.Signal, 1)
		signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-exitSignal
			pprof.StopCPUProfile()
			f.Close()
			fmt.Println("saved cpu profile to cpu.profile")
			fmt.Println("the profile can be inspected using: go tool pprof -http localhost:3333 cpu.profile")
			os.Exit(0)
		}()
	}

	// Loading the .env if available
	_, err := os.Stat(".env")
	if err == nil {
		err := godotenv.Load()
		if err != nil {
			log.Fatalf("Error loading .env file: %s", err.Error())
		}
	} else {
		log.Info("No .env file found")
	}

	// Initialize the mail service
	err = emailservice.Setup(
		emailservice.EmailServerConfiguration{
			Identity: os.Getenv("EMAIL_IDENTITY"),
			Username: os.Getenv("EMAIL_USER"),
			Password: os.Getenv("EMAIL_PASSWORD"),
			Host:     os.Getenv("EMAIL_HOST"),
			Port:     os.Getenv("EMAIL_PORT"),
			From:     os.Getenv("EMAIL_FROM"),
		},
		nil,
	)
	if err != nil {
		log.WithError(err).Error("Error initializing email service")
		os.Exit(1)
	}

	// Initialize the database
	var dbConn db.Connection
	useTestingDB := strings.ToLower(os.Getenv("USE_TESTING_DB")) == "true"
	if useTestingDB {
		dbConn = mock.NewMockDB()
		log.WithField("id", mock.DashboardKey.ID.Hex()).WithField("key", mock.DashboardKey.Key).Info("Mock dashboard key")
	} else {
		dbConn = mongo.ConnectToDB()
	}

	dbConn.RegisterEntries(
		&models.APIKey{},
		&models.Profile{},
		&models.Secret{},
		&models.Match{},
		&models.Backup{},
	)

	if !useTestingDB && strings.ToLower(os.Getenv("MONGODB_BACKUP_ENABLED")) == "true" {
		options := backup.StartScheduleOptions{
			BackupEncryptionKey: os.Getenv("MONGODB_BACKUP_KEY"),
			S3Endpoint:          os.Getenv("BACKUP_S3_ENDPOINT"),
			S3AccessKeyID:       os.Getenv("BACKUP_S3_ACCESS_KEY_ID"),
			S3SecretAccessKey:   os.Getenv("BACKUP_S3_SECRET_ACCESS_KEY"),
			S3Bucket:            os.Getenv("BACKUP_S3_BUCKET"),
			S3UseSSL:            strings.ToLower(os.Getenv("BACKUP_S3_USE_SSL")) == "true",
		}
		backup.StartsSchedule(dbConn, options)
	}

	models.CheckDashboardKeyExists(dbConn)

	// Create a new fiber instance (http server)
	// do not use fiber Prefork!, this app is not written to support it
	app := fiber.New(fiber.Config{
		ErrorHandler: controller.FiberErrorHandler,
	})
	app.Use(recover.New())
	app.Use(cors.New())
	app.Use(logger.New())
	app.Use(func(c *fiber.Ctx) error {
		err = c.Next()
		c.Set("X-App-Version", AppVersion)
		return err
	})

	// Setup the app routes
	controller.Routes(app, AppVersion, dbConn, false)

	testingDieAfterInit := os.Getenv("TESTING_DIE_AFTER_INIT")
	if testingDieAfterInit == "true" || testingDieAfterInit == "TRUE" {
		// Used in the CD/CI to test if the application can startup without problems
		return
	}

	// Start the webserver
	log.Fatal(app.Listen(":4000").Error())
}
