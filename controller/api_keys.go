package controller

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/script-development/RT-CV/controller/ctx"
	"github.com/script-development/RT-CV/db"
	"github.com/script-development/RT-CV/helpers/routeBuilder"
	"github.com/script-development/RT-CV/helpers/validation"
	"github.com/script-development/RT-CV/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var routeGetScraperKeys = routeBuilder.R{
	Description: "Get all scraper keys from the database",
	Res:         []models.APIKey{},
	Fn: func(c *fiber.Ctx) error {
		dbConn := ctx.GetDbConn(c)
		keys, err := models.GetScraperAPIKeys(dbConn)
		if err != nil {
			return err
		}
		return c.JSON(keys)
	},
}

var routeGetKeys = routeBuilder.R{
	Description: "get all api keys from the database",
	Res:         []models.APIKey{},
	Fn: func(c *fiber.Ctx) error {
		dbConn := ctx.GetDbConn(c)
		keys, err := models.GetAPIKeys(dbConn)
		if err != nil {
			return err
		}
		return c.JSON(keys)
	},
}

type apiKeyModifyCreateData struct {
	Enabled *bool              `json:"enabled"`
	Name    *string            `json:"name"`
	Domains []string           `json:"domains"`
	Key     *string            `json:"key"`
	Roles   *models.APIKeyRole `json:"roles"`
}

var routeCreateKey = routeBuilder.R{
	Description: "create a new api key",
	Body:        apiKeyModifyCreateData{},
	Res:         models.APIKey{},
	Fn: func(c *fiber.Ctx) error {
		dbConn := ctx.GetDbConn(c)

		body := apiKeyModifyCreateData{}
		err := c.BodyParser(&body)
		if err != nil {
			return err
		}

		newAPIKey := &models.APIKey{
			M:       db.NewM(),
			Enabled: body.Enabled == nil || *body.Enabled,
		}

		if body.Name == nil {
			return errors.New("name is required")
		} else if len(*body.Name) == 0 {
			return errors.New("name cannot be empty")
		}
		newAPIKey.Name = *body.Name

		if body.Domains == nil {
			return errors.New("domains should be set")
		} else if len(body.Domains) < 1 {
			return errors.New("there should at least be one domain")
		} else {
			err := validation.ValidDomainListAndFormat(&body.Domains, true)
			if err != nil {
				return err
			}
			for idx, domain := range body.Domains {
				body.Domains[idx] = strings.ToLower(domain)
			}
			newAPIKey.Domains = body.Domains
		}

		if body.Key == nil {
			return errors.New("key should be set")
		} else if len(*body.Key) < 16 {
			return errors.New("key must have a length of at least 16 chars")
		} else {
			newAPIKey.Key = *body.Key
		}

		if body.Roles == nil {
			return errors.New("roles should be set")
		} else if !body.Roles.Valid() {
			return errors.New("roles are invalid")
		} else {
			newAPIKey.Roles = *body.Roles
		}

		err = dbConn.Insert(newAPIKey)
		if err != nil {
			return err
		}

		return c.JSON(newAPIKey)
	},
}

var routeDeleteKey = routeBuilder.R{
	Description: "delete an api key",
	Res:         models.APIKey{},
	Fn: func(c *fiber.Ctx) error {
		apiKey := ctx.GetAPIKeyFromParam(c)
		if apiKey.System {
			return errors.New("you are not allowed to remove system keys")
		}

		dbConn := ctx.GetDbConn(c)
		err := dbConn.DeleteByID(apiKey)
		if err != nil {
			return err
		}

		ctx.GetAuth(c).RemoveKeyCache(apiKey.ID.Hex())

		return c.JSON(apiKey)
	},
}

var routeGetKey = routeBuilder.R{
	Description: "get an api key from the database based on it's ID",
	Res:         models.APIKey{},
	Fn: func(c *fiber.Ctx) error {
		apiKey := ctx.GetAPIKeyFromParam(c)
		return c.JSON(apiKey)
	},
}

var routeUpdateKey = routeBuilder.R{
	Description: "Update an api key",
	Body:        apiKeyModifyCreateData{},
	Res:         models.APIKey{},
	Fn: func(c *fiber.Ctx) error {
		dbConn := ctx.GetDbConn(c)
		apiKey := ctx.GetAPIKeyFromParam(c)
		if apiKey.System {
			return errors.New("you are not allowed to remove system keys")
		}

		body := apiKeyModifyCreateData{}
		err := c.BodyParser(&body)
		if err != nil {
			return err
		}

		if body.Enabled != nil {
			apiKey.Enabled = *body.Enabled
		}

		if body.Name != nil {
			apiKey.Name = *body.Name
		}

		if body.Domains != nil {
			if len(body.Domains) < 1 {
				return errors.New("there should at least be one domain")
			}
			err := validation.ValidDomainListAndFormat(&body.Domains, true)
			if err != nil {
				return err
			}
			for idx, domain := range body.Domains {
				body.Domains[idx] = strings.ToLower(domain)
			}
			apiKey.Domains = body.Domains
		}

		keyChanged := false
		if body.Key != nil {
			if len(*body.Key) < 16 {
				return errors.New("key must have a length of at least 16 chars")
			}
			keyChanged = apiKey.Key != *body.Key
			apiKey.Key = *body.Key
		}

		if body.Roles != nil {
			if !body.Roles.Valid() {
				return errors.New("roles are invalid")
			}
			apiKey.Roles = *body.Roles
		}

		err = dbConn.UpdateByID(apiKey)
		if err != nil {
			return err
		}

		if keyChanged {
			ctx.GetAuth(c).RemoveKeyCache(apiKey.ID.Hex())
		}

		return c.JSON(apiKey)
	},
}

// middlewareBindMyKey sets the APIKeyFromParam to the api key used to authenticate
func middlewareBindMyKey() routeBuilder.M {
	return routeBuilder.M{
		Fn: func(c *fiber.Ctx) error {
			apiKey := ctx.GetKey(c)

			c.SetUserContext(
				ctx.SetAPIKeyFromParam(
					c.UserContext(),
					apiKey,
				),
			)

			return c.Next()
		},
	}
}

func middlewareBindKey() routeBuilder.M {
	return routeBuilder.M{
		Fn: func(c *fiber.Ctx) error {
			keyParam := c.Params(`keyID`)
			keyID, err := primitive.ObjectIDFromHex(keyParam)
			if err != nil {
				return err
			}
			dbConn := ctx.GetDbConn(c)
			apiKey := models.APIKey{}
			query := bson.M{"_id": keyID}
			args := db.FindOptions{NoDefaultFilters: true}
			err = dbConn.FindOne(&apiKey, query, args)
			if err != nil {
				return err
			}

			c.SetUserContext(
				ctx.SetAPIKeyFromParam(
					c.UserContext(),
					&apiKey,
				),
			)

			return c.Next()
		},
	}
}
