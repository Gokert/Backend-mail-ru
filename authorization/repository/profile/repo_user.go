package profile

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
	_ "github.com/jackc/pgx/stdlib"
	"github.com/lib/pq"
)

type IUserRepo interface {
	GetUser(login string, password string) (*models.UserItem, bool, error)
	GetUserProfileId(login string) (int64, error)
	FindUser(login string) (bool, error)
	CreateUser(login string, password string, name string, birthDate string, email string) error
	GetUserProfile(login string) (*models.UserItem, error)
	EditProfile(prevLogin string, login string, password string, email string, birthDate string, photo string) error
	GetNamesAndPaths(ids []int32) ([]string, []string, error)
	CheckUserPassword(login string, password string) (bool, error)
	GetUserRole(login string) (string, error)
	IsSubscribed(login string) (bool, error)
	ChangeSubsribe(login string, isSubscribed bool) error
	FindUsers(login string, role string, first, limit uint64) ([]models.UserItem, error)
	ChangeUsersRole(login string, role string) error
}

type RepoPostgre struct {
	db *sql.DB
}

func GetUserRepo(config *configs.DbDsnCfg, lg *slog.Logger) (*RepoPostgre, error) {
	dsn := fmt.Sprintf("user=%s dbname=%s password= %s host=%s port=%d sslmode=%s",
		config.User, config.DbName, config.Password, config.Host, config.Port, config.Sslmode)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		lg.Error("sql open error", "err", err.Error())
		return nil, fmt.Errorf("get user repo err: %w", err)
	}
	err = db.Ping()
	if err != nil {
		lg.Error("sql ping error", "err", err.Error())
		return nil, fmt.Errorf("get user repo err: %w", err)
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
			lg.Error("Repo Profile db ping error", "err", err.Error())
		}

		time.Sleep(time.Duration(timer) * time.Second)
	}
}

func (repo *RepoPostgre) CheckUserPassword(login string, password string) (bool, error) {
	post := &models.UserItem{}

	err := repo.db.QueryRow(
		"SELECT login FROM profile "+
			"WHERE login = $1 AND password = $2", login, password).Scan(&post.Login)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("CheckUserPassword err: %w", err)
	}

	return true, nil
}

func (repo *RepoPostgre) GetUser(login string, password string) (*models.UserItem, bool, error) {
	post := &models.UserItem{}

	err := repo.db.QueryRow(
		"SELECT login, photo FROM profile "+
			"WHERE login = $1 AND password = $2", login, password).Scan(&post.Login, &post.Photo)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("GetUser err: %w", err)
	}

	return post, true, nil
}

func (repo *RepoPostgre) FindUser(login string) (bool, error) {
	post := &models.UserItem{}

	err := repo.db.QueryRow(
		"SELECT login FROM profile "+
			"WHERE login = $1", login).Scan(&post.Login)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("FindUser err: %w", err)
	}

	return true, nil
}

func (repo *RepoPostgre) GetUserProfileId(login string) (int64, error) {
	var userID int64

	err := repo.db.QueryRow(
		"SELECT id FROM profile WHERE login = $1", login).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("User not found for login: %s", login)
		}
		return 0, fmt.Errorf("GetUserProfileID error: %w", err)
	}

	return userID, nil
}

func (repo *RepoPostgre) CreateUser(login string, password string, name string, birthDate string, email string) error {
	_, err := repo.db.Exec(
		"INSERT INTO profile(name, birth_date, photo, login, password, email, registration_date) "+
			"VALUES($1, $2, '/avatars/default.jpg', $3, $4, $5, CURRENT_TIMESTAMP)",
		name, birthDate, login, password, email)
	if err != nil {
		return fmt.Errorf("CreateUser err: %w", err)
	}

	return nil
}

func (repo *RepoPostgre) GetNamesAndPaths(ids []int32) ([]string, []string, error) {
	var s strings.Builder
	s.WriteString("SELECT login, photo FROM profile WHERE id = ANY ($1::INTEGER[]) " +
		"ORDER BY array_position($1::INTEGER[], id)")

	rows, err := repo.db.Query(s.String(), pq.Array(ids))
	if err != nil {
		return nil, nil, fmt.Errorf("GetMatchingNamesAndPaths query error: %w", err)
	}
	defer rows.Close()

	var names []string
	var paths []string

	for rows.Next() {
		var name string
		var path string
		if err := rows.Scan(&name, &path); err != nil {
			return nil, nil, fmt.Errorf("GetMatchingNamesAndPaths scan error: %w", err)
		}
		names = append(names, name)
		paths = append(paths, path)
	}

	return names, paths, nil
}

func (repo *RepoPostgre) GetUserProfile(login string) (*models.UserItem, error) {
	post := &models.UserItem{}

	err := repo.db.QueryRow(
		"SELECT name, birth_date, login, email, photo FROM profile "+
			"WHERE login = $1", login).Scan(&post.Name, &post.Birthdate, &post.Login, &post.Email, &post.Photo)
	if err != nil {
		return nil, fmt.Errorf("GetUserProfile err: %w", err)
	}

	return post, nil
}

func (repo *RepoPostgre) EditProfile(prevLogin string, login string, password string, email string, birthDate string, photo string) error {
	var s strings.Builder
	paramNum := 1
	var params []interface{}

	s.WriteString("UPDATE profile SET ")

	if login != "" {
		s.WriteString("login = $" + strconv.Itoa(paramNum))
		paramNum++
		params = append(params, login)
	}
	if photo != "" {
		if paramNum != 1 {
			s.WriteString(", ")
		}
		s.WriteString("photo = $" + strconv.Itoa(paramNum))
		paramNum++
		params = append(params, photo)
	}
	if email != "" {
		if paramNum != 1 {
			s.WriteString(", ")
		}
		s.WriteString("email = $" + strconv.Itoa(paramNum))
		paramNum++
		params = append(params, email)
	}
	if password != "" {
		if paramNum != 1 {
			s.WriteString(", ")
		}
		s.WriteString("password = $" + strconv.Itoa(paramNum))
		paramNum++
		params = append(params, password)
	}
	if birthDate != "" {
		if paramNum != 1 {
			s.WriteString(", ")
		}
		s.WriteString("birth_date = $" + strconv.Itoa(paramNum))
		paramNum++
		params = append(params, birthDate)
	}
	s.WriteString(" WHERE login = $" + strconv.Itoa(paramNum))
	params = append(params, prevLogin)
	_, err := repo.db.Exec(s.String(), params...)
	if err != nil {
		return fmt.Errorf("failed to edit profile in db: %w", err)
	}

	return nil
}

func (repo *RepoPostgre) GetUserRole(login string) (string, error) {
	var role string

	err := repo.db.QueryRow("SELECT role FROM profile WHERE login = $1", login).Scan(&role)
	if err != nil {
		return "", fmt.Errorf("get user role err: %w", err)
	}

	return role, nil
}

func (repo *RepoPostgre) IsSubscribed(login string) (bool, error) {
	var isSubcribed bool

	err := repo.db.QueryRow("SELECT profile.is_subsсribed FROM profile WHERE login = $1", login).Scan(&isSubcribed)
	if err != nil {
		return false, fmt.Errorf("is subscribed err: %w", err)
	}

	return isSubcribed, nil
}

func (repo *RepoPostgre) ChangeSubsribe(login string, isSubscribed bool) error {
	_, err := repo.db.Exec("UPDATE profile SET is_subsсribed = $1 WHERE login = $2", isSubscribed, login)
	if err != nil {
		return fmt.Errorf("change subscribe error: %w", err)
	}

	return nil
}

func (repo *RepoPostgre) FindUsers(login string, role string, first, limit uint64) ([]models.UserItem, error) {
	users := []models.UserItem{}

	var hasWhere bool
	paramNum := 1
	var params []interface{}
	var s strings.Builder
	s.WriteString("SELECT id, login, photo, role FROM profile ")
	if login != "" {
		s.WriteString("WHERE login = $1 ")
		hasWhere = true
		paramNum++
		params = append(params, login)
	}
	if role != "" {
		if !hasWhere {
			s.WriteString("WHERE ")
		} else {
			s.WriteString("AND ")
		}
		s.WriteString("role = $" + strconv.Itoa(paramNum) + " ")
		paramNum++
		params = append(params, role)
	}
	s.WriteString("LIMIT $" + strconv.Itoa(paramNum) + " OFFSET $" + strconv.Itoa(paramNum+1))
	params = append(params, limit, first)

	rows, err := repo.db.Query(s.String(), params...)
	if err != nil {
		return nil, fmt.Errorf("find users err: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		post := models.UserItem{}
		err := rows.Scan(&post.Id, &post.Login, &post.Photo, &post.Role)
		if err != nil {
			return nil, fmt.Errorf("find users scan err: %w", err)
		}
		users = append(users, post)
	}

	return users, nil
}

func (repo *RepoPostgre) ChangeUsersRole(login string, role string) error {
	_, err := repo.db.Exec("UPDATE profile SET role = $1 WHERE login = $2", role, login)
	if err != nil {
		return fmt.Errorf("change user role error: %w", err)
	}

	return nil
}
