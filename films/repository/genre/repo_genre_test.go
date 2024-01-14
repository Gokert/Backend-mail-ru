package genre

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/models"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/requests"
)

func TestGetFilmGenres(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Id", "Title"})

	expect := []models.GenreItem{
		{Id: 1, Title: "t1"},
	}

	for _, item := range expect {
		rows = rows.AddRow(item.Id, item.Title)
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT genre.id, genre.title FROM genre JOIN films_genre ON genre.id = films_genre.id_genre WHERE films_genre.id_film = $1")).
		WithArgs(1).
		WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	genres, err := repo.GetFilmGenres(1)
	if err != nil {
		t.Errorf("GetFilm error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	if !reflect.DeepEqual(genres, expect) {
		t.Errorf("results not match, want %v, have %v", expect, genres)
		return
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT genre.id, genre.title FROM genre JOIN films_genre ON genre.id = films_genre.id_genre WHERE films_genre.id_film = $1")).
		WithArgs(1).
		WillReturnError(fmt.Errorf("db_error"))

	genres, err = repo.GetFilmGenres(1)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	if genres != nil {
		t.Errorf("get comments error, comments should be nil")
	}
}

func TestGetGenreById(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Title"})

	expect := "genre1"

	rows = rows.AddRow(expect)

	selectStr := "SELECT title FROM genre WHERE id = $1"

	mock.ExpectQuery(
		regexp.QuoteMeta(selectStr)).
		WithArgs(1).
		WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	genre, err := repo.GetGenreById(1)
	if err != nil {
		t.Errorf("GetFilm error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	if !reflect.DeepEqual(genre, expect) {
		t.Errorf("results not match, want %v, have %v", expect, genre)
		return
	}

	mock.ExpectQuery(
		regexp.QuoteMeta(selectStr)).
		WithArgs(1).
		WillReturnError(fmt.Errorf("db_error"))

	genre, err = repo.GetGenreById(1)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	if genre != "" {
		t.Errorf("get comments error, comments should be nil")
	}
}

func TestAddFilm(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	selectRow := "INSERT INTO films_genre(id_film, id_genre) VALUES($1, $2)"

	mock.ExpectExec(
		regexp.QuoteMeta(selectRow)).
		WithArgs(1, 1).WillReturnResult(sqlmock.NewResult(0, 1))

	repo := &RepoPostgre{
		db: db,
	}

	err = repo.AddFilm([]uint64{1}, 1)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	mock.ExpectExec(
		regexp.QuoteMeta(selectRow)).
		WithArgs(1, 1).WillReturnError(fmt.Errorf("repo err"))

	err = repo.AddFilm([]uint64{1}, 1)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
}

func TestUsersStatistics(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Id", "AVG", "Count"})

	expect := []requests.UsersStatisticsResponse{{GenreId: 1, Avg: 1, Count: 1}}

	for _, item := range expect {
		rows = rows.AddRow(item.GenreId, item.Avg, item.Count)
	}

	selectStr := "SELECT genre.id, AVG(rating), COUNT(rating) FROM film JOIN users_comment ON film.id = users_comment.id_film JOIN films_genre ON film.id = films_genre.id_film JOIN genre ON genre.id = films_genre.id_genre WHERE id_user = $1 GROUP BY genre.id"

	mock.ExpectQuery(
		regexp.QuoteMeta(selectStr)).
		WithArgs(1).
		WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	res, err := repo.UsersStatistics(1)
	if err != nil {
		t.Errorf("GetFilm error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	if !reflect.DeepEqual(res, expect) {
		t.Errorf("results not match, want %v, have %v", expect, res)
		return
	}

	mock.ExpectQuery(
		regexp.QuoteMeta(selectStr)).
		WithArgs(1).
		WillReturnError(fmt.Errorf("db_error"))

	res, err = repo.UsersStatistics(1)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	if res != nil {
		t.Errorf("expected error, got nil")
		return
	}
}
