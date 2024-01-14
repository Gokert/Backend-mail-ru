package models

//easyjson:json
type FilmItem struct {
	Id          uint64  `json:"id"`
	Title       string  `json:"title"`
	Info        string  `json:"info"`
	Poster      string  `json:"poster"`
	ReleaseDate string  `json:"release_date"`
	Country     string  `json:"country"`
	Mpaa        string  `json:"mpaa"`
	Rating      float64 `json:"rating"`
}

type NearFilm struct {
	IdFilm uint64
	IdUser uint64
}