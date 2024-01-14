package crew

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

//go:generate mockgen -source=repo_crew.go -destination=../../mocks/crew_repo_mock.go -package=mocks

type ICrewRepo interface {
	GetFilmDirectors(filmId uint64) ([]models.CrewItem, error)
	GetFilmScenarists(filmId uint64) ([]models.CrewItem, error)
	GetFilmCharacters(filmId uint64) ([]models.Character, error)
	GetActor(actorId uint64) (*models.CrewItem, error)
	FindActor(name string, birthDate string, films []string, career []string, country string, first, limit uint64) ([]models.Character, error)
	GetFavoriteActors(userId uint64, start uint64, end uint64) ([]models.Character, error)
	CheckActor(userId uint64, actorId uint64) (bool, error)
	AddFavoriteActor(userId uint64, actorId uint64) error
	RemoveFavoriteActor(userId uint64, actorId uint64) error
	AddFilm(actors []uint64, filmId uint64) error
}

type RepoPostgre struct {
	db *sql.DB
}

func GetCrewRepo(config *configs.DbDsnCfg, lg *slog.Logger) (*RepoPostgre, error) {
	dsn := fmt.Sprintf("user=%s dbname=%s password= %s host=%s port=%d sslmode=%s",
		config.User, config.DbName, config.Password, config.Host, config.Port, config.Sslmode)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		lg.Error("sql open error", "err", err.Error())
		return nil, fmt.Errorf("get crew repo: %w", err)
	}
	err = db.Ping()
	if err != nil {
		lg.Error("sql ping error", "err", err.Error())
		return nil, fmt.Errorf("get crew repo: %w", err)
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

func (repo *RepoPostgre) GetFilmDirectors(filmId uint64) ([]models.CrewItem, error) {
	directors := []models.CrewItem{}

	rows, err := repo.db.Query(
		"SELECT crew.id, name, photo  FROM crew "+
			"JOIN person_in_film ON crew.id = person_in_film.id_person "+
			"WHERE id_film = $1 AND id_profession = "+
			"(SELECT id FROM profession WHERE title = 'режиссёр')", filmId)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("GetFilmDirectors err: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		post := models.CrewItem{}
		err := rows.Scan(&post.Id, &post.Name, &post.Photo)
		if err != nil {
			return nil, fmt.Errorf("GetFilmDirectors scan err: %w", err)
		}
		directors = append(directors, post)
	}

	return directors, nil
}

func (repo *RepoPostgre) GetFilmScenarists(filmId uint64) ([]models.CrewItem, error) {
	scenarists := []models.CrewItem{}

	rows, err := repo.db.Query(
		"SELECT crew.id, name, photo  FROM crew "+
			"JOIN person_in_film ON crew.id = person_in_film.id_person "+
			"WHERE id_film = $1 AND id_profession = "+
			"(SELECT id FROM profession WHERE title = 'сценарист')", filmId)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("GetFilmScenarists err: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		post := models.CrewItem{}
		err := rows.Scan(&post.Id, &post.Name, &post.Photo)
		if err != nil {
			return nil, fmt.Errorf("GetFilmScenarists scan err: %w", err)
		}
		scenarists = append(scenarists, post)
	}

	return scenarists, nil
}

func (repo *RepoPostgre) GetFilmCharacters(filmId uint64) ([]models.Character, error) {
	characters := []models.Character{}

	rows, err := repo.db.Query(
		"SELECT crew.id, name, photo, person_in_film.character_name FROM crew "+
			"JOIN person_in_film ON crew.id = person_in_film.id_person "+
			"WHERE id_film = $1 AND id_profession = "+
			"(SELECT id FROM profession WHERE title = 'актёр')", filmId)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("GetFilmCharacters err: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		post := models.Character{}
		err := rows.Scan(&post.IdActor, &post.NameActor, &post.ActorPhoto, &post.NameCharacter)
		if err != nil {
			return nil, fmt.Errorf("GetFilmCharacters scan err: %w", err)
		}
		characters = append(characters, post)
	}

	return characters, nil
}

func (repo *RepoPostgre) GetActor(actorId uint64) (*models.CrewItem, error) {
	actor := &models.CrewItem{}

	err := repo.db.QueryRow(
		"SELECT id, name, birth_date, photo, info FROM crew "+
			"WHERE id = $1", actorId).
		Scan(&actor.Id, &actor.Name, &actor.Birthdate, &actor.Photo, &actor.Info)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return actor, nil
		}
		return nil, fmt.Errorf("GetActor err: %w", err)
	}

	return actor, nil
}

func (repo *RepoPostgre) FindActor(name string, birthDate string, films []string, career []string, country string, first, limit uint64) ([]models.Character, error) {
	actors := []models.Character{}
	var hasWhere bool
	paramNum := 1
	var params []interface{}
	var s strings.Builder
	s.WriteString("SELECT DISTINCT crew.id, crew.name, crew.photo FROM crew " +
		"JOIN person_in_film ON crew.id = person_in_film.id_person " +
		"JOIN film ON person_in_film.id_film = film.id " +
		"JOIN profession ON person_in_film.id_profession = profession.id ")
	if name != "" {
		s.WriteString("WHERE ")
		hasWhere = true
		s.WriteString("name LIKE '%' || $1 || '%'")
		paramNum++
		params = append(params, name)
	}
	if birthDate != "" {
		if !hasWhere {
			s.WriteString("WHERE ")
			hasWhere = true
		} else {
			s.WriteString("AND ")
		}
		s.WriteString("birth_date = $" + strconv.Itoa(paramNum) + " ")
		paramNum++
		params = append(params, birthDate)
	}
	if films[0] != "" {
		if !hasWhere {
			s.WriteString("WHERE ")
			hasWhere = true
		} else {
			s.WriteString("AND ")
		}
		s.WriteString("(CASE WHEN array_length($" + strconv.Itoa(paramNum) + "::varchar[], 1)> 0 " +
			"THEN film.title = ANY($" + strconv.Itoa(paramNum) + "::varchar[]) ELSE TRUE END) ")

		paramNum++
		params = append(params, pq.Array(films))
	}
	if career[0] != "" {
		if !hasWhere {
			s.WriteString("WHERE ")
			hasWhere = true
		} else {
			s.WriteString("AND ")
		}
		s.WriteString("(CASE WHEN array_length($" + strconv.Itoa(paramNum) + "::varchar[], 1)> 0 " +
			"THEN profession.title = ANY($" + strconv.Itoa(paramNum) + "::varchar[]) ELSE TRUE END) ")

		paramNum++
		params = append(params, pq.Array(career))
	}
	if country != "" {
		if !hasWhere {
			s.WriteString("WHERE ")
		} else {
			s.WriteString("AND ")
		}
		s.WriteString("country = $" + strconv.Itoa(paramNum))
		params = append(params, country)
	}

	s.WriteString("LIMIT $" + strconv.Itoa(paramNum) + " OFFSET $" + strconv.Itoa(paramNum+1))
	params = append(params, limit, first)

	rows, err := repo.db.Query(s.String(), params...)
	if err != nil {
		return nil, fmt.Errorf("find actor err: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		post := models.Character{}
		err := rows.Scan(&post.IdActor, &post.NameActor, &post.ActorPhoto)
		if err != nil {
			return nil, fmt.Errorf("find actor scan err: %w", err)
		}
		actors = append(actors, post)
	}

	return actors, nil
}

func (repo *RepoPostgre) GetFavoriteActors(userId uint64, start uint64, end uint64) ([]models.Character, error) {
	actors := []models.Character{}

	rows, err := repo.db.Query(
		"SELECT crew.name, crew.id, crew.photo FROM crew "+
			"JOIN users_favorite_actor ON crew.id = users_favorite_actor.id_actor "+
			"WHERE id_user = $1 "+
			"OFFSET $2 LIMIT $3", userId, start, end)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("get favorite actors err: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		post := models.Character{}
		err := rows.Scan(&post.NameActor, &post.IdActor, &post.ActorPhoto)
		if err != nil {
			return nil, fmt.Errorf("get favorite actors scan err: %w", err)
		}
		actors = append(actors, post)
	}

	return actors, nil
}

func (repo *RepoPostgre) CheckActor(userId uint64, actorId uint64) (bool, error) {
	actor := models.Character{}
	err := repo.db.QueryRow("SELECT id_actor FROM users_favorite_actor WHERE id_actor = $1 AND id_user = $2", actorId, userId).Scan(&actor.IdActor)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, fmt.Errorf("check actor err: %w", err)
	}

	return true, nil
}

func (repo *RepoPostgre) AddFavoriteActor(userId uint64, actorId uint64) error {
	_, err := repo.db.Exec(
		"INSERT INTO users_favorite_actor(id_user, id_actor) VALUES ($1, $2)", userId, actorId)
	if err != nil {
		return fmt.Errorf("add favorite actor err: %w", err)
	}

	return nil
}

func (repo *RepoPostgre) RemoveFavoriteActor(userId uint64, actorId uint64) error {
	_, err := repo.db.Exec(
		"DELETE FROM users_favorite_actor "+
			"WHERE id_user = $1 AND id_actor = $2", userId, actorId)
	if err != nil {
		return fmt.Errorf("remove favorite actor err: %w", err)
	}

	return nil
}

func (repo *RepoPostgre) AddFilm(actors []uint64, filmId uint64) error {
	var s strings.Builder
	var params []interface{}
	params = append(params, filmId)

	s.WriteString("INSERT INTO person_in_film(id_film, id_person, id_profession, character_name) VALUES")
	for i, actor := range actors {
		if i != 0 {
			s.WriteString(",")
		}
		s.WriteString("($1, $" + strconv.Itoa(i+2) + ", 1, '')")
		params = append(params, actor)
	}

	_, err := repo.db.Exec(s.String(), params...)
	if err != nil {
		return fmt.Errorf("add films actors error: %w", err)
	}
	return nil
}
