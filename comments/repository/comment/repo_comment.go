package comment

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

//go:generate mockgen -source=repo_comment.go -destination=../../mocks/repo_mock.go -package=mocks

type ICommentRepo interface {
	GetFilmComments(filmId uint64, first uint64, limit uint64) ([]models.CommentItem, error)
	AddComment(filmId uint64, userId uint64, rating uint16, text string) error
	HasUsersComment(userId uint64, filmId uint64) (bool, error)
	DeleteComment(idUser uint64, idFilm uint64) error
}

type RepoPostgre struct {
	db *sql.DB
}

func GetCommentRepo(config *configs.CommentCfg, lg *slog.Logger) (*RepoPostgre, error) {
	dsn := fmt.Sprintf("user=%s dbname=%s password= %s host=%s port=%d sslmode=%s",
		config.User, config.DbName, config.Password, config.Host, config.Port, config.Sslmode)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		lg.Error("sql open error", "err", err.Error())
		return nil, fmt.Errorf("get comment repo: %w", err)
	}
	err = db.Ping()
	if err != nil {
		lg.Error("sql ping error", "err", err.Error())
		return nil, fmt.Errorf("get comment repo: %w", err)
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
			lg.Error("repo comment db ping error", "err", err.Error())
		}

		time.Sleep(time.Duration(timer) * time.Second)
	}
}

func (repo *RepoPostgre) GetFilmComments(filmId uint64, first uint64, limit uint64) ([]models.CommentItem, error) {
	comments := []models.CommentItem{}

	rows, err := repo.db.Query(
		"SELECT id_user, rating, comment FROM users_comment "+
			"WHERE id_film = $1 "+
			"OFFSET $2 LIMIT $3", filmId, first, limit)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("GetFilmRating err: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		post := models.CommentItem{}
		err := rows.Scan(&post.IdUser, &post.Rating, &post.Comment)
		if err != nil {
			return nil, fmt.Errorf("GetFilmRating scan err: %w", err)
		}
		comments = append(comments, post)
	}

	return comments, nil
}

func (repo *RepoPostgre) AddComment(filmId uint64, userId uint64, rating uint16, text string) error {
	_, err := repo.db.Exec(
		"INSERT INTO users_comment(id_film, rating, comment, id_user) "+
			"VALUES($1, $2, $3, $4)", filmId, rating, text, userId)
	if err != nil {
		return fmt.Errorf("AddComment: %w", err)
	}

	return nil
}

func (repo *RepoPostgre) HasUsersComment(userId uint64, filmId uint64) (bool, error) {
	var id uint64
	err := repo.db.QueryRow(
		"SELECT id_user FROM users_comment "+
			"WHERE id_user = $1 AND id_film = $2", userId, filmId).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (repo *RepoPostgre) DeleteComment(idUser uint64, idFilm uint64) error {
	_, err := repo.db.Exec("DELETE FROM users_comment WHERE id_user = $1 AND id_film = $2", idUser, idFilm)
	if err != nil {
		return fmt.Errorf("delete comment err: %w", err)
	}
	return nil
}
