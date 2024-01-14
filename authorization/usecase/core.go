package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"regexp"
	"sync"
	"time"

	"github.com/go-park-mail-ru/2023_2_Vkladyshi/authorization/repository/csrf"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/authorization/repository/profile"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/authorization/repository/session"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/configs"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/models"
)

type ICore interface {
	CreateSession(ctx context.Context, login string) (string, session.Session, error)
	KillSession(ctx context.Context, sid string) error
	FindActiveSession(ctx context.Context, sid string) (bool, error)
	CreateUserAccount(login string, password string, name string, birthDate string, email string) error
	FindUserAccount(login string, password string) (*models.UserItem, bool, error)
	FindUserByLogin(login string) (bool, error)
	GetUserName(ctx context.Context, sid string) (string, error)
	GetUserProfile(login string) (*models.UserItem, error)
	EditProfile(prevLogin string, login string, password string, email string, birthDate string, photo string) error
	CheckCsrfToken(ctx context.Context, token string) (bool, error)
	CreateCsrfToken(ctx context.Context) (string, error)
	CheckPassword(login string, password string) (bool, error)
	GetUserRole(login string) (string, error)
	Subscribe(userName string) (bool, error)
	IsSubscribed(userName string) (bool, error)
	FindUsers(login string, role string, first, limit uint64) ([]models.UserItem, error)
	ChangeUsersRole(login string, role string, currentUserRole string) error
}

type Core struct {
	sessions   session.SessionRepo
	mutex      sync.RWMutex
	lg         *slog.Logger
	users      profile.IUserRepo
	csrfTokens csrf.CsrfRepo
}

var (
	ErrNotFound    = errors.New("not found")
	ErrNotAllowed  = errors.New("not allowed")
	LostConnection = errors.New("Redis connection lost")
	InvalideEmail  = errors.New("invalide email")
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func GetCore(cfg_sql *configs.DbDsnCfg, cfg_csrf configs.DbRedisCfg, cfg_sessions configs.DbRedisCfg, lg *slog.Logger) (*Core, error) {
	session, err := session.GetSessionRepo(cfg_sessions, lg)

	if err != nil {
		lg.Error("Session repository is not responding")
		return nil, err
	}

	users, err := profile.GetUserRepo(cfg_sql, lg)
	if err != nil {
		lg.Error("cant create repo")
		return nil, err
	}

	csrf, err := csrf.GetCsrfRepo(cfg_csrf, lg)
	if err != nil {
		lg.Error("Csrf repository is not responding")
		return nil, err
	}

	core := Core{
		sessions:   *session,
		lg:         lg.With("module", "core"),
		users:      users,
		csrfTokens: *csrf,
	}
	return &core, nil
}

func (core *Core) CheckPassword(login string, password string) (bool, error) {
	found, err := core.users.CheckUserPassword(login, password)
	if err != nil {
		core.lg.Error("find user error", "err", err.Error())
		return false, fmt.Errorf("FindUserAccount err: %w", err)
	}
	return found, nil
}

func (core *Core) EditProfile(prevLogin string, login string, password string, email string, birthDate string, photo string) error {
	err := core.users.EditProfile(prevLogin, login, password, email, birthDate, photo)
	if err != nil {
		core.lg.Error("Edit profile error", "err", err.Error())
		return fmt.Errorf("Edit profile error: %w", err)
	}

	return nil
}

func (core *Core) GetUserName(ctx context.Context, sid string) (string, error) {
	core.mutex.RLock()
	login, err := core.sessions.GetUserLogin(ctx, sid, core.lg)
	core.mutex.RUnlock()

	if err != nil {
		return "", err
	}

	return login, nil
}

func (core *Core) CreateSession(ctx context.Context, login string) (string, session.Session, error) {
	sid := RandStringRunes(32)

	newSession := session.Session{
		Login:     login,
		SID:       sid,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	core.mutex.Lock()
	sessionAdded, err := core.sessions.AddSession(ctx, newSession, core.lg)
	core.mutex.Unlock()

	if !sessionAdded && err != nil {
		return "", session.Session{}, err
	}

	if !sessionAdded {
		return "", session.Session{}, nil
	}

	return sid, newSession, nil
}

func (core *Core) FindActiveSession(ctx context.Context, sid string) (bool, error) {
	core.mutex.RLock()
	found, err := core.sessions.CheckActiveSession(ctx, sid, core.lg)
	core.mutex.RUnlock()

	if err != nil {
		return false, err
	}

	return found, nil
}

func (core *Core) KillSession(ctx context.Context, sid string) error {
	core.mutex.Lock()
	_, err := core.sessions.DeleteSession(ctx, sid, core.lg)
	core.mutex.Unlock()

	if err != nil {
		return err
	}

	return nil
}

func (core *Core) CreateUserAccount(login string, password string, name string, birthDate string, email string) error {
	if matched, _ := regexp.MatchString(`@`, email); !matched {
		return InvalideEmail
	}
	err := core.users.CreateUser(login, password, name, birthDate, email)
	if err != nil {
		core.lg.Error("create user error", "err", err.Error())
		return fmt.Errorf("CreateUserAccount err: %w", err)
	}

	return nil
}

func (core *Core) FindUserAccount(login string, password string) (*models.UserItem, bool, error) {
	user, found, err := core.users.GetUser(login, password)
	if err != nil {
		core.lg.Error("find user error", "err", err.Error())
		return nil, false, fmt.Errorf("FindUserAccount err: %w", err)
	}
	return user, found, nil
}

func (core *Core) FindUserByLogin(login string) (bool, error) {
	found, err := core.users.FindUser(login)
	if err != nil {
		core.lg.Error("find user error", "err", err.Error())
		return false, fmt.Errorf("FindUserByLogin err: %w", err)
	}

	return found, nil
}

func RandStringRunes(seed int) string {
	symbols := make([]rune, seed)
	for i := range symbols {
		symbols[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(symbols)
}

func (core *Core) GetUserProfile(login string) (*models.UserItem, error) {
	profile, err := core.users.GetUserProfile(login)
	if err != nil {
		core.lg.Error("GetUserProfile error", "err", err.Error())
		return nil, fmt.Errorf("GetUserProfile err: %w", err)
	}

	return profile, nil
}

func (core *Core) CheckCsrfToken(ctx context.Context, token string) (bool, error) {
	core.mutex.RLock()
	found, err := core.csrfTokens.CheckActiveCsrf(ctx, token, core.lg)
	core.mutex.RUnlock()

	if err != nil {
		return false, err
	}

	return found, err
}

func (core *Core) CreateCsrfToken(ctx context.Context) (string, error) {
	sid := RandStringRunes(32)

	core.mutex.Lock()
	csrfAdded, err := core.csrfTokens.AddCsrf(
		ctx,
		models.Csrf{
			SID:       sid,
			ExpiresAt: time.Now().Add(3 * time.Hour),
		},
		core.lg,
	)
	core.mutex.Unlock()

	if !csrfAdded && err != nil {
		return "", err
	}

	if !csrfAdded {
		return "", nil
	}

	return sid, nil
}

func (core *Core) GetUserRole(login string) (string, error) {
	role, err := core.users.GetUserRole(login)
	if err != nil {
		core.lg.Error("get user role error", "err", err.Error())
		return "", fmt.Errorf("get user role err: %w", err)
	}

	return role, nil
}

func (core *Core) Subscribe(userName string) (bool, error) {
	isSubcribed, err := core.users.IsSubscribed(userName)
	if err != nil {
		core.lg.Error("is subsribed error", "err", err.Error())
		return false, fmt.Errorf("")
	}

	err = core.users.ChangeSubsribe(userName, !isSubcribed)
	if err != nil {
		core.lg.Error("change subscribe error", "err", err.Error())
		return false, fmt.Errorf("")
	}

	return !isSubcribed, nil
}

func (core *Core) IsSubscribed(userName string) (bool, error) {
	isSubcribed, err := core.users.IsSubscribed(userName)
	if err != nil {
		core.lg.Error("is subcsribed error", "err", err.Error())
		return false, fmt.Errorf("")
	}

	return isSubcribed, nil
}

func (core *Core) FindUsers(login string, role string, first, limit uint64) ([]models.UserItem, error) {
	users, err := core.users.FindUsers(login, role, first, limit)
	if err != nil {
		core.lg.Error("find user error", "err:", err.Error())
		return nil, fmt.Errorf("find user error: %w", err)
	}
	if len(users) == 0 {
		return nil, ErrNotFound
	}

	return users, nil
}

func (core *Core) ChangeUsersRole(login string, role string, currentUserRole string) error {
	if currentUserRole != "super" {
		return ErrNotAllowed
	}

	err := core.users.ChangeUsersRole(login, role)
	if err != nil {
		core.lg.Error("change user role error", "err:", err.Error())
		return fmt.Errorf("change user role error: %w", err)
	}
	return nil
}
