package controller

import (
	"context"

	"github.com/apex/log"
	"github.com/gofiber/fiber/v2"
	"github.com/script-development/RT-CV/controller/ctx"
	"github.com/script-development/RT-CV/db"
	"github.com/script-development/RT-CV/helpers/auth"
	"github.com/script-development/RT-CV/models"
)

// InsertData adds the profiles to every route
func InsertData(dbConn db.Connection, serverSeed []byte) fiber.Handler {
	profiles, err := models.GetProfiles(dbConn)
	if err != nil {
		log.Fatal(err.Error())
	}
	requestContext := ctx.SetProfiles(context.Background(), &profiles)

	keys, err := models.GetAPIKeys(dbConn)
	if err != nil {
		log.Fatal(err.Error())
	}

	requestContext = ctx.SetAuth(requestContext, auth.New(keys, serverSeed))
	requestContext = ctx.SetDbConn(requestContext, dbConn)

	// Pre define loggerEntity so we only take once memory
	loggerEntity := log.Entry{
		Logger: log.Log.(*log.Logger),
	}

	return func(c *fiber.Ctx) error {
		// reset loggerEntity
		loggerEntity = log.Entry{
			Logger: loggerEntity.Logger,
		}

		c.SetUserContext(
			ctx.SetLogger(requestContext, &loggerEntity),
		)
		return c.Next()
	}
}
