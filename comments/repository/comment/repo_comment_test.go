package comment

import (
	"database/sql"
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/models"
)

func TestGetFilmComments(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Id", "Rating", "Comment"})

	expect := []models.CommentItem{
		{IdUser: 1, Rating: 4, Comment: "c1"},
	}

	for _, item := range expect {
		rows = rows.AddRow(item.IdUser, item.Rating, item.Comment)
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT id_user, rating, comment FROM users_comment WHERE id_film = $1 OFFSET $2 LIMIT $3")).
		WithArgs(1, 0, 5).
		WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	comments, err := repo.GetFilmComments(1, 0, 5)
	if err != nil {
		t.Errorf("GetFilm error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	if !reflect.DeepEqual(comments, expect) {
		t.Errorf("results not match, want %v, have %v", expect, comments)
		return
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT id_user, rating, comment FROM users_comment WHERE id_film = $1 OFFSET $2 LIMIT $3")).
		WithArgs(1, 0, 5).
		WillReturnError(fmt.Errorf("db_error"))

	comments, err = repo.GetFilmComments(1, 0, 5)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	if comments != nil {
		t.Errorf("get comments error, comments should be nil")
	}
}

func TestAddComment(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	testComment := models.CommentItem{
		IdFilm:  1,
		Rating:  1,
		Comment: "c1",
	}
	idUser := uint64(1)

	sqlQuery := "INSERT INTO users_comment(id_film, rating, comment, id_user) VALUES($1, $2, $3, $4)"

	mock.ExpectExec(
		regexp.QuoteMeta(sqlQuery)).
		WithArgs(testComment.IdFilm, testComment.Rating, testComment.Comment, idUser).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := &RepoPostgre{
		db: db,
	}

	err = repo.AddComment(testComment.IdFilm, idUser, testComment.Rating, testComment.Comment)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	mock.ExpectExec(
		regexp.QuoteMeta(sqlQuery)).
		WithArgs(testComment.IdFilm, testComment.Rating, testComment.Comment, idUser).
		WillReturnError(fmt.Errorf("db_error"))

	err = repo.AddComment(testComment.IdFilm, idUser, testComment.Rating, testComment.Comment)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
}

func TestHasUsersComment(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	idFilm := uint64(1)
	idUser := uint64(1)

	rows := sqlmock.NewRows([]string{"Id"})
	rows = rows.AddRow(idUser)

	sqlQuery := "SELECT id_user FROM users_comment WHERE id_user = $1 AND id_film = $2"

	mock.ExpectQuery(
		regexp.QuoteMeta(sqlQuery)).
		WithArgs(idUser, idFilm).
		WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	found, err := repo.HasUsersComment(idUser, idFilm)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if !found {
		t.Errorf("waited to find comment")
		return
	}

	mock.ExpectQuery(
		regexp.QuoteMeta(sqlQuery)).
		WithArgs(idUser, idFilm).
		WillReturnError(fmt.Errorf("db_error"))

	found, err = repo.HasUsersComment(idUser, idFilm)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	if found {
		t.Errorf("waited not to find comment")
		return
	}

	mock.ExpectQuery(
		regexp.QuoteMeta(sqlQuery)).
		WithArgs(idUser, idFilm).
		WillReturnError(sql.ErrNoRows)

	found, err = repo.HasUsersComment(idUser, idFilm)
	if err != nil {
		t.Errorf("waited no errors")
		return
	}
	if found {
		t.Errorf("waited not to find")
		return
	}
}

func TestDeleteComment(t *testing.T) {
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

	err = repo.DeleteComment(1, 1)
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

	err = repo.DeleteComment(1, 1)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
}
