package match

import (
	"testing"
	"time"

	"github.com/script-development/RT-CV/models"
	. "github.com/stretchr/testify/assert"
)

func MustMatchSingle(t *testing.T, p models.Profile, cv models.Cv) {
	p.Site.Domain = "werk.nl"
	siteId := 1
	p.SiteId = &siteId
	p.Active = true

	matches := Match("werk.nl", []models.Profile{p}, cv)
	Equal(t, 1, len(matches), matches)
}

func MustNotMatchSingle(t *testing.T, p models.Profile, cv models.Cv) {
	p.Site.Domain = "werk.nl"
	p.Active = true

	matches := Match("werk.nl", []models.Profile{p}, cv)
	Equal(t, 0, len(matches), matches)
}

func TestMatchSiteMismatch(t *testing.T) {
	siteId := 1
	matches := Match("werk.nl", []models.Profile{{
		Site:   models.Site{Domain: "gangster_at_work.crib"},
		SiteId: &siteId,
		Active: true,
	}}, models.Cv{})
	Equal(t, 0, len(matches), matches)
}

func TestMatchNonActive(t *testing.T) {
	matches := Match("werk.nl", []models.Profile{{Active: false}}, models.Cv{})
	Equal(t, 0, len(matches), matches)
}

func TestMatchEmptyProfile(t *testing.T) {
	MustMatchSingle(t, models.Profile{}, models.Cv{})
}

func TestMatchZipCode(t *testing.T) {
	cases := []string{"1500AB", "1000AB", "2000"}
	for _, case_ := range cases {
		// Valid 1 to 1 match
		MustMatchSingle(
			t,
			models.Profile{Zipcodes: []models.ProfileZipcode{{Zipcode: models.Zipcode{From: 1000, To: 2000}}}},
			models.Cv{PersonalDetails: models.PersonalDetails{Zip: case_}},
		)

		// Outside of range
		MustNotMatchSingle(
			t,
			models.Profile{Zipcodes: []models.ProfileZipcode{{Zipcode: models.Zipcode{From: 6000, To: 9000}}}},
			models.Cv{PersonalDetails: models.PersonalDetails{Zip: case_}},
		)
	}

	// invalid CV zip code
	MustNotMatchSingle(
		t,
		models.Profile{Zipcodes: []models.ProfileZipcode{{Zipcode: models.Zipcode{From: 1000, To: 2000}}}},
		models.Cv{PersonalDetails: models.PersonalDetails{Zip: "AAAAAA"}},
	)

	// Multiple zip codes
	MustMatchSingle(
		t,
		models.Profile{Zipcodes: []models.ProfileZipcode{
			{Zipcode: models.Zipcode{From: 1000, To: 2000}},
			{Zipcode: models.Zipcode{From: 3000, To: 3500}},
			{Zipcode: models.Zipcode{From: 4000, To: 5000}},
			{Zipcode: models.Zipcode{From: 6000, To: 8000}},
		}},
		models.Cv{PersonalDetails: models.PersonalDetails{Zip: "4100AB"}},
	)

	// Reverse zip code
	MustMatchSingle(
		t,
		models.Profile{Zipcodes: []models.ProfileZipcode{
			{Zipcode: models.Zipcode{From: 6000, To: 2000}},
		}},
		models.Cv{PersonalDetails: models.PersonalDetails{Zip: "4100AB"}},
	)
}

func TestMatchEducation(t *testing.T) {
	// No educations in CV
	MustNotMatchSingle(
		t,
		models.Profile{MustEducation: true, Educations: []models.ProfileEducation{{}}},
		models.Cv{},
	)

	// No educations in CV
	MustNotMatchSingle(
		t,
		models.Profile{Educations: []models.ProfileEducation{{}}},
		models.Cv{},
	)

	// Match on education
	MustMatchSingle(
		t,
		models.Profile{Educations: []models.ProfileEducation{{Education: models.DBEducation{Name: "Bananenplukker"}}}},
		models.Cv{Educations: []models.Education{{Name: "Bananenplukker"}}},
	)

	// Match with multiple educations
	MustMatchSingle(
		t,
		models.Profile{Educations: []models.ProfileEducation{
			{Education: models.DBEducation{Name: "professioneel peren eten"}},
			{Education: models.DBEducation{Name: "Bananenplukker"}},
		}},
		models.Cv{Educations: []models.Education{
			{Name: "Pro gangster"},
			{Name: "Bananenplukker"},
		}},
	)

	// Match on courses
	MustMatchSingle(
		t,
		models.Profile{Educations: []models.ProfileEducation{{Education: models.DBEducation{Name: "Bananenplukker"}}}},
		models.Cv{Courses: []models.Course{{Name: "Bananenplukker"}}},
	)

	// No matches
	MustNotMatchSingle(
		t,
		models.Profile{
			MustEducation: true,
			Educations: []models.ProfileEducation{
				{Education: models.DBEducation{Name: "Peren Plukker"}},
			},
		},
		models.Cv{
			Educations: []models.Education{{Name: "Bananenplukker"}},
			Courses:    []models.Course{{Name: "How to be a gangster for dummies"}},
		},
	)
}

func TestMatchEducationMustFinish(t *testing.T) {
	// Education not finished
	MustNotMatchSingle(
		t,
		models.Profile{
			MustEducationFinished: true,
			Educations:            []models.ProfileEducation{{Education: models.DBEducation{Name: "Bananenplukker"}}},
		},
		models.Cv{Educations: []models.Education{{Name: "Bananenplukker"}}},
	)

	// Education finished
	MustMatchSingle(
		t,
		models.Profile{
			MustEducationFinished: true,
			Educations:            []models.ProfileEducation{{Education: models.DBEducation{Name: "Bananenplukker"}}},
		},
		models.Cv{Educations: []models.Education{{Name: "Bananenplukker", HasDiploma: true}}},
	)
}

func TestMatchEducationYearsSinceEducation(t *testing.T) {
	MustMatchSingle(
		t,
		models.Profile{
			YearsSinceEducation: 2,
		},
		models.Cv{
			Educations: []models.Education{{
				Name:    "Bananenplukker",
				EndDate: time.Now().AddDate(-1, 0, 0).Format(time.RFC3339),
			}},
		},
	)

	MustMatchSingle(
		t,
		models.Profile{
			YearsSinceEducation: 2,
		},
		models.Cv{
			Courses: []models.Course{{
				Name:    "Bananenplukker",
				EndDate: time.Now().AddDate(-1, 0, 0).Format(time.RFC3339),
			}},
		},
	)

	MustMatchSingle(
		t,
		models.Profile{
			YearsSinceEducation: 2,
		},
		models.Cv{
			Educations: []models.Education{
				{
					Name:    "Bananenplukker",
					EndDate: time.Now().AddDate(-3, 0, 0).Format(time.RFC3339),
				},
				{
					Name:    "Bananenplukker",
					EndDate: time.Now().AddDate(-1, 0, 0).Format(time.RFC3339),
				},
				{
					Name:    "Bananenplukker",
					EndDate: time.Now().AddDate(-3, 0, 0).Format(time.RFC3339),
				},
			},
		},
	)

	MustNotMatchSingle(
		t,
		models.Profile{
			YearsSinceEducation: 1,
		},
		models.Cv{
			Educations: []models.Education{{
				Name:    "Bananenplukker",
				EndDate: time.Now().AddDate(-2, 0, 0).Format(time.RFC3339),
			}},
		},
	)
}

func TestMatchDesiredProfession(t *testing.T) {
	// Match on desired profession
	MustMatchSingle(
		t,
		models.Profile{
			MustDesiredProfession: true,
			DesiredProfessions:    []models.ProfileDesiredProfession{{Profession: models.Profession{Name: "Bananenplukker"}}},
		},
		models.Cv{PreferredJobs: []string{"Bananenplukker"}},
	)

	// no desired profession match
	MustNotMatchSingle(
		t,
		models.Profile{
			MustDesiredProfession: true,
			DesiredProfessions:    []models.ProfileDesiredProfession{{Profession: models.Profession{Name: "Real gangster"}}},
		},
		models.Cv{PreferredJobs: []string{"Bananenplukker"}},
	)
}

func TestMatchDesiredProfessionExperienced(t *testing.T) {
	// Match on profession experienced
	MustMatchSingle(
		t,
		models.Profile{
			MustExpProfession:     true,
			ProfessionExperienced: []models.ProfileProfessionExperience{{Profession: models.Profession{Name: "Bananenplukker"}}},
		},
		models.Cv{WorkExperiences: []models.WorkExperience{{Profession: "Bananenplukker"}}},
	)

	// No profession experienced match
	MustNotMatchSingle(
		t,
		models.Profile{
			MustExpProfession:     true,
			ProfessionExperienced: []models.ProfileProfessionExperience{{Profession: models.Profession{Name: "Real gangster stuff"}}},
		},
		models.Cv{WorkExperiences: []models.WorkExperience{{Profession: "Bananenplukker"}}},
	)
}

func TestMatchYearsSinceWork(t *testing.T) {
	yearsSinceWork := 2
	MustMatchSingle(
		t,
		models.Profile{YearsSinceWork: &yearsSinceWork},
		models.Cv{WorkExperiences: []models.WorkExperience{{EndDate: time.Now().AddDate(-1, 0, 0).Format(time.RFC3339)}}},
	)

	yearsSinceWork = 1
	MustNotMatchSingle(
		t,
		models.Profile{YearsSinceWork: &yearsSinceWork},
		models.Cv{WorkExperiences: []models.WorkExperience{{EndDate: time.Now().AddDate(-2, 0, 0).Format(time.RFC3339)}}},
	)
}

func TestMatchDriversLicense(t *testing.T) {
	// Match on drivers license
	MustMatchSingle(
		t,
		models.Profile{
			MustDriversLicense: true,
			DriversLicenses:    []models.ProfileDriversLicense{{DriversLicense: models.DriversLicense{Name: "A"}}},
		},
		models.Cv{DriversLicenses: []string{"A"}},
	)

	// No drivers license match
	MustNotMatchSingle(
		t,
		models.Profile{
			MustDriversLicense: true,
			DriversLicenses:    []models.ProfileDriversLicense{{DriversLicense: models.DriversLicense{Name: "A"}}},
		},
		models.Cv{DriversLicenses: []string{"B"}},
	)
}
