package film

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
	"github.com/lib/pq"

	_ "github.com/jackc/pgx/stdlib"
)

//go:generate mockgen -source=repo_film.go -destination=../../mocks/film_repo_mock.go -package=mocks
type IFilmsRepo interface {
	GetFilmsByGenre(genre uint64, start uint64, end uint64) ([]models.FilmItem, error)
	GetFilms(start uint64, end uint64) ([]models.FilmItem, error)
	GetFilm(filmId uint64) (*models.FilmItem, error)
	GetFilmRating(filmId uint64) (float64, uint64, error)
	FindFilm(title string, dateFrom string, dateTo string, ratingFrom float32, ratingTo float32,
		mpaa string, genres []uint32, actors []string, first uint64, limit uint64,
	) ([]models.FilmItem, error)
	GetFavoriteFilms(userId uint64, start uint64, end uint64) ([]models.FilmItem, error)
	AddFavoriteFilm(userId uint64, filmId uint64) error
	RemoveFavoriteFilm(userId uint64, filmId uint64) error
	CheckFilm(userId uint64, filmId uint64) (bool, error)
	AddRating(filmId uint64, userId uint64, rating uint16) error
	HasUsersRating(userId uint64, filmId uint64) (bool, error)
	AddFilm(film models.FilmItem) error
	GetFilmId(title string) (uint64, error)
	DeleteRating(idUser uint64, idFilm uint64) error
	Trends() ([]models.FilmItem, error)
	GetLasts(ids []uint64) ([]models.FilmItem, error)
}

type RepoPostgre struct {
	db *sql.DB
}

func GetFilmRepo(config *configs.DbDsnCfg, lg *slog.Logger) (*RepoPostgre, error) {
	dsn := fmt.Sprintf("user=%s dbname=%s password= %s host=%s port=%d sslmode=%s",
		config.User, config.DbName, config.Password, config.Host, config.Port, config.Sslmode)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		lg.Error("sql open error", "err", err.Error())
		return nil, fmt.Errorf("get film repo: %w", err)
	}
	err = db.Ping()
	if err != nil {
		lg.Error("sql ping error", "err", err.Error())
		return nil, fmt.Errorf("get film repo: %w", err)
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
			lg.Error("Repo Film db ping error", "err", err.Error())
		}

		time.Sleep(time.Duration(timer) * time.Second)
	}
}

func (repo *RepoPostgre) GetFilmsByGenre(genre uint64, start uint64, end uint64) ([]models.FilmItem, error) {
	films := make([]models.FilmItem, 0, end-start)

	rows, err := repo.db.Query(
		"SELECT film.id, film.title, poster FROM film "+
			"JOIN films_genre ON film.id = films_genre.id_film "+
			"WHERE id_genre = $1 "+
			"ORDER BY release_date DESC "+
			"OFFSET $2 LIMIT $3",
		genre, start, end)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("GetFilmsByGenre err: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		post := models.FilmItem{}
		err := rows.Scan(&post.Id, &post.Title, &post.Poster)
		if err != nil {
			return nil, fmt.Errorf("GetFilmsByGenre scan err: %w", err)
		}
		films = append(films, post)
	}

	return films, nil
}

func (repo *RepoPostgre) GetFilms(start uint64, end uint64) ([]models.FilmItem, error) {
	films := make([]models.FilmItem, 0, end-start)

	rows, err := repo.db.Query(
		"SELECT film.id, film.title, poster FROM film "+
			"ORDER BY release_date DESC "+
			"OFFSET $1 LIMIT $2",
		start, end)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("GetFilms err: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		post := models.FilmItem{}
		err := rows.Scan(&post.Id, &post.Title, &post.Poster)
		if err != nil {
			return nil, fmt.Errorf("GetFilms scan err: %w", err)
		}
		films = append(films, post)
	}

	return films, nil
}

func (repo *RepoPostgre) GetFilm(filmId uint64) (*models.FilmItem, error) {
	film := &models.FilmItem{}
	err := repo.db.QueryRow(
		"SELECT id, title, info, poster, release_date, country, mpaa FROM film "+
			"WHERE id = $1", filmId).
		Scan(&film.Id, &film.Title, &film.Info, &film.Poster, &film.ReleaseDate, &film.Country, &film.Mpaa)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return film, nil
		}

		return nil, fmt.Errorf("GetFilm err: %w", err)
	}

	return film, nil
}

func (repo *RepoPostgre) GetFilmRating(filmId uint64) (float64, uint64, error) {
	var rating sql.NullFloat64
	var number sql.NullInt64
	err := repo.db.QueryRow(
		"SELECT AVG(rating), COUNT(rating) FROM users_comment "+
			"WHERE id_film = $1", filmId).Scan(&rating, &number)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, 0, nil
		}
		return 0, 0, fmt.Errorf("GetFilmRating err: %w", err)
	}

	return rating.Float64, uint64(number.Int64), nil
}

func (repo *RepoPostgre) FindFilm(title string, dateFrom string, dateTo string, ratingFrom float32, ratingTo float32,
	mpaa string, genres []uint32, actors []string, first uint64, limit uint64,
) ([]models.FilmItem, error) {

	films := []models.FilmItem{}
	var hasWhere bool
	paramNum := 1
	var params []interface{}
	var s strings.Builder
	s.WriteString(
		"SELECT DISTINCT film.title, film.id, film.poster, AVG(users_comment.rating) FROM film " +
			"JOIN films_genre ON film.id = films_genre.id_film " +
			"LEFT JOIN users_comment ON film.id = users_comment.id_film " +
			"JOIN person_in_film ON film.id = person_in_film.id_film " +
			"JOIN crew ON person_in_film.id_person = crew.id ")
	if title != "" {
		s.WriteString("WHERE ")
		hasWhere = true
		s.WriteString("fts @@ to_tsquery($" + strconv.Itoa(paramNum) + ") ")
		paramNum++
		params = append(params, title)
	}
	if dateFrom != "" {
		if !hasWhere {
			s.WriteString("WHERE ")
			hasWhere = true
		} else {
			s.WriteString("AND ")
		}
		s.WriteString("release_date >= $" + strconv.Itoa(paramNum) + " ")
		paramNum++
		params = append(params, dateFrom)
	}
	if dateTo != "" {
		if !hasWhere {
			s.WriteString("WHERE ")
			hasWhere = true
		} else {
			s.WriteString("AND ")
		}
		s.WriteString("release_date <= $" + strconv.Itoa(paramNum) + " ")
		paramNum++
		params = append(params, dateTo)
	}
	if mpaa != "" {
		if !hasWhere {
			s.WriteString("WHERE ")
			hasWhere = true
		} else {
			s.WriteString("AND ")
		}
		s.WriteString("mpaa = $" + strconv.Itoa(paramNum) + " ")
		paramNum++
		params = append(params, mpaa)
	}
	if len(genres) > 0 {
		if !hasWhere {
			s.WriteString("WHERE ")
			hasWhere = true
		} else {
			s.WriteString("AND ")
		}
		s.WriteString("(CASE WHEN array_length($" + strconv.Itoa(paramNum) + "::int[], 1)> 0 " +
			"THEN films_genre.id_genre = ANY ($" + strconv.Itoa(paramNum) + "::int[]) ELSE TRUE END) ")
		paramNum++
		params = append(params, pq.Array(genres))
	}
	if actors[0] != "" {
		if !hasWhere {
			s.WriteString("WHERE ")
		} else {
			s.WriteString("AND ")
		}
		s.WriteString("(CASE WHEN array_length($" + strconv.Itoa(paramNum) + "::varchar[], 1)> 0 " +
			"THEN crew.name = ANY ($" + strconv.Itoa(paramNum) + "::varchar[]) ELSE TRUE END) ")
		paramNum++
		params = append(params, pq.Array(actors))
	}
	s.WriteString(
		"GROUP BY film.title, film.id " +
			"HAVING (AVG(users_comment.rating) >= $" + strconv.Itoa(paramNum) + " AND AVG(users_comment.rating) <= $" + strconv.Itoa(paramNum+1) + ") " +
			"OR AVG(users_comment.rating) IS NULL " +
			"ORDER BY film.title " +
			"LIMIT $" + strconv.Itoa(paramNum+2) + " OFFSET $" + strconv.Itoa(paramNum+3))

	params = append(params, ratingFrom, ratingTo, limit, first)
	rows, err := repo.db.Query(s.String(), params...)

	if err != nil {
		return nil, fmt.Errorf("find film err: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		post := models.FilmItem{}
		ratingPost := sql.NullFloat64{}
		err := rows.Scan(&post.Title, &post.Id, &post.Poster, &ratingPost)
		if err != nil {
			return nil, fmt.Errorf("find film scan err: %w", err)
		}
		if !ratingPost.Valid {
			ratingPost.Float64 = 0
		}
		post.Rating = ratingPost.Float64
		films = append(films, post)
	}

	return films, nil
}

func (repo *RepoPostgre) GetFavoriteFilms(userId uint64, start uint64, end uint64) ([]models.FilmItem, error) {
	films := []models.FilmItem{}

	rows, err := repo.db.Query(
		"SELECT film.title, film.id, film.poster FROM film "+
			"JOIN users_favorite_film ON film.id = users_favorite_film.id_film "+
			"WHERE id_user = $1 "+
			"OFFSET $2 LIMIT $3", userId, start, end)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("get favorite films err: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		post := models.FilmItem{}
		err := rows.Scan(&post.Title, &post.Id, &post.Poster)
		if err != nil {
			return nil, fmt.Errorf("get favorite films scan err: %w", err)
		}
		films = append(films, post)
	}

	return films, nil
}

func (repo *RepoPostgre) AddFavoriteFilm(userId uint64, filmId uint64) error {
	_, err := repo.db.Exec(
		"INSERT INTO users_favorite_film(id_user, id_film) VALUES ($1, $2)", userId, filmId)
	if err != nil {
		return fmt.Errorf("add favorite film err: %w", err)
	}

	return nil
}

func (repo *RepoPostgre) RemoveFavoriteFilm(userId uint64, filmId uint64) error {
	_, err := repo.db.Exec(
		"DELETE FROM users_favorite_film "+
			"WHERE id_user = $1 AND id_film = $2", userId, filmId)
	if err != nil {
		return fmt.Errorf("remove favorite film err: %w", err)
	}

	return nil
}

func (repo *RepoPostgre) CheckFilm(userId uint64, filmId uint64) (bool, error) {
	film := models.FilmItem{}
	err := repo.db.QueryRow("SELECT id_film FROM users_favorite_film WHERE id_film = $1 AND id_user = $2", filmId, userId).Scan(&film.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, fmt.Errorf("GetFilm err: %w", err)
	}

	return true, nil
}

func (repo *RepoPostgre) AddRating(filmId uint64, userId uint64, rating uint16) error {
	_, err := repo.db.Exec(
		"INSERT INTO users_comment(id_film, rating, id_user) "+
			"VALUES($1, $2, $3)", filmId, rating, userId)
	if err != nil {
		return fmt.Errorf("AddComment: %w", err)
	}

	return nil
}

func (repo *RepoPostgre) HasUsersRating(userId uint64, filmId uint64) (bool, error) {
	var id uint64
	err := repo.db.QueryRow(
		"SELECT id_user FROM users_comment "+
			"WHERE id_user = $1 AND id_film = $2", userId, filmId).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, fmt.Errorf("has users rating: %w", err)
	}

	return true, nil
}

func (repo *RepoPostgre) AddFilm(film models.FilmItem) error {
	_, err := repo.db.Exec("INSERT INTO film(title, info, poster, release_date, country, mpaa) "+
		"VALUES($1, $2, $3, $4, $5, $6)",
		film.Title, film.Info, film.Poster, film.ReleaseDate, film.Country, film.Mpaa)
	if err != nil {
		return fmt.Errorf("add film error: %w", err)
	}

	return nil
}

func (repo *RepoPostgre) GetFilmId(title string) (uint64, error) {
	var id uint64
	err := repo.db.QueryRow("SELECT id FROM film WHERE title = $1", title).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("get film id err: %w", err)
	}

	return id, nil
}

func (repo *RepoPostgre) DeleteRating(idUser uint64, idFilm uint64) error {
	_, err := repo.db.Exec("DELETE FROM users_comment WHERE id_user = $1 AND id_film = $2", idUser, idFilm)
	if err != nil {
		return fmt.Errorf("delete rating err: %w", err)
	}
	return nil
}

func (repo *RepoPostgre) Trends() ([]models.FilmItem, error) {
	trends := []models.FilmItem{}

	rows, err := repo.db.Query("SELECT film.id, film.title, film.poster FROM film " +
		"JOIN users_comment ON film.id = users_comment.id_film " +
		"WHERE users_comment.date > (CURRENT_TIMESTAMP - interval'48 hours') " +
		"GROUP BY film.title, film.id, film.poster " +
		"ORDER BY COUNT(users_comment.id_film) DESC " +
		"LIMIT 5")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("trends err: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		post := models.FilmItem{}
		err := rows.Scan(&post.Id, &post.Title, &post.Poster)
		if err != nil {
			return nil, fmt.Errorf("trends scan err: %w", err)
		}
		trends = append(trends, post)
	}

	return trends, nil
}

func (repo *RepoPostgre) GetLasts(ids []uint64) ([]models.FilmItem, error) {
	films := []models.FilmItem{}

	rows, err := repo.db.Query("SELECT id, title, poster FROM film "+
		"WHERE (CASE WHEN array_length($1::int[], 1)> 0 "+
		"THEN id = ANY ($1::int[]) ELSE FALSE END) "+
		"ORDER BY array_position($1::int[], id) "+
		"LIMIT 10", pq.Array(ids))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("get lasts err: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		post := models.FilmItem{}
		err := rows.Scan(&post.Id, &post.Title, &post.Poster)
		if err != nil {
			return nil, fmt.Errorf("get lasts scan err: %w", err)
		}
		films = append(films, post)
	}

	return films, nil
}
