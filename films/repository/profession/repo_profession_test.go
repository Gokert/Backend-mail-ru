package profession

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/models"
)

func TestGetActorProfessions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Title"})

	expect := []models.ProfessionItem{
		{Title: "p1"},
	}

	for _, item := range expect {
		rows = rows.AddRow(item.Title)
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT DISTINCT title FROM profession JOIN person_in_film ON profession.id = person_in_film.id_profession WHERE id_person = $1")).
		WithArgs(1).
		WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	career, err := repo.GetActorsProfessions(1)
	if err != nil {
		t.Errorf("GetFilm error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	if !reflect.DeepEqual(career, expect) {
		t.Errorf("results not match, want %v, have %v", expect, career)
		return
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT DISTINCT title FROM profession JOIN person_in_film ON profession.id = person_in_film.id_profession WHERE id_person = $1")).
		WithArgs(1).
		WillReturnError(fmt.Errorf("db_error"))

	career, err = repo.GetActorsProfessions(1)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	if career != nil {
		t.Errorf("get comments error, comments should be nil")
	}
}
