package entity

import (
	"github.com/photoprism/photoprism/pkg/s2"
)

type PlacesMap map[string]Place

func (m PlacesMap) Get(name string) Place {
	if result, ok := m[name]; ok {
		return result
	}

	return UnknownPlace
}

func (m PlacesMap) Pointer(name string) *Place {
	if result, ok := m[name]; ok {
		return &result
	}

	return &UnknownPlace
}

var PlaceFixtures = PlacesMap{
	"mexico": {
		ID:          s2.TokenPrefix + "85d1ea7d3278",
		LocLabel:    "Teotihuacán, Mexico, Mexico",
		LocCity:     "Teotihuacán",
		LocState:    "State of Mexico",
		LocCountry:  "mx",
		LocKeywords: "ancient, pyramid",
		LocNotes:    "",
		LocFavorite: false,
		PhotoCount:  1,
		CreatedAt:   Timestamp(),
		UpdatedAt:   Timestamp(),
	},
	"zinkwazi": {
		ID:          s2.TokenPrefix + "1ef744d1e279",
		LocLabel:    "KwaDukuza, KwaZulu-Natal, South Africa",
		LocCity:     "KwaDukuza",
		LocState:    "KwaZulu-Natal",
		LocCountry:  "za",
		LocKeywords: "",
		LocNotes:    "africa",
		LocFavorite: true,
		PhotoCount:  2,
		CreatedAt:   Timestamp(),
		UpdatedAt:   Timestamp(),
	},
	"holidaypark": {
		ID:          s2.TokenPrefix + "1ef744d1e280",
		LocLabel:    "Holiday Park, Amusement",
		LocCity:     "",
		LocState:    "Rheinland-Pfalz",
		LocCountry:  "de",
		LocKeywords: "",
		LocNotes:    "germany",
		LocFavorite: true,
		PhotoCount:  2,
		CreatedAt:   Timestamp(),
		UpdatedAt:   Timestamp(),
	},
	"emptyNameLongCity": {
		ID:          s2.TokenPrefix + "1ef744d1e281",
		LocLabel:    "labelEmptyNameLongCity",
		LocCity:     "longlonglonglonglongcity",
		LocState:    "Rheinland-Pfalz",
		LocCountry:  "de",
		LocKeywords: "",
		LocNotes:    "germany",
		LocFavorite: true,
		PhotoCount:  2,
		CreatedAt:   Timestamp(),
		UpdatedAt:   Timestamp(),
	},
	"emptyNameShortCity": {
		ID:          s2.TokenPrefix + "1ef744d1e282",
		LocLabel:    "labelEmptyNameShortCity",
		LocCity:     "shortcity",
		LocState:    "Rheinland-Pfalz",
		LocCountry:  "de",
		LocKeywords: "",
		LocNotes:    "germany",
		LocFavorite: true,
		PhotoCount:  2,
		CreatedAt:   Timestamp(),
		UpdatedAt:   Timestamp(),
	},
	"veryLongLocName": {
		ID:          s2.TokenPrefix + "1ef744d1e283",
		LocLabel:    "labelVeryLongLocName",
		LocCity:     "Mainz",
		LocState:    "Rheinland-Pfalz",
		LocCountry:  "de",
		LocKeywords: "",
		LocNotes:    "germany",
		LocFavorite: true,
		PhotoCount:  2,
		CreatedAt:   Timestamp(),
		UpdatedAt:   Timestamp(),
	},
	"mediumLongLocName": {
		ID:          s2.TokenPrefix + "1ef744d1e284",
		LocLabel:    "labelMediumLongLocName",
		LocCity:     "New york",
		LocState:    "New york",
		LocCountry:  "us",
		LocKeywords: "",
		LocNotes:    "",
		LocFavorite: true,
		PhotoCount:  2,
		CreatedAt:   Timestamp(),
		UpdatedAt:   Timestamp(),
	},
}

// CreatePlaceFixtures inserts known entities into the database for testing.
func CreatePlaceFixtures() {
	for _, entity := range PlaceFixtures {
		Db().Create(&entity)
	}
}
