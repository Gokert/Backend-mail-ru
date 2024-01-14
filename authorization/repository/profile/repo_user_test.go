package profile

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/models"
)

func TestGetUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"photo", "login"})

	testUser := models.UserItem{
		Photo: "url1",
		Login: "l1",
	}
	expect := []*models.UserItem{&testUser}

	for _, item := range expect {
		rows = rows.AddRow(item.Login, item.Photo)
	}

	mock.ExpectQuery("SELECT login, photo FROM profile WHERE").WithArgs(expect[0].Login, expect[0].Password).WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	user, foundAccount, err := repo.GetUser(expect[0].Login, expect[0].Password)
	if err != nil {
		t.Errorf("GetUser error: %s", err)
	}
	if !foundAccount {
		t.Errorf("user not found")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	if !reflect.DeepEqual(user, expect[0]) {
		t.Errorf("results not match, want %v, have %v", expect[0], user)
		return
	}

	mock.
		ExpectQuery("SELECT login, photo FROM profile WHERE").
		WithArgs(expect[0].Login, expect[0].Password).
		WillReturnError(fmt.Errorf("db_error"))

	_, found, err := repo.GetUser(expect[0].Login, expect[0].Password)
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
	}
}

func TestFindUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"login"})

	testUser := models.UserItem{
		Login: "l1",
	}
	expect := []*models.UserItem{&testUser}

	for _, item := range expect {
		rows = rows.AddRow(item.Login)
	}

	mock.ExpectQuery("SELECT login FROM profile WHERE").WithArgs(expect[0].Login).WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	foundAccount, err := repo.FindUser(expect[0].Login)
	if err != nil {
		t.Errorf("GetUser error: %s", err)
	}
	if !foundAccount {
		t.Errorf("user not found")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	mock.
		ExpectQuery("SELECT login FROM profile WHERE").
		WithArgs(expect[0].Login).
		WillReturnError(fmt.Errorf("db_error"))

	found, err := repo.FindUser(expect[0].Login)
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
	}
}

func TestCreateUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"login"})

	testUser := models.UserItem{
		Login:     "l1",
		Password:  "p1",
		Birthdate: "2003-10-08",
		Name:      "n1",
		Email:     "e1",
	}
	expect := []*models.UserItem{&testUser}

	for _, item := range expect {
		rows = rows.AddRow(item.Login)
	}

	mock.
		ExpectExec("INSERT INTO profile").
		WithArgs(testUser.Name, testUser.Birthdate, testUser.Login, testUser.Password, testUser.Email).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := &RepoPostgre{
		db: db,
	}

	err = repo.CreateUser(testUser.Login, testUser.Password, testUser.Name, testUser.Birthdate, testUser.Email)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	mock.
		ExpectExec("INSERT INTO profile").
		WithArgs(testUser.Name, testUser.Birthdate, testUser.Login, testUser.Password, testUser.Email).
		WillReturnError(fmt.Errorf("db_error"))

	err = repo.CreateUser(testUser.Login, testUser.Password, testUser.Name, testUser.Birthdate, testUser.Email)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
}

func TestEditProfile(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"login"})

	prev := "l0"
	testUser := models.UserItem{
		Login:     "l1",
		Password:  "p1",
		Birthdate: "2003-10-08",
		Email:     "e1",
		Photo:     "ph1",
	}
	expect := []*models.UserItem{&testUser}

	for _, item := range expect {
		rows = rows.AddRow(item.Login)
	}

	mock.ExpectExec(
		regexp.QuoteMeta("UPDATE profile SET login = $1, photo = $2, email = $3, password = $4, birth_date = $5 WHERE login = $6")).
		WithArgs(testUser.Login, testUser.Photo, testUser.Email, testUser.Password, testUser.Birthdate, prev).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := &RepoPostgre{
		db: db,
	}

	err = repo.EditProfile(prev, testUser.Login, testUser.Password, testUser.Email, testUser.Birthdate, testUser.Photo)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	mock.ExpectExec(
		regexp.QuoteMeta("UPDATE profile SET login = $1, photo = $2, email = $3, password = $4, birth_date = $5 WHERE login = $6")).
		WithArgs(testUser.Login, testUser.Password, testUser.Photo, testUser.Email, testUser.Birthdate, prev).
		WillReturnError(fmt.Errorf("db_error"))

	err = repo.EditProfile(prev, testUser.Login, testUser.Password, testUser.Email, testUser.Birthdate, testUser.Photo)
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
}
