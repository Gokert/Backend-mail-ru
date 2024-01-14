package genre

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2023_2_Vkladyshi/configs"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/models"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/requests"

	_ "github.com/jackc/pgx/stdlib"
)

//go:generate mockgen -source=repo_genre.go -destination=../../mocks/genre_repo_mock.go -package=mocks

type IGenreRepo interface {
	GetFilmGenres(filmId uint64) ([]models.GenreItem, error)
	GetGenreById(genreId uint64) (string, error)
	AddFilm(genres []uint64, filmId uint64) error
	UsersStatistics(idUser uint64) ([]requests.UsersStatisticsResponse, error)
}

type RepoPostgre struct {
	db *sql.DB
}

func GetGenreRepo(config *configs.DbDsnCfg, lg *slog.Logger) (*RepoPostgre, error) {
	dsn := fmt.Sprintf("user=%s dbname=%s password= %s host=%s port=%d sslmode=%s",
		config.User, config.DbName, config.Password, config.Host, config.Port, config.Sslmode)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		lg.Error("sql open error", "err", err.Error())
		return nil, fmt.Errorf("get genre repo: %w", err)
	}
	err = db.Ping()
	if err != nil {
		lg.Error("sql ping error", "err", err.Error())
		return nil, fmt.Errorf("get genre repo: %w", err)
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
			lg.Error("Repo Genre db ping error", "err", err.Error())
		}

		time.Sleep(time.Duration(timer) * time.Second)
	}
}

func (repo *RepoPostgre) GetFilmGenres(filmId uint64) ([]models.GenreItem, error) {
	genres := []models.GenreItem{}

	rows, err := repo.db.Query(
		"SELECT genre.id, genre.title FROM genre "+
			"JOIN films_genre ON genre.id = films_genre.id_genre "+
			"WHERE films_genre.id_film = $1", filmId)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("GetFilmGenres err: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		post := models.GenreItem{}
		err := rows.Scan(&post.Id, &post.Title)
		if err != nil {
			return nil, fmt.Errorf("GetFilmGenres scan err: %w", err)
		}
		genres = append(genres, post)
	}

	return genres, nil
}

func (repo *RepoPostgre) GetGenreById(genreId uint64) (string, error) {
	var genre string

	err := repo.db.QueryRow(
		"SELECT title FROM genre "+
			"WHERE id = $1", genreId).Scan(&genre)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}

	return genre, nil
}

func (repo *RepoPostgre) AddFilm(genres []uint64, filmId uint64) error {
	var s strings.Builder
	var params []interface{}
	params = append(params, filmId)

	s.WriteString("INSERT INTO films_genre(id_film, id_genre) VALUES")
	for i, genre := range genres {
		if i != 0 {
			s.WriteString(",")
		}
		s.WriteString("($1, $" + strconv.Itoa(i+2) + ")")
		params = append(params, genre)
	}

	_, err := repo.db.Exec(s.String(), params...)
	if err != nil {
		return fmt.Errorf("add films genres error: %w", err)
	}
	return nil
}

func (repo *RepoPostgre) UsersStatistics(idUser uint64) ([]requests.UsersStatisticsResponse, error) {
	response := []requests.UsersStatisticsResponse{}

	rows, err := repo.db.Query("SELECT genre.id, AVG(rating), COUNT(rating) FROM film "+
		"JOIN users_comment ON film.id = users_comment.id_film "+
		"JOIN films_genre ON film.id = films_genre.id_film "+
		"JOIN genre ON genre.id = films_genre.id_genre "+
		"WHERE id_user = $1 "+
		"GROUP BY genre.id", idUser)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("users stats err: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		post := requests.UsersStatisticsResponse{}
		err := rows.Scan(&post.GenreId, &post.Avg, &post.Count)
		if err != nil {
			return nil, fmt.Errorf("users stats scan err: %w", err)
		}
		response = append(response, post)
	}

	return response, nil
}
