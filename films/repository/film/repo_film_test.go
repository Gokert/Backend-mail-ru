package film

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/models"
)

func TestGetFilmsByGenre(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Id", "Title", "Poster"})

	expect := []models.FilmItem{
		{Id: 1, Title: "t1", Poster: "url1"},
	}

	for _, item := range expect {
		rows = rows.AddRow(item.Id, item.Title, item.Poster)
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT film.id, film.title, poster FROM film  JOIN films_genre ON film.id = films_genre.id_film WHERE id_genre = $1 ORDER BY release_date DESC OFFSET $2 LIMIT $3")).
		WithArgs(1, 1, 2).
		WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	films, err := repo.GetFilmsByGenre(1, 1, 2)
	if err != nil {
		t.Errorf("GetFilmsByGenre error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	if !reflect.DeepEqual(films, expect) {
		t.Errorf("results not match, want %v, have %v", expect, films)
		return
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT film.id, film.title, poster FROM film  JOIN films_genre ON film.id = films_genre.id_film WHERE id_genre = $1 ORDER BY release_date DESC OFFSET $2 LIMIT $3")).
		WithArgs(1, 1, 2).
		WillReturnError(fmt.Errorf("db_error"))

	_, err = repo.GetFilmsByGenre(1, 1, 2)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
}

func TestGetFilms(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Id", "Title", "Poster"})

	expect := []models.FilmItem{
		{Id: 1, Title: "t1", Poster: "url1"},
	}

	for _, item := range expect {
		rows = rows.AddRow(item.Id, item.Title, item.Poster)
	}

	mock.ExpectQuery("SELECT film.id, film.title, poster FROM film").WithArgs(1, 2).WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	films, err := repo.GetFilms(1, 2)
	if err != nil {
		t.Errorf("GetFilms error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	if !reflect.DeepEqual(films, expect) {
		t.Errorf("results not match, want %v, have %v", expect, films)
		return
	}

	mock.
		ExpectQuery("SELECT film.id, film.title, poster FROM film").
		WithArgs(1, 2).
		WillReturnError(fmt.Errorf("db_error"))

	_, err = repo.GetFilms(1, 2)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
}

func TestGetFilm(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Id", "Title", "Info", "Poster", "ReleaseDate", "Country", "Mpaa"})

	expect := []models.FilmItem{
		{Id: 1, Title: "t1", Info: "i1", Poster: "url1", ReleaseDate: "date1", Country: "c1", Mpaa: "12"},
	}

	for _, item := range expect {
		rows = rows.AddRow(item.Id, item.Title, item.Info, item.Poster, item.ReleaseDate, item.Country, item.Mpaa)
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT id, title, info, poster, release_date, country, mpaa FROM film WHERE id = $1")).
		WithArgs(1).
		WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	films, err := repo.GetFilm(1)
	if err != nil {
		t.Errorf("GetFilm error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	if !reflect.DeepEqual(films, &expect[0]) {
		t.Errorf("results not match, want %v, have %v", expect, films)
		return
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT id, title, info, poster, release_date, country, mpaa FROM film WHERE id = $1")).
		WithArgs(1).
		WillReturnError(fmt.Errorf("db_error"))

	_, err = repo.GetFilm(1)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
}

func TestGetFilmRating(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Average", "Amount"})

	expectRating := 4.2
	expectAmount := uint64(3)

	rows = rows.AddRow(expectRating, expectAmount)

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT AVG(rating), COUNT(rating) FROM users_comment WHERE id_film")).
		WithArgs(1).
		WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	rating, number, err := repo.GetFilmRating(1)
	if err != nil {
		t.Errorf("GetFilm error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	if rating != expectRating {
		t.Errorf("results not match, want %v, have %v", expectRating, rating)
		return
	}
	if number != expectAmount {
		t.Errorf("results not match, want %v, have %v", expectAmount, number)
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT AVG(rating), COUNT(rating) FROM users_comment WHERE id_film")).
		WithArgs(1).
		WillReturnError(fmt.Errorf("db_error"))

	rating, number, err = repo.GetFilmRating(1)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	if rating != 0 {
		t.Errorf("expected rating 0, got %f", rating)
	}
	if number != 0 {
		t.Errorf("expected number 0, got %d", number)
	}
}

func TestFindFilm(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Title", "Id", "Poster", "Rating"})

	expectFilm := []models.FilmItem{
		{Id: 1, Title: "t1", Poster: "url1"},
	}
	expectRating := []float32{8}

	for _, item := range expectFilm {
		rows = rows.AddRow(item.Title, item.Id, item.Poster, expectRating[0])
	}

	selectStr := "SELECT DISTINCT film.title, film.id, film.poster, AVG(users_comment.rating) FROM film JOIN films_genre ON film.id = films_genre.id_film LEFT JOIN users_comment ON film.id = users_comment.id_film JOIN person_in_film ON film.id = person_in_film.id_film JOIN crew ON person_in_film.id_person = crew.id GROUP BY film.title, film.id HAVING (AVG(users_comment.rating) >= $1 AND AVG(users_comment.rating) <= $2) OR AVG(users_comment.rating) IS NULL ORDER BY film.title LIMIT $3 OFFSET $4"
	mock.ExpectQuery(
		regexp.QuoteMeta(selectStr)).
		WithArgs(float32(0), float32(10), uint64(1), uint64(0)).
		WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	film, err := repo.FindFilm("", "", "", float32(0), float32(10), "", []uint32{}, []string{""}, 0, 1)
	if err != nil {
		t.Errorf("GetFilm error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	if reflect.DeepEqual(film, expectFilm) {
		t.Errorf("film results not match, want %v, have %v", expectFilm, film)
	}

	mock.ExpectQuery(
		regexp.QuoteMeta(selectStr)).
		WithArgs(float32(0), float32(10), uint64(0), uint64(0)).
		WillReturnError(fmt.Errorf("db_error"))

	film, err = repo.FindFilm("", "", "", float32(0), float32(10), "", []uint32{0}, []string{""}, 0, 0)
	if err == mock.ExpectationsWereMet() {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}

	if film != nil {
		t.Errorf("expected film nil, got %v", film)
	}
}

func TestGetFavoriteFilms(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Title", "Id", "Poster"})

	expect := []models.FilmItem{
		{Id: 1, Title: "t1", Poster: "url1"},
	}

	for _, item := range expect {
		rows = rows.AddRow(item.Title, item.Id, item.Poster)
	}
	selectRow := "SELECT film.title, film.id, film.poster FROM film JOIN users_favorite_film ON film.id = users_favorite_film.id_film WHERE id_user = $1 OFFSET $2 LIMIT $3"

	mock.ExpectQuery(
		regexp.QuoteMeta(selectRow)).
		WithArgs(1, 1, 2).
		WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	films, err := repo.GetFavoriteFilms(1, 1, 2)
	if err != nil {
		t.Errorf("GetFilmsByGenre error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	if !reflect.DeepEqual(films, expect) {
		t.Errorf("results not match, want %v, have %v", expect, films)
		return
	}

	mock.ExpectQuery(
		regexp.QuoteMeta(selectRow)).
		WithArgs(1, 1, 2).
		WillReturnError(fmt.Errorf("db_error"))

	_, err = repo.GetFavoriteFilms(1, 1, 2)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
}

func TestCheckFilm(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Id"})

	expect := []models.FilmItem{
		{Id: 1},
	}

	for _, item := range expect {
		rows = rows.AddRow(item.Id)
	}
	selectRow := "SELECT id_film FROM users_favorite_film WHERE id_film = $1 AND id_user = $2"

	mock.ExpectQuery(
		regexp.QuoteMeta(selectRow)).
		WithArgs(1, 1).
		WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	found, err := repo.CheckFilm(1, 1)
	if err != nil {
		t.Errorf("GetFilmsByGenre error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	if !found {
		t.Errorf("expected found")
		return
	}

	mock.ExpectQuery(
		regexp.QuoteMeta(selectRow)).
		WithArgs(1, 1).
		WillReturnError(fmt.Errorf("db_error"))

	found, err = repo.CheckFilm(1, 1)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	if found {
		t.Errorf("expected not found")
		return
	}
}

func TestHasUsersRating(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Id"})

	expect := []models.FilmItem{
		{Id: 1},
	}

	for _, item := range expect {
		rows = rows.AddRow(item.Id)
	}
	selectRow := "SELECT id_user FROM users_comment WHERE id_user = $1 AND id_film = $2"

	mock.ExpectQuery(
		regexp.QuoteMeta(selectRow)).
		WithArgs(1, 1).
		WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	found, err := repo.HasUsersRating(1, 1)
	if err != nil {
		t.Errorf("GetFilmsByGenre error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	if !found {
		t.Errorf("expected found")
		return
	}

	mock.ExpectQuery(
		regexp.QuoteMeta(selectRow)).
		WithArgs(1, 1).
		WillReturnError(fmt.Errorf("db_error"))

	found, err = repo.HasUsersRating(1, 1)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	if found {
		t.Errorf("expected not found")
		return
	}
}

func TestAddFavoriteFilm(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	selectRow := "INSERT INTO users_favorite_film(id_user, id_film) VALUES ($1, $2)"

	mock.ExpectExec(
		regexp.QuoteMeta(selectRow)).
		WithArgs(1, 1).WillReturnResult(sqlmock.NewResult(0, 1))

	repo := &RepoPostgre{
		db: db,
	}

	err = repo.AddFavoriteFilm(1, 1)
	if err != nil {
		t.Errorf("GetFilmsByGenre error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	mock.ExpectExec(
		regexp.QuoteMeta(selectRow)).
		WithArgs(1, 1).WillReturnError(fmt.Errorf("repo err"))

	err = repo.AddFavoriteFilm(1, 1)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
}

func TestRemoveFavoriteFilm(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	selectRow := "DELETE FROM users_favorite_film WHERE id_user = $1 AND id_film = $2"

	mock.ExpectExec(
		regexp.QuoteMeta(selectRow)).
		WithArgs(1, 1).WillReturnResult(sqlmock.NewResult(0, 1))

	repo := &RepoPostgre{
		db: db,
	}

	err = repo.RemoveFavoriteFilm(1, 1)
	if err != nil {
		t.Errorf("GetFilmsByGenre error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	mock.ExpectExec(
		regexp.QuoteMeta(selectRow)).
		WithArgs(1, 1).WillReturnError(fmt.Errorf("repo err"))

	err = repo.RemoveFavoriteFilm(1, 1)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
}

func TestAddRating(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	selectRow := "INSERT INTO users_comment(id_film, rating, id_user) VALUES($1, $2, $3)"

	mock.ExpectExec(
		regexp.QuoteMeta(selectRow)).
		WithArgs(1, 5, 1).WillReturnResult(sqlmock.NewResult(0, 1))

	repo := &RepoPostgre{
		db: db,
	}

	err = repo.AddRating(1, 1, 5)
	if err != nil {
		t.Errorf("unexpected err: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	mock.ExpectExec(
		regexp.QuoteMeta(selectRow)).
		WithArgs(1, 1, 5).WillReturnError(fmt.Errorf("repo err"))

	err = repo.AddRating(1, 5, 1)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
}

func TestAddFilm(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	filmItem := models.FilmItem{
		Title:       "t",
		Info:        "i",
		Poster:      "p",
		ReleaseDate: "rd",
		Country:     "c",
		Mpaa:        "m",
	}
	selectRow := "INSERT INTO film(title, info, poster, release_date, country, mpaa) VALUES($1, $2, $3, $4, $5, $6)"

	mock.ExpectExec(
		regexp.QuoteMeta(selectRow)).
		WithArgs("t", "i", "p", "rd", "c", "m").WillReturnResult(sqlmock.NewResult(0, 1))

	repo := &RepoPostgre{
		db: db,
	}

	err = repo.AddFilm(filmItem)
	if err != nil {
		t.Errorf("unexpected err: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	mock.ExpectExec(
		regexp.QuoteMeta(selectRow)).
		WithArgs("t", "i", "p", "rd", "c", "m").WillReturnError(fmt.Errorf("repo err"))

	err = repo.AddFilm(filmItem)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
}

func TestGetFilmId(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Id"})

	expect := []models.FilmItem{
		{Id: 1, Title: "t"},
	}

	for _, item := range expect {
		rows = rows.AddRow(item.Id)
	}
	selectRow := "SELECT id FROM film WHERE title = $1"

	mock.ExpectQuery(
		regexp.QuoteMeta(selectRow)).
		WithArgs("t").
		WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	id, err := repo.GetFilmId(expect[0].Title)
	if err != nil {
		t.Errorf("GetFilmsByGenre error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	if id != expect[0].Id {
		t.Errorf("wanted %d, got %d", expect[0].Id, id)
		return
	}

	mock.ExpectQuery(
		regexp.QuoteMeta(selectRow)).
		WithArgs("t").
		WillReturnError(fmt.Errorf("db_error"))

	id, err = repo.GetFilmId(expect[0].Title)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	if id != 0 {
		t.Errorf("wanted 0, got %d", id)
		return
	}
}

func TestDeleteRating(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	selectRow := "DELETE FROM users_comment WHERE id_user = $1 AND id_film = $2"

	mock.ExpectExec(
		regexp.QuoteMeta(selectRow)).
		WithArgs(1, 1).WillReturnResult(sqlmock.NewResult(0, 1))

	repo := &RepoPostgre{
		db: db,
	}

	err = repo.DeleteRating(1, 1)
	if err != nil {
		t.Errorf("unexpected err: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	mock.ExpectExec(
		regexp.QuoteMeta(selectRow)).
		WithArgs(1, 1).WillReturnError(fmt.Errorf("repo err"))

	err = repo.DeleteRating(1, 1)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
}
