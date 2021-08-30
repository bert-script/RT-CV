package controller

import (
	"errors"
	"os"

	"github.com/apex/log"
	"github.com/gofiber/fiber/v2"
	"github.com/script-development/RT-CV/db"
	"github.com/script-development/RT-CV/helpers/routeBuilder"
	"github.com/script-development/RT-CV/models"
	"go.mongodb.org/mongo-driver/mongo"
)

// IMap is a wrapper around map[string]interface{} that's faster to use
type IMap map[string]interface{}

// Routes defines the routes used
func Routes(app *fiber.App, dbConn db.Connection, serverSeed []byte) {
	b := routeBuilder.New(app)

	b.Group(`/api/v1`, func(b *routeBuilder.Router) {
		b.Group(`/schema`, func(b *routeBuilder.Router) {
			b.NGet(`/openAPI`, routeGetOpenAPISchema(b))
			b.NGet(`/cv`, routeGetCvSchema)
		})

		b.Group(`/auth`, func(b *routeBuilder.Router) {
			b.NGet(`/keyinfo`, routeGetKeyInfo, requiresAuth(0))
			b.NGet(`/seed`, routeAuthSeed)
		})

		b.Group(`/scraper`, func(b *routeBuilder.Router) {
			b.Post(`/scanCV`, routeScraperScanCV)
		}, requiresAuth(models.APIKeyRoleScraper|models.APIKeyRoleDashboard))

		secretsRoutes := func(b *routeBuilder.Router) {
			b.Get(``, routeGetSecrets)
			b.Group(`/:key`, func(b *routeBuilder.Router) {
				b.Delete(``, routeDeleteSecret)
				b.Group(`/:encryptionKey`, func(b *routeBuilder.Router) {
					b.Get(``, routeGetSecret)
					b.Put(``, routeUpdateSecret)
					b.Post(``, routeCreateSecret)
				}, validEncryptionKeyMiddleware())
			}, validKeyMiddleware())
		}
		b.Group(`/secrets/myKey`, secretsRoutes, requiresAuth(models.APIKeyRoleAll), middlewareBindMyKey())
		b.Group(`/secrets/otherKey`, func(b *routeBuilder.Router) {
			b.Get(``, routeGetAllSecretsFromAllKeys)
			b.Group(`/:keyID`, secretsRoutes, middlewareBindKey())
		}, requiresAuth(models.APIKeyRoleDashboard))

		b.Group(`/control`, func(b *routeBuilder.Router) {
			b.Group(`/profiles`, func(b *routeBuilder.Router) {
				b.Post(``, routeCreateProfile)
				b.Get(``, routeAllProfiles)
				b.Group(`/:profile`, func(b *routeBuilder.Router) {
					b.Get(``, routeGetProfile)
					// b.Put(``, routeModifyProfile) // TODO
					b.Delete(``, routeDeleteProfile)
				}, middlewareBindProfile())
			})
		}, requiresAuth(models.APIKeyRoleController))

		b.Group(`/keys`, func(b *routeBuilder.Router) {
			b.Get(``, routeGetKeys)
			b.Post(``, routeCreateKey)
			b.Group(`/:keyID`, func(b *routeBuilder.Router) {
				b.Get(``, routeGetKey)
				b.Put(``, routeUpdateKey)
				b.Delete(``, routeDeleteKey)
			}, middlewareBindKey())
		}, requiresAuth(models.APIKeyRoleDashboard))
	}, InsertData(dbConn, serverSeed))

	_, err := os.Stat("./dashboard/out")
	if err == os.ErrNotExist {
		log.Warn("dashboard not build, you won't beable to use the dashboard")
	} else if err != nil {
		log.WithError(err).Warn("unable to set dashboard routes, you won't beable to use the dashboard")
	} else {
		b.Static("", "./dashboard/out/index.html", fiber.Static{Compress: true})
		b.Static("login", "./dashboard/out/login.html", fiber.Static{Compress: true})
		b.Static("tryMatcher", "./dashboard/out/tryMatcher.html", fiber.Static{Compress: true})
		b.Static("docs", "./dashboard/out/docs.html", fiber.Static{Compress: true})
		b.Static("_next", "./dashboard/out/_next", fiber.Static{Compress: true})
		b.Static("favicon.ico", "./dashboard/out/favicon.ico", fiber.Static{Compress: true})
		app.Use(func(c *fiber.Ctx) error {
			// 404 page
			return c.Status(404).SendFile("./dashboard/out/404.html", true)
		})
	}
}

// FiberErrorHandler handles errors in fiber
// In our case that means we change the errors from text to json
func FiberErrorHandler(c *fiber.Ctx, err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrorRes(c, 404, errors.New("item not found"))
	}
	return ErrorRes(c, 500, err)
}

// ErrorRes returns the error response
func ErrorRes(c *fiber.Ctx, status int, err error) error {
	return c.Status(status).JSON(IMap{
		"error": err.Error(),
	})
}
