package models

import (
	"testing"
	"time"

	"github.com/script-development/RT-CV/db"
	"github.com/script-development/RT-CV/helpers/jsonHelpers"
	. "github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestGetHtml(t *testing.T) {
	matchTest := "this is a test text that should re-appear in the response html"

	cv := CV{
		Title:           "Pilot with experience in farming simulator 2020",
		ReferenceNumber: "4455-PIETER",

		PersonalDetails: PersonalDetails{
			Initials:          "P.S.",
			FirstName:         "D.R. Pietter",
			SurNamePrefix:     "Ven ther",
			SurName:           "Steen",
			DateOfBirth:       jsonHelpers.RFC3339Nano(time.Now()),
			Gender:            "Apache helicopter",
			StreetName:        "Streetname abc",
			HouseNumber:       "33",
			HouseNumberSuffix: "b",
			Zip:               "9999AB",
			City:              "Groningen",
			Country:           "Netherlands",
			PhoneNumber:       "06-11223344",
			Email:             "dr.p.steen@smart-people.com",
		},
	}

	profileObjectID := primitive.NewObjectID()
	profile := Profile{
		M:       db.M{ID: profileObjectID},
		Name:    "profile name",
		Domains: []string{"test.com"},
	}

	htmlBuff, err := cv.GetHTML(profile, matchTest)
	if err != nil {
		NoError(t, err)
		return
	}

	html := htmlBuff.String()
	Contains(t, html, matchTest)
	Contains(t, html, cv.PersonalDetails.FirstName+" "+cv.PersonalDetails.SurName)
	Contains(t, html, cv.PersonalDetails.Email)
	Contains(t, html, cv.PersonalDetails.PhoneNumber)
	Contains(t, html, profile.Name)
	Contains(t, html, cv.ReferenceNumber)
	Contains(t, html, profile.ID.Hex())
}
