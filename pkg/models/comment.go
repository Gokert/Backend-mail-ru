package models

//easyjson:json
type CommentItem struct {
	IdUser   uint64 `json:"id_user"`
	Username string `json:"name"`
	IdFilm   uint64 `json:"id_film"`
	Rating   uint16 `json:"rating"`
	Comment  string `json:"text"`
	Photo    string `json:"photo"`
}
