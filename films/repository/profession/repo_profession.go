package profession

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-park-mail-ru/2023_2_Vkladyshi/configs"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/models"
)

//go:generate mockgen -source=repo_profession.go -destination=../../mocks/profession_repo_mock.go -package=mocks

type IProfessionRepo interface {
	GetActorsProfessions(actorId uint64) ([]models.ProfessionItem, error)
}

type RepoPostgre struct {
	db *sql.DB
}

func GetProfessionRepo(config *configs.DbDsnCfg, lg *slog.Logger) (*RepoPostgre, error) {
	dsn := fmt.Sprintf("user=%s dbname=%s password= %s host=%s port=%d sslmode=%s",
		config.User, config.DbName, config.Password, config.Host, config.Port, config.Sslmode)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		lg.Error("sql open error", "err", err.Error())
		return nil, fmt.Errorf("get prof repo: %w", err)
	}
	err = db.Ping()
	if err != nil {
		lg.Error("sql ping error", "err", err.Error())
		return nil, fmt.Errorf("get prof repo: %w", err)
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
			lg.Error("Repo Profession db ping error", "err", err.Error())
		}

		time.Sleep(time.Duration(timer) * time.Second)
	}
}

func (repo *RepoPostgre) GetActorsProfessions(actorId uint64) ([]models.ProfessionItem, error) {
	professions := []models.ProfessionItem{}

	rows, err := repo.db.Query(
		"SELECT DISTINCT title FROM profession "+
			"JOIN person_in_film ON profession.id = person_in_film.id_profession "+
			"WHERE id_person = $1", actorId)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("GetActorsProfessions err: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		post := models.ProfessionItem{}
		err := rows.Scan(&post.Title)
		if err != nil {
			return nil, fmt.Errorf("GetActorsProfessions scan err: %w", err)
		}
		professions = append(professions, post)
	}

	return professions, nil
}
