package controller

import (
	"errors"
	"sync"

	"github.com/apex/log"
	"github.com/gofiber/fiber/v2"
	"github.com/script-development/RT-CV/controller/ctx"
	"github.com/script-development/RT-CV/db"
	"github.com/script-development/RT-CV/helpers/match"
	"github.com/script-development/RT-CV/helpers/routeBuilder"
	"github.com/script-development/RT-CV/models"
)

// RouteScraperScanCVBody is the request body of the routeScraperScanCV
type RouteScraperScanCVBody struct {
	CV    models.CV `json:"cv"`
	Debug bool      `json:"debug" jsonSchema:"hidden"`
}

// RouteScraperScanCVRes contains the response data of routeScraperScanCV
type RouteScraperScanCVRes struct {
	Success bool `json:"success"`

	// Matches is only set if the debug property is set
	Matches []match.FoundMatch `json:"matches" jsonSchema:"hidden"`
}

var routeScraperScanCV = routeBuilder.R{
	Description: "Main route to scrape the CV",
	Res:         RouteScraperScanCVRes{},
	Body:        RouteScraperScanCVBody{},
	Fn: func(c *fiber.Ctx) error {
		key := ctx.GetKey(c)
		requestID := ctx.GetRequestID(c)
		dbConn := ctx.GetDbConn(c)
		logger := ctx.GetLogger(c)

		body := RouteScraperScanCVBody{}
		err := c.BodyParser(&body)
		if err != nil {
			return err
		}

		err = dashboardListeners.publish("recived_cv", body.CV)
		if err != nil {
			return err
		}

		if body.Debug && !key.Roles.ContainsSome(models.APIKeyRoleDashboard) {
			return ErrorRes(
				c,
				fiber.StatusForbidden,
				errors.New("you are not allowed to set the debug field, only api keys with the Dashboard role can set it"),
			)
		}

		err = body.CV.Validate()
		if err != nil {
			return ErrorRes(
				c,
				fiber.StatusBadRequest,
				err,
			)
		}

		profiles, err := models.GetProfiles(dbConn)
		if err != nil {
			return err
		}
		matchedProfiles := match.Match(key.Domains, profiles, body.CV)
		foundMatches := len(matchedProfiles) != 0

		// Insert analytics data
		analyticsData := make([]db.Entry, len(matchedProfiles))
		for idx := range matchedProfiles {
			matchedProfiles[idx].Matches.RequestID = requestID
			matchedProfiles[idx].Matches.KeyID = key.ID
			matchedProfiles[idx].Matches.Debug = body.Debug

			analyticsData[idx] = &matchedProfiles[idx].Matches
		}

		if foundMatches {
			go func(logger *log.Entry, analyticsData []db.Entry) {
				err := dbConn.Insert(analyticsData...)
				if err != nil {
					logger.WithError(err).Error("analytics data insertion failed")
				}
			}(logger.WithField("analytics_entries_count", len(analyticsData)), analyticsData)
		}

		if body.Debug {
			return c.JSON(RouteScraperScanCVRes{Success: true, Matches: matchedProfiles})
		}

		if foundMatches {
			logger.Infof("found %d matches", len(matchedProfiles))

			var wg sync.WaitGroup

			for _, aMatch := range matchedProfiles {
				err := aMatch.SendMatch(&wg, &body.CV)
				if err != nil {
					log.WithError(err).Error("sending match error")
				}
			}

			wg.Wait()
		}

		return c.JSON(RouteScraperScanCVRes{Success: true})
	},
}
