package models

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path"
	"strconv"
	"time"

	"github.com/mjarkk/jsonschema"
	"github.com/script-development/RT-CV/helpers/jsonHelpers"
)

// CV contains all information that belongs to a curriculum vitae
// TODO check the json removed fields if we actually should use them
type CV struct {
	Title                string                       `json:"-"` // Not supported yet
	ReferenceNumber      string                       `json:"referenceNumber" description:"The reference number (ID) of this CV on the site it was scraped from. We use this to track duplicated CVs"`
	CreatedAt            *jsonHelpers.RFC3339Nano     `json:"createdAt,omitempty"`
	LastChanged          *jsonHelpers.RFC3339Nano     `json:"lastChanged,omitempty"`
	Educations           []Education                  `json:"educations,omitempty"`
	WorkExperiences      []WorkExperience             `json:"workExperiences,omitempty"`
	PreferredJobs        []string                     `json:"preferredJobs,omitempty"`
	Languages            []Language                   `json:"languages,omitempty"`
	Competences          []Competence                 `json:"-"` // Not supported yet
	Interests            []Interest                   `json:"-"` // Not supported yet
	PersonalDetails      PersonalDetails              `json:"personalDetails" jsonSchema:"notRequired"`
	PersonalPresentation string                       `json:"-"` // Not supported yet
	DriversLicenses      []jsonHelpers.DriversLicense `json:"driversLicenses,omitempty"`
}

// Education is something a user has followed
type Education struct {
	Is uint8 `json:"is" description:"What kind of education is this?: 0: Unknown, 1: Education, 2: Course"`

	Name        string `json:"name"`
	Description string `json:"description"`
	Institute   string `json:"institute"`
	// TODO find difference between isCompleted and hasdiploma
	IsCompleted bool                     `json:"isCompleted"`
	HasDiploma  bool                     `json:"hasDiploma"`
	StartDate   *jsonHelpers.RFC3339Nano `json:"startDate"`
	EndDate     *jsonHelpers.RFC3339Nano `json:"endDate"`
}

// WorkExperience is experience in work
type WorkExperience struct {
	Description       string                   `json:"description"`
	Profession        string                   `json:"profession"`
	StartDate         *jsonHelpers.RFC3339Nano `json:"startDate"`
	EndDate           *jsonHelpers.RFC3339Nano `json:"endDate"`
	StillEmployed     bool                     `json:"stillEmployed"`
	Employer          string                   `json:"employer"`
	WeeklyHoursWorked int                      `json:"weeklyHoursWorked"`
}

// LanguageLevel is something that i'm not sure what it is
type LanguageLevel uint

// The lanague levels available
const (
	LanguageLevelUnknown LanguageLevel = iota
	LanguageLevelReasonable
	LanguageLevelGood
	LanguageLevelExcellent
)

func (ll LanguageLevel) String() string {
	switch ll {
	case LanguageLevelReasonable:
		return "Redelijk"
	case LanguageLevelGood:
		return "Goed"
	case LanguageLevelExcellent:
		return "Uitstekend"
	default:
		return "Onbekend"
	}
}

const langLevelDescription = `0. Unknown
1. Reasonable
2. Good
3. Excellent`

// Valid returns weather the language level is valid
func (ll LanguageLevel) Valid() bool {
	return ll >= LanguageLevelUnknown && ll <= LanguageLevelExcellent
}

func (ll LanguageLevel) asjson() json.RawMessage {
	return []byte(strconv.FormatUint(uint64(ll), 10))
}

// JSONSchemaDescribe implements schema.Describe
func (LanguageLevel) JSONSchemaDescribe() jsonschema.Property {
	return jsonschema.Property{
		Title:       "Language level",
		Description: langLevelDescription,
		Type:        jsonschema.PropertyTypeNumber,
		Enum: []json.RawMessage{
			LanguageLevelUnknown.asjson(),
			LanguageLevelReasonable.asjson(),
			LanguageLevelGood.asjson(),
			LanguageLevelExcellent.asjson(),
		},
	}
}

// Language is a language a user can speak
type Language struct {
	Name         string        `json:"name"`
	LevelSpoken  LanguageLevel `json:"levelSpoken"`
	LevelWritten LanguageLevel `json:"levelWritten"`
}

// Competence is an activity a user is "good" at
type Competence struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Interest contains a job the user is interested in
type Interest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// PersonalDetails contains personal info
type PersonalDetails struct {
	Initials          string                   `json:"initials,omitempty" jsonSchema:"notRequired"`
	FirstName         string                   `json:"firstName,omitempty" jsonSchema:"notRequired"`
	SurNamePrefix     string                   `json:"surNamePrefix,omitempty" jsonSchema:"notRequired"`
	SurName           string                   `json:"surName,omitempty" jsonSchema:"notRequired"`
	DateOfBirth       *jsonHelpers.RFC3339Nano `json:"dob,omitempty" jsonSchema:"notRequired"`
	Gender            string                   `json:"gender,omitempty" jsonSchema:"notRequired"`
	StreetName        string                   `json:"streetName,omitempty" jsonSchema:"notRequired"`
	HouseNumber       string                   `json:"houseNumber,omitempty" jsonSchema:"notRequired"`
	HouseNumberSuffix string                   `json:"houseNumberSuffix,omitempty" jsonSchema:"notRequired"`
	Zip               string                   `json:"zip,omitempty" jsonSchema:"notRequired"`
	City              string                   `json:"city,omitempty" jsonSchema:"notRequired"`
	Country           string                   `json:"country,omitempty" jsonSchema:"notRequired"`
	PhoneNumber       *jsonHelpers.PhoneNumber `json:"phoneNumber,omitempty" jsonSchema:"notRequired"`
	Email             string                   `json:"email,omitempty" jsonSchema:"notRequired"`
}

func getTemplateFromFile(funcs template.FuncMap, filename string) (*template.Template, error) {
	tmpl, err := template.New(filename).Funcs(funcs).ParseFiles("./assets/" + filename)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		// For testing perposes
		tmpl, err = template.New(filename).Funcs(funcs).ParseFiles("../assets/" + filename)
		if err != nil {
			return nil, err
		}
	}
	return tmpl, nil
}

// GetEmailHTML generates a HTML document that is used as email body
func (cv *CV) GetEmailHTML(profile Profile, matchText, domain string) (*bytes.Buffer, error) {
	tmpl, err := getTemplateFromFile(template.FuncMap{}, "email-template.html")
	if err != nil {
		return nil, err
	}

	input := struct {
		Profile   Profile
		Cv        *CV
		MatchText string
		LogoURL   string
		Domain    string

		// The normal `Profile.ID.String()`` is more of a debug value than a real id value so we add the hex to this field
		ProfileIDHex string
	}{
		Profile:      profile,
		ProfileIDHex: profile.ID.Hex(),
		Cv:           cv,
		MatchText:    matchText,
		LogoURL:      os.Getenv("EMAIL_LOGO_URL"),
		Domain:       domain,
	}

	buff := bytes.NewBuffer(nil)
	err = tmpl.Execute(buff, input)
	return buff, err
}

// GetPDF generates a PDF from a cv that can be send
// the pdfGeneratorProjectPath argument can be used to define the path to the pdf generator project
func (cv *CV) GetPDF(options *PdfOptions, pdfGeneratorProjectPath *string) (*os.File, error) {
	cvJSON, err := json.Marshal(cv)
	if err != nil {
		return nil, err
	}

	if pdfGeneratorProjectPath == nil {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}

		newPdfGeneratorProjectPath := path.Join(cwd, "pdf_generator")
		pdfGeneratorProjectPath = &newPdfGeneratorProjectPath
	}

	pdfGeneratorBin := path.Join(*pdfGeneratorProjectPath, "bin/pdf_generator.exe")
	f, err := os.CreateTemp(*pdfGeneratorProjectPath, "cv-*.pdf")
	if err != nil {
		return nil, err
	}
	pdfOutFile := f.Name()
	f.Close()

	args := []string{
		"--data", string(cvJSON),
		"--out", pdfOutFile,
	}

	emailLogoEnv := os.Getenv("EMAIL_LOGO_URL")
	if options != nil {
		if options.FontHeader != nil {
			args = append(args, "--font-bold", *options.FontHeader)
		}
		if options.FontRegular != nil {
			args = append(args, "--font-regular", *options.FontRegular)
		}
		if options.Style != nil {
			args = append(args, "--style", *options.Style)
		}
		if options.HeaderColor != nil {
			args = append(args, "--header-color", *options.HeaderColor)
		}
		if options.SubHeaderColor != nil {
			args = append(args, "--sub-header-color", *options.SubHeaderColor)
		}
		if options.LogoImageURL != nil {
			args = append(args, "--logo-image-url", *options.LogoImageURL)
		} else if len(emailLogoEnv) != 0 {
			args = append(args, "--logo-image-url", emailLogoEnv)
		}
		if options.CompanyName != nil {
			args = append(args, "--company-name", *options.CompanyName)
		}
		if options.CompanyAddress != nil {
			args = append(args, "--company-address", *options.CompanyAddress)
		}
	} else if len(emailLogoEnv) != 0 {
		args = append(args, "--logo-image-url", emailLogoEnv)
	}

	cmd := exec.Command(pdfGeneratorBin, args...)
	cmd.Dir = *pdfGeneratorProjectPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(pdfOutFile)
		if len(out) != 0 {
			return nil, errors.New(string(out))
		}
		return nil, err
	}

	pdfFile, err := os.Open(pdfOutFile)
	if err != nil {
		os.Remove(pdfOutFile)
		return nil, err
	}

	return pdfFile, nil
}

// Validate validates the cv and returns an error if it's not valid
func (cv *CV) Validate() error {
	// TODO: Needs more validation

	if cv.ReferenceNumber == "" {
		return errors.New("referenceNumber must be set")
	}

	now := time.Now()
	tomorrow := now.Add(time.Hour * 24)

	if cv.CreatedAt != nil && cv.CreatedAt.Time().After(tomorrow) {
		return errors.New("createdAt can't be in the future")
	}
	if cv.LastChanged != nil && cv.LastChanged.Time().After(tomorrow) {
		return errors.New("lastChanged can't be in the future")
	}
	if cv.PersonalDetails.DateOfBirth != nil && cv.PersonalDetails.DateOfBirth.Time().After(now.AddDate(-13, 0, 0)) {
		return errors.New("you need to be at least 13 years old to work")
	}

	for idx, lang := range cv.Languages {
		if !lang.LevelSpoken.Valid() {
			return fmt.Errorf("languages.%d.levelSpoken is invalid", idx)
		}
		if !lang.LevelWritten.Valid() {
			return fmt.Errorf("languages.%d.levelWritten is invalid", idx)
		}
	}

	return nil
}

// FullName returns the full name of the cv
func (cv *CV) FullName() string {
	details := cv.PersonalDetails

	res := details.FirstName
	if details.SurName == "" {
		return res
	}

	if details.SurNamePrefix == "" {
		return res + " " + details.SurName
	}

	return res + " " + details.SurNamePrefix + " " + details.SurName
}

// ExampleCV is a cv that can be used for demonstrative purposes
func ExampleCV() *CV {
	now := jsonHelpers.RFC3339Nano(time.Now()).ToPtr()
	return &CV{
		Title:           "Pilot with experience in farming simulator 2020",
		ReferenceNumber: "4455-PIETER",
		CreatedAt:       now,
		LastChanged:     now,

		Educations: []Education{{
			Name:        "Education name",
			Description: "Education description",
			Institute:   "Institute name",
			IsCompleted: true,
			HasDiploma:  true,
			StartDate:   now,
			EndDate:     now,
		}},
		WorkExperiences: []WorkExperience{{
			Description:       "WorkExperience description",
			Profession:        "hitman",
			StartDate:         now,
			EndDate:           now,
			StillEmployed:     true,
			Employer:          "Bond.. James bond",
			WeeklyHoursWorked: 60,
		}},
		PreferredJobs: []string{"hitman"},
		Languages: []Language{{
			Name:         "Language name",
			LevelSpoken:  LanguageLevelExcellent,
			LevelWritten: LanguageLevelGood,
		}},
		Competences: []Competence{{
			Name:        "Competence name",
			Description: "Competence description",
		}},
		Interests: []Interest{{
			Name:        "Interest name",
			Description: "Interest description",
		}},
		PersonalPresentation: "Sir",
		DriversLicenses: []jsonHelpers.DriversLicense{
			jsonHelpers.NewDriversLicense("A"),
			jsonHelpers.NewDriversLicense("B"),
			jsonHelpers.NewDriversLicense("C"),
		},
		PersonalDetails: PersonalDetails{
			Initials:          "P.S.",
			FirstName:         "Pietter",
			SurNamePrefix:     "Ven ther",
			SurName:           "Steen",
			DateOfBirth:       now,
			Gender:            "Male",
			StreetName:        "Streetname abc",
			HouseNumber:       "33",
			HouseNumberSuffix: "b",
			Zip:               "9779AB",
			City:              "Groningen",
			Country:           "Netherlands",
			PhoneNumber:       &jsonHelpers.PhoneNumber{IsLocal: true, Number: 611223344},
			Email:             "p.steen@very-smart-people.com",
		},
	}
}
