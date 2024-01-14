package crew

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/models"
)

func TestGetFilmDirectors(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Id", "Name", "Photo"})

	expect := []models.CrewItem{
		{Id: 1, Name: "n1", Photo: "p1"},
	}

	for _, item := range expect {
		rows = rows.AddRow(item.Id, item.Name, item.Photo)
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT crew.id, name, photo  FROM crew JOIN person_in_film ON crew.id = person_in_film.id_person WHERE id_film = $1 AND id_profession = (SELECT id FROM profession WHERE title = 'режиссёр')")).
		WithArgs(1).
		WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	directors, err := repo.GetFilmDirectors(1)
	if err != nil {
		t.Errorf("GetFilm error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	if !reflect.DeepEqual(directors, expect) {
		t.Errorf("results not match, want %v, have %v", expect, directors)
		return
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT crew.id, name, photo  FROM crew JOIN person_in_film ON crew.id = person_in_film.id_person WHERE id_film = $1 AND id_profession = (SELECT id FROM profession WHERE title = 'режиссёр')")).
		WithArgs(1).
		WillReturnError(fmt.Errorf("db_error"))

	directors, err = repo.GetFilmDirectors(1)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	if directors != nil {
		t.Errorf("get comments error, comments should be nil")
	}
}

func TestGetFilmScenarists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Id", "Name", "Photo"})

	expect := []models.CrewItem{
		{Id: 1, Name: "n1", Photo: "p1"},
	}

	for _, item := range expect {
		rows = rows.AddRow(item.Id, item.Name, item.Photo)
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT crew.id, name, photo  FROM crew JOIN person_in_film ON crew.id = person_in_film.id_person WHERE id_film = $1 AND id_profession = (SELECT id FROM profession WHERE title = 'сценарист')")).
		WithArgs(1).
		WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	scenarists, err := repo.GetFilmScenarists(1)
	if err != nil {
		t.Errorf("GetFilm error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	if !reflect.DeepEqual(scenarists, expect) {
		t.Errorf("results not match, want %v, have %v", expect, scenarists)
		return
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT crew.id, name, photo  FROM crew JOIN person_in_film ON crew.id = person_in_film.id_person WHERE id_film = $1 AND id_profession = (SELECT id FROM profession WHERE title = 'сценарист')")).
		WithArgs(1).
		WillReturnError(fmt.Errorf("db_error"))

	scenarists, err = repo.GetFilmScenarists(1)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	if scenarists != nil {
		t.Errorf("get comments error, comments should be nil")
	}
}

func TestGetFilmCharacters(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Id", "Name", "Photo", "CharacterName"})

	expect := []models.Character{
		{IdActor: 1, NameActor: "n1", ActorPhoto: "p1", NameCharacter: "chn1"},
	}

	for _, item := range expect {
		rows = rows.AddRow(item.IdActor, item.NameActor, item.ActorPhoto, item.NameCharacter)
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT crew.id, name, photo, person_in_film.character_name FROM crew JOIN person_in_film ON crew.id = person_in_film.id_person WHERE id_film = $1 AND id_profession = (SELECT id FROM profession WHERE title = 'актёр')")).
		WithArgs(1).
		WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	characters, err := repo.GetFilmCharacters(1)
	if err != nil {
		t.Errorf("GetFilm error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	if !reflect.DeepEqual(characters, expect) {
		t.Errorf("results not match, want %v, have %v", expect, characters)
		return
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT crew.id, name, photo, person_in_film.character_name FROM crew JOIN person_in_film ON crew.id = person_in_film.id_person WHERE id_film = $1 AND id_profession = (SELECT id FROM profession WHERE title = 'актёр')")).
		WithArgs(1).
		WillReturnError(fmt.Errorf("db_error"))

	characters, err = repo.GetFilmCharacters(1)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	if characters != nil {
		t.Errorf("get comments error, comments should be nil")
	}
}

func TestGetActor(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Id", "Name", "Birthdate", "Photo", "Info"})

	expect := []models.CrewItem{
		{Id: 1, Name: "n1", Birthdate: "2003", Photo: "p1", Info: "i1"},
	}

	for _, item := range expect {
		rows = rows.AddRow(item.Id, item.Name, item.Birthdate, item.Photo, item.Info)
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT id, name, birth_date, photo, info FROM crew WHERE id = $1")).
		WithArgs(1).
		WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	actor, err := repo.GetActor(1)
	if err != nil {
		t.Errorf("GetFilm error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	if !reflect.DeepEqual(actor, &expect[0]) {
		t.Errorf("results not match, want %v, have %v", &expect[0], actor)
		return
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT id, name, birth_date, photo, info FROM crew WHERE id = $1")).
		WithArgs(1).
		WillReturnError(fmt.Errorf("db_error"))

	actor, err = repo.GetActor(1)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	if actor != nil {
		t.Errorf("get comments error, comments should be nil")
	}
}

func TestCheckActor(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Id"})

	expect := []models.Character{
		{IdActor: 1},
	}

	for _, item := range expect {
		rows = rows.AddRow(item.IdActor)
	}
	selectRow := "SELECT id_actor FROM users_favorite_actor WHERE id_actor = $1 AND id_user = $2"

	mock.ExpectQuery(
		regexp.QuoteMeta(selectRow)).
		WithArgs(1, 1).
		WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	found, err := repo.CheckActor(1, 1)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
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

	found, err = repo.CheckActor(1, 1)
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

func TestGetFavoriteActors(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Name", "Id", "Photo"})

	expect := []models.Character{
		{IdActor: 1, NameActor: "t1", ActorPhoto: "url1"},
	}

	for _, item := range expect {
		rows = rows.AddRow(item.NameActor, item.IdActor, item.ActorPhoto)
	}
	selectRow := "SELECT crew.name, crew.id, crew.photo FROM crew JOIN users_favorite_actor ON crew.id = users_favorite_actor.id_actor WHERE id_user = $1 OFFSET $2 LIMIT $3"

	mock.ExpectQuery(
		regexp.QuoteMeta(selectRow)).
		WithArgs(1, 1, 2).
		WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	films, err := repo.GetFavoriteActors(1, 1, 2)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
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

	_, err = repo.GetFavoriteActors(1, 1, 2)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
}

func TestAddFavoriteActor(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	selectRow := "INSERT INTO users_favorite_actor(id_user, id_actor) VALUES ($1, $2)"

	mock.ExpectExec(
		regexp.QuoteMeta(selectRow)).
		WithArgs(1, 1).WillReturnResult(sqlmock.NewResult(0, 1))

	repo := &RepoPostgre{
		db: db,
	}

	err = repo.AddFavoriteActor(1, 1)
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

	err = repo.AddFavoriteActor(1, 1)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
}

func TestRemoveFavoriteActor(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	selectRow := "DELETE FROM users_favorite_actor WHERE id_user = $1 AND id_actor = $2"

	mock.ExpectExec(
		regexp.QuoteMeta(selectRow)).
		WithArgs(1, 1).WillReturnResult(sqlmock.NewResult(0, 1))

	repo := &RepoPostgre{
		db: db,
	}

	err = repo.RemoveFavoriteActor(1, 1)
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

	err = repo.RemoveFavoriteActor(1, 1)
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

	selectRow := "INSERT INTO person_in_film(id_film, id_person, id_profession, character_name) VALUES($1, $2, 1, '')"

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
