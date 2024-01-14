package models

//easyjson:json
type DayItem struct {
	DayNumber uint8  `json:"dayNumber"`
	DayNews   string `json:"dayNews"`
	IdFilm    uint64 `json:"id"`
	Poster    string `json:"poster"`
}
