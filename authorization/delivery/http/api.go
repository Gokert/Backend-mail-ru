package delivery

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2023_2_Vkladyshi/authorization/usecase"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/requests"
	"github.com/mailru/easyjson"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type IApi interface {
	SendResponse(w http.ResponseWriter, response requests.Response)
	Signin(w http.ResponseWriter, r *http.Request)
	SigninResponse(w http.ResponseWriter, r *http.Request)
	Signup(w http.ResponseWriter, r *http.Request)
	LogoutSession(w http.ResponseWriter, r *http.Request)
	AuthAccept(w http.ResponseWriter, r *http.Request)
}

type API struct {
	core  usecase.ICore
	lg   *slog.Logger
	ct   *requests.Collector
	mx   *http.ServeMux
}

func (a *API) ListenAndServe() error {
	err := http.ListenAndServe(":8081", a.mx)
	if err != nil {
		a.lg.Error("ListenAndServe error", "err", err.Error())
		return fmt.Errorf("listen and serve error: %w", err)
	}

	return nil
}

func GetApi(c *usecase.Core, l *slog.Logger) *API {
	api := &API{
		core: c,
		lg:   l.With("module", "api"),
		ct:   requests.GetCollector(),
		mx:   http.NewServeMux(),
	}

	api.mx.Handle("/metrics", promhttp.Handler())
	api.mx.HandleFunc("/signin", api.Signin)
	api.mx.HandleFunc("/signup", api.Signup)
	api.mx.HandleFunc("/logout", api.LogoutSession)
	api.mx.HandleFunc("/authcheck", api.AuthAccept)
	api.mx.HandleFunc("/api/v1/csrf", api.GetCsrfToken)
	api.mx.HandleFunc("/api/v1/settings", api.Profile)
	api.mx.HandleFunc("/api/v1/user/subscribePush", api.SubcribePush)
	api.mx.HandleFunc("/api/v1/user/isSubscribed", api.IsSubcribed)
	api.mx.HandleFunc("/api/v1/users/list", api.GetUsers)
	api.mx.HandleFunc("/api/v1/users/updateRole", api.ChangeUserRole)

	return api
}

func (a *API) LogoutSession(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}

	start := time.Now()

	session, err := r.Cookie("session_id")
	if err == http.ErrNoCookie {
		response.Status = http.StatusUnauthorized
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	found, _ := a.core.FindActiveSession(r.Context(), session.Value)
	if !found {
		response.Status = http.StatusUnauthorized
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	} else {
		err := a.core.KillSession(r.Context(), session.Value)
		if err != nil {
			a.lg.Error("failed to kill session", "err", err.Error())
		}
		session.Expires = time.Now().AddDate(0, 0, -1)
		http.SetCookie(w, session)
	}
	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) AuthAccept(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}

	start := time.Now()
	var authorized bool

	session, err := r.Cookie("session_id")
	if err == nil && session != nil {
		authorized, _ = a.core.FindActiveSession(r.Context(), session.Value)
	}

	if !authorized {
		response.Status = http.StatusUnauthorized
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}
	login, err := a.core.GetUserName(r.Context(), session.Value)
	if err != nil {
		a.lg.Error("auth accept error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	role, err := a.core.GetUserRole(login)
	if err != nil {
		a.lg.Error("auth accept error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	authCheckResponse := requests.AuthCheckResponse{Login: login, Role: role}
	response.Body = authCheckResponse
	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) Signin(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}

	start := time.Now()
	if r.Method != http.MethodPost {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	csrfToken := r.Header.Get("x-csrf-token")

	_, err := a.core.CheckCsrfToken(r.Context(), csrfToken)
	if err != nil {
		w.Header().Set("X-CSRF-Token", "null")
		response.Status = http.StatusPreconditionFailed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	var request requests.SigninRequest

	body, err := io.ReadAll(r.Body)
	if err != nil {
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	if err = easyjson.Unmarshal(body, &request); err != nil {
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	user, found, err := a.core.FindUserAccount(request.Login, request.Password)
	if err != nil {
		a.lg.Error("Signin error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}
	if !found {
		response.Status = http.StatusUnauthorized
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	} else {
		sid, session, _ := a.core.CreateSession(r.Context(), user.Login)
		cookie := &http.Cookie{
			Name:     "session_id",
			Value:    sid,
			Path:     "/",
			Expires:  session.ExpiresAt,
			HttpOnly: true,
		}
		http.SetCookie(w, cookie)
	}
	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) Signup(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}

	start := time.Now()
	if r.Method != http.MethodPost {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	csrfToken := r.Header.Get("x-csrf-token")

	_, err := a.core.CheckCsrfToken(r.Context(), csrfToken)
	if err != nil {
		w.Header().Set("X-CSRF-Token", "null")
		response.Status = http.StatusPreconditionFailed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	var request requests.SignupRequest

	body, err := io.ReadAll(r.Body)
	if err != nil {
		a.lg.Error("Signup error", "err", err.Error())
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	err = easyjson.Unmarshal(body, &request)
	if err != nil {
		a.lg.Error("Signup error", "err", err.Error())
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	found, err := a.core.FindUserByLogin(request.Login)
	if err != nil {
		a.lg.Error("Signup error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	if found {
		response.Status = http.StatusConflict
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	err = a.core.CreateUserAccount(request.Login, request.Password, request.Name, request.BirthDate, request.Email)
	if err == usecase.InvalideEmail {
		a.lg.Error("create user error", "err", err.Error())
		response.Status = http.StatusBadRequest
	}
	if err != nil {
		a.lg.Error("failed to create user account", "err", err.Error())
		response.Status = http.StatusBadRequest
	}
	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) GetCsrfToken(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}

	start := time.Now()

	csrfToken := r.Header.Get("x-csrf-token")

	found, err := a.core.CheckCsrfToken(r.Context(), csrfToken)
	if err != nil {
		w.Header().Set("X-CSRF-Token", "null")
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}
	if csrfToken != "" && found {
		w.Header().Set("X-CSRF-Token", csrfToken)
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	token, err := a.core.CreateCsrfToken(r.Context())
	if err != nil {
		w.Header().Set("X-CSRF-Token", "null")
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	w.Header().Set("X-CSRF-Token", token)
	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) GetUsers(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	start := time.Now()

	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	login := r.URL.Query().Get("login")
	role := r.URL.Query().Get("role")

	page, err := strconv.ParseUint(r.URL.Query().Get("page"), 10, 64)
	if err != nil {
		page = 1
	}

	pageSize, err := strconv.ParseUint(r.URL.Query().Get("per_page"), 10, 64)
	if err != nil {
		pageSize = 10
	}

	users, err := a.core.FindUsers(login, role, (page-1)*pageSize, pageSize)
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			response.Status = http.StatusNotFound
			a.ct.SendResponse(w, r, response, a.lg, start)
			return
		}
		a.lg.Error("get users error", "err:", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	response.Body = users
	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) ChangeUserRole(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}

	start := time.Now()
	if r.Method != http.MethodPost {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	csrfToken := r.Header.Get("x-csrf-token")

	_, err := a.core.CheckCsrfToken(r.Context(), csrfToken)
	if err != nil {
		w.Header().Set("X-CSRF-Token", "null")
		response.Status = http.StatusPreconditionFailed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		a.lg.Error("ChangeRole error", "err", err.Error())
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	var request requests.ChangeRoleRequest

	err = easyjson.Unmarshal(body, &request)
	if err != nil {
		a.lg.Error("Signup error", "err", err.Error())
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	session, err := r.Cookie("session_id")
	if errors.Is(err, http.ErrNoCookie) {
		response.Status = http.StatusUnauthorized
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	userName, err := a.core.GetUserName(r.Context(), session.Value)
	if err != nil {
		a.lg.Error("User login not found", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	userRole, err := a.core.GetUserRole(userName)
	if err != nil {
		a.lg.Error("User role not found", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	err = a.core.ChangeUsersRole(request.Login, request.Role, userRole)
	if err != nil {
		if errors.Is(err, usecase.ErrNotAllowed) {
			response.Status = http.StatusForbidden
			a.ct.SendResponse(w, r, response, a.lg, start)
			return
		}
		a.lg.Error("Change user role error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) Profile(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}

	start := time.Now()
	if r.Method == http.MethodGet {
		session, err := r.Cookie("session_id")
		if err == http.ErrNoCookie {
			response.Status = http.StatusUnauthorized
			a.ct.SendResponse(w, r, response, a.lg, start)
			return
		}

		login, err := a.core.GetUserName(r.Context(), session.Value)
		if err != nil {
			a.lg.Error("Get Profile error", "err", err.Error())
		}

		profile, err := a.core.GetUserProfile(login)
		if err != nil {
			response.Status = http.StatusInternalServerError
			a.ct.SendResponse(w, r, response, a.lg, start)
			return
		}

		profileResponse := requests.ProfileResponse{
			Email:     profile.Email,
			Name:      profile.Name,
			Login:     profile.Login,
			Photo:     profile.Photo,
			BirthDate: profile.Birthdate,
		}

		response.Body = profileResponse
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	if r.Method != http.MethodPost {
		response.Status = http.StatusUnauthorized
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}
	session, err := r.Cookie("session_id")
	if err == http.ErrNoCookie {
		response.Status = http.StatusUnauthorized
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	prevLogin, err := a.core.GetUserName(r.Context(), session.Value)
	if err != nil {
		a.lg.Error("Get Profile error", "err", err.Error())
	}

	err1 := r.ParseMultipartForm(10 << 20)
	if err1 != nil {
		a.lg.Error("Post profile error", "err", err.Error())
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	email := r.FormValue("email")
	login := r.FormValue("login")
	birthDate := r.FormValue("birthday")
	password := r.FormValue("password")
	photo, handler, err := r.FormFile("photo")
	if err != nil && !errors.Is(err, http.ErrMissingFile) {
		a.lg.Error("Post profile error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	isRepeatPassword, err := a.core.CheckPassword(login, password)

	if isRepeatPassword {
		response.Status = http.StatusConflict
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	var filename string
	if handler == nil {
		filename = ""

		err = a.core.EditProfile(prevLogin, login, password, email, birthDate, filename)
		if err != nil {
			a.lg.Error("Post profile error", "err", err.Error())
			response.Status = http.StatusInternalServerError
			a.ct.SendResponse(w, r, response, a.lg, start)
			return
		}
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	filename = "/avatars/" + handler.Filename

	if err != nil && handler != nil && photo != nil {
		a.lg.Error("Post profile error", "err", err.Error())
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	filePhoto, err := os.OpenFile("/home/ubuntu/frontend-project"+filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		a.lg.Error("Post profile error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}
	defer filePhoto.Close()

	_, err = io.Copy(filePhoto, photo)
	if err != nil {
		a.lg.Error("Post profile error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	err = a.core.EditProfile(prevLogin, login, password, email, birthDate, filename)
	if err != nil {
		a.lg.Error("Post profile error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}
	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) SubcribePush(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	start := time.Now()
	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	session, err := r.Cookie("session_id")
	if errors.Is(err, http.ErrNoCookie) {
		response.Status = http.StatusUnauthorized
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	userName, err := a.core.GetUserName(r.Context(), session.Value)
	if err != nil {
		a.lg.Error("subcribe push error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	isSubcribed, err := a.core.Subscribe(userName)
	if err != nil {
		a.lg.Error("subcribe push error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	subResponse := requests.SubcribeResponse{IsSubcribed: isSubcribed}
	response.Body = subResponse
	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) IsSubcribed(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	start := time.Now()
	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	session, err := r.Cookie("session_id")
	if errors.Is(err, http.ErrNoCookie) {
		response.Status = http.StatusUnauthorized
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	userName, err := a.core.GetUserName(r.Context(), session.Value)
	if err != nil {
		a.lg.Error("is subcribed error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	isSubcribed, err := a.core.IsSubscribed(userName)
	if err != nil {
		a.lg.Error("is subcribed error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	subResponse := requests.SubcribeResponse{IsSubcribed: isSubcribed}
	response.Body = subResponse
	a.ct.SendResponse(w, r, response, a.lg, start)
}
