package controller

import (
	"github.com/gofiber/fiber/v2"
	"github.com/script-development/RT-CV/controller/ctx"
	"github.com/script-development/RT-CV/db"
	"github.com/script-development/RT-CV/helpers/routeBuilder"
	"github.com/script-development/RT-CV/models"
	"go.mongodb.org/mongo-driver/mongo"
)

var routeAllProfiles = routeBuilder.R{
	Description: "get all profiles stored in the database",
	Res:         []models.Profile{},
	Fn: func(c *fiber.Ctx) error {
		profiles := ctx.GetProfiles(c)
		return c.JSON(profiles)
	},
}

func middlewareBindProfile() fiber.Handler {
	return func(c *fiber.Ctx) error {
		profileParam := c.Params(`profile`)
		profiles := ctx.GetProfiles(c)
		for _, profile := range *profiles {
			if profile.ID.Hex() == profileParam {
				c.SetUserContext(
					ctx.SetProfile(
						c.UserContext(),
						&profile,
					),
				)
				return c.Next()
			}
		}
		return mongo.ErrNoDocuments
	}
}

var routeGetProfile = routeBuilder.R{
	Description: "get a profile based on it's ID from the database",
	Res:         models.Profile{},
	Fn: func(c *fiber.Ctx) error {
		profile := ctx.GetProfile(c)
		return c.JSON(profile)
	},
}

var routeCreateProfile = routeBuilder.R{
	Description: "create a new profile that can match scraped CVs",
	Res:         models.Profile{},
	Body:        models.Profile{},
	Fn: func(c *fiber.Ctx) error {
		var profile models.Profile
		err := c.BodyParser(&profile)
		if err != nil {
			return err
		}

		// FIXME add validation to profile

		// Set the ID of the profile
		profile.M = db.NewM()

		// Save the profile to the database
		dbConn := ctx.GetDbConn(c)
		err = dbConn.Insert(&profile)
		if err != nil {
			return err
		}

		ctxProfiles := ctx.GetProfiles(c)
		*ctxProfiles = append(*ctxProfiles, profile)

		return c.JSON(profile)
	},
}

func routeModifyProfile(c *fiber.Ctx) error {
	profile := ctx.GetProfile(c)
	// FIXME
	return c.JSON(profile)
}

var routeDeleteProfile = routeBuilder.R{
	Description: "Delete a profile stored in the database",
	Res:         models.Profile{},
	Fn: func(c *fiber.Ctx) error {
		profile := ctx.GetProfile(c)
		dbConn := ctx.GetDbConn(c)
		err := dbConn.DeleteByID(profile)
		if err != nil {
			return err
		}

		// Update the cached local profiles list
		profilesInDB, err := models.GetProfiles(dbConn)
		if err != nil {
			return err
		}
		ctxProfiles := ctx.GetProfiles(c)
		*ctxProfiles = profilesInDB

		return c.JSON(profile)
	},
}
