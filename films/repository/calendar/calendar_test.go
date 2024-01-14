package calendar

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/models"
)

func TestGetCalendar(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Title", "RealeseDay", "Poster", "Id"})

	expect := []models.DayItem{
		{DayNews: "n1", DayNumber: 1, IdFilm: 1, Poster: "p"},
	}

	for _, item := range expect {
		rows = rows.AddRow(item.DayNews, item.DayNumber, item.Poster, item.IdFilm)
	}

	selectRow := "SELECT film.title, release_day, film.poster, film.id FROM calendar JOIN film ON film.id = calendar.id WHERE release_month = DATE_PART('MONTH', CURRENT_DATE) ORDER BY release_day"

	mock.ExpectQuery(
		regexp.QuoteMeta(selectRow)).
		WithArgs().
		WillReturnRows(rows)

	repo := &RepoPostgre{
		db: db,
	}

	days, err := repo.GetCalendar()
	if err != nil {
		t.Errorf("get calendar error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	if !reflect.DeepEqual(days, expect) {
		t.Errorf("results not match, want %v, have %v", expect, days)
		return
	}

	mock.ExpectQuery(
		regexp.QuoteMeta(selectRow)).
		WithArgs().
		WillReturnError(fmt.Errorf("db_error"))

	days, err = repo.GetCalendar()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	if days != nil {
		t.Errorf("get calendar error, days should be nil")
	}
}
