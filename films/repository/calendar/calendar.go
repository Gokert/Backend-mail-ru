package calendar

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-park-mail-ru/2023_2_Vkladyshi/configs"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/models"

	_ "github.com/jackc/pgx/stdlib"
)

//go:generate mockgen -source=calendar.go -destination=../../mocks/calendar_repo_mock.go -package=mocks

type ICalendarRepo interface {
	GetCalendar() ([]models.DayItem, error)
}

type RepoPostgre struct {
	db *sql.DB
}

func GetCalendarRepo(config *configs.DbDsnCfg, lg *slog.Logger) (*RepoPostgre, error) {
	dsn := fmt.Sprintf("user=%s dbname=%s password= %s host=%s port=%d sslmode=%s",
		config.User, config.DbName, config.Password, config.Host, config.Port, config.Sslmode)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		lg.Error("sql open error", "err", err.Error())
		return nil, fmt.Errorf("get calendar repo: %w", err)
	}
	err = db.Ping()
	if err != nil {
		lg.Error("sql ping error", "err", err.Error())
		return nil, fmt.Errorf("get calendar repo: %w", err)
	}
	db.SetMaxOpenConns(config.MaxOpenConns)

	postgreDb := RepoPostgre{db: db}

	go postgreDb.pingDb(config.Timer, lg)
	return &postgreDb, nil
}

func (repo *RepoPostgre) pingDb(timer uint32, lg *slog.Logger) {
	for {
		err := repo.db.Ping()
		if err != nil {
			lg.Error("Repo Crew db ping error", "err", err.Error())
		}

		time.Sleep(time.Duration(timer) * time.Second)
	}
}

func (repo *RepoPostgre) GetCalendar() ([]models.DayItem, error) {
	calendar := []models.DayItem{}
	lastAppendDay := uint8(0)
	news := ""

	rows, err := repo.db.Query("SELECT film.title, release_day, film.poster, film.id FROM calendar " +
		"JOIN film ON film.id = calendar.id " +
		"WHERE release_month = DATE_PART('MONTH', CURRENT_DATE) " +
		"ORDER BY release_day")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	defer rows.Close()

	post1 := models.DayItem{}
	for rows.Next() {
		post2 := models.DayItem{}
		err := rows.Scan(&post2.DayNews, &post2.DayNumber, &post2.Poster, &post2.IdFilm)
		if err != nil {
			return nil, fmt.Errorf("get calendar scan err: %w", err)
		}

		if post1.DayNumber == 0 {
			post1 = post2
			continue
		}

		if post1.DayNumber != post2.DayNumber {
			news += post1.DayNews
			lastAppendDay = post1.DayNumber

			calendar = append(calendar, models.DayItem{DayNumber: lastAppendDay, DayNews: news, IdFilm: post1.IdFilm, Poster: post1.Poster})
			news = ""
		} else {
			news += post1.DayNews + " "
		}
		post1 = post2
	}
	if lastAppendDay != post1.DayNumber {
		calendar = append(calendar, models.DayItem{DayNumber: post1.DayNumber, DayNews: news + post1.DayNews, IdFilm: post1.IdFilm, Poster: post1.Poster})
	}

	return calendar, nil
}
