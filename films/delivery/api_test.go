package delivery

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/go-park-mail-ru/2023_2_Vkladyshi/films/mocks"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/films/usecase"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/middleware"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/models"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/requests"
	"github.com/golang/mock/gomock"
	"github.com/mailru/easyjson"
)

func getExpectedResult(res *requests.Response) *requests.Response {
	jsonResponse, _ := easyjson.Marshal(res)
	var response requests.Response
	err := easyjson.Unmarshal(jsonResponse, &response)
	if err != nil {
		fmt.Println("unexpected error")
	}
	return &response
}

func getResponse(w *httptest.ResponseRecorder) (*requests.Response, error) {
	var response requests.Response

	body, _ := io.ReadAll(w.Body)
	err := easyjson.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("cant unmarshal jsone")
	}

	return &response, nil
}

func createBody(req requests.FindFilmRequest) io.Reader {
	jsonReq, _ := easyjson.Marshal(req)

	body := bytes.NewBuffer(jsonReq)
	return body
}

func createActorBody(req requests.FindActorRequest) io.Reader {
	jsonReq, _ := easyjson.Marshal(req)

	body := bytes.NewBuffer(jsonReq)
	return body
}

func createRatingBody(req requests.CommentRequest) io.Reader {
	jsonReq, _ := easyjson.Marshal(req)

	body := bytes.NewBuffer(jsonReq)
	return body
}

func createDelRatingBody(req requests.DeleteCommentRequest) io.Reader {
	jsonReq, _ := easyjson.Marshal(req)

	body := bytes.NewBuffer(jsonReq)
	return body
}

var collector *requests.Collector = requests.GetCollector()

func TestFilms(t *testing.T) {
	expectedGenre := "g1"
	filmItem := models.FilmItem{Title: "t1"}
	expectedFilms := []models.FilmItem{filmItem}
	expectedResponse := requests.FilmsResponse{
		Page:           1,
		PageSize:       8,
		CollectionName: expectedGenre,
		Total:          uint64(len(expectedFilms)),
		Films:          expectedFilms,
	}

	testCases := map[string]struct {
		method string
		result *requests.Response
		params map[string]string
	}{
		"Bad method": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusMethodNotAllowed, Body: nil},
		},
		"Core error": {
			method: http.MethodGet,
			result: &requests.Response{Status: http.StatusInternalServerError, Body: nil},
		},
		"Ok": {
			method: http.MethodGet,
			result: getExpectedResult(&requests.Response{Status: http.StatusOK, Body: expectedResponse}),
			params: map[string]string{"collection_id": "1"},
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCore := mocks.NewMockICore(mockCtrl)

	mockCore.EXPECT().GetFilmsAndGenreTitle(uint64(0), uint64(0), uint64(8)).Return(nil, "", fmt.Errorf("core_err")).Times(1)
	mockCore.EXPECT().GetFilmsAndGenreTitle(uint64(1), uint64(0), uint64(8)).Return(expectedFilms, expectedGenre, nil).Times(1)
	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	api := API{core: mockCore, lg: logger, ct: collector}

	for _, curr := range testCases {
		r := httptest.NewRequest(curr.method, "/api/v1/films", nil)
		q := r.URL.Query()
		for key, value := range curr.params {
			q.Add(key, value)
		}
		r.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()

		api.Films(w, r)
		response, err := getResponse(w)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}
		if response.Status != curr.result.Status {
			t.Errorf("unexpected status: %d, want %d", response.Status, curr.result.Status)
			return
		}
		if !reflect.DeepEqual(response.Body, curr.result.Body) {
			t.Errorf("wanted %v, got %v", curr.result.Body, response.Body)
			return
		}
	}
}

func TestFilm(t *testing.T) {
	genreItem := models.GenreItem{Title: "g1"}
	expectedGenre := []models.GenreItem{genreItem}
	filmItem := models.FilmItem{Title: "t1"}
	expectedResponse := &requests.FilmResponse{
		Film:       filmItem,
		Genres:     expectedGenre,
		Rating:     9.5,
		Number:     10,
		Directors:  nil,
		Scenarists: nil,
		Characters: nil,
	}

	testCases := map[string]struct {
		method string
		params map[string]string
		result *requests.Response
	}{
		"Bad method": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusMethodNotAllowed, Body: nil},
		},
		"bad request error": {
			method: http.MethodGet,
			params: map[string]string{},
			result: &requests.Response{Status: http.StatusBadRequest, Body: nil},
		},
		"Core error": {
			method: http.MethodGet,
			params: map[string]string{"film_id": "1"},
			result: &requests.Response{Status: http.StatusInternalServerError, Body: nil},
		},
		"not found error": {
			method: http.MethodGet,
			params: map[string]string{"film_id": "2"},
			result: getExpectedResult(&requests.Response{Status: http.StatusNotFound, Body: nil}),
		},
		"Ok": {
			method: http.MethodGet,
			params: map[string]string{"film_id": "3"},
			result: getExpectedResult(&requests.Response{Status: http.StatusOK, Body: expectedResponse}),
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	mockCore := mocks.NewMockICore(mockCtrl)
	mockCore.EXPECT().GetFilmInfo(uint64(1)).Return(nil, fmt.Errorf("core_err")).Times(1)
	mockCore.EXPECT().GetFilmInfo(uint64(2)).Return(nil, usecase.ErrNotFound).Times(1)
	mockCore.EXPECT().GetFilmInfo(uint64(3)).Return(expectedResponse, nil).Times(1)

	api := API{core: mockCore, lg: logger, ct: collector}

	for _, curr := range testCases {
		r := httptest.NewRequest(curr.method, "/api/v1/film", nil)
		q := r.URL.Query()
		for key, value := range curr.params {
			q.Add(key, value)
		}
		r.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()

		api.Film(w, r)
		response, err := getResponse(w)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}
		if response.Status != curr.result.Status {
			t.Errorf("unexpected status: %d, want %d", response.Status, curr.result.Status)
			return
		}
		if !reflect.DeepEqual(response.Body, curr.result.Body) {
			t.Errorf("wanted %v, got %v", curr.result.Body, response.Body)
			return
		}
	}
}

func TestActor(t *testing.T) {
	careerItem := models.ProfessionItem{Title: "g1"}
	expectedCareer := []models.ProfessionItem{careerItem}
	expectedResponse := &requests.ActorResponse{
		Name:   "n",
		Career: expectedCareer,
	}

	testCases := map[string]struct {
		method string
		params map[string]string
		result *requests.Response
	}{
		"Bad method": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusMethodNotAllowed, Body: nil},
		},
		"bad request error": {
			method: http.MethodGet,
			params: map[string]string{},
			result: &requests.Response{Status: http.StatusBadRequest, Body: nil},
		},
		"Core error": {
			method: http.MethodGet,
			params: map[string]string{"actor_id": "1"},
			result: &requests.Response{Status: http.StatusInternalServerError, Body: nil},
		},
		"not found error": {
			method: http.MethodGet,
			params: map[string]string{"actor_id": "2"},
			result: &requests.Response{Status: http.StatusNotFound, Body: nil},
		},
		"Ok": {
			method: http.MethodGet,
			params: map[string]string{"actor_id": "3"},
			result: getExpectedResult(&requests.Response{Status: http.StatusOK, Body: expectedResponse}),
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCore := mocks.NewMockICore(mockCtrl)
	mockCore.EXPECT().GetActorInfo(uint64(1)).Return(nil, fmt.Errorf("core_err")).Times(1)
	mockCore.EXPECT().GetActorInfo(uint64(2)).Return(nil, usecase.ErrNotFound).Times(1)
	mockCore.EXPECT().GetActorInfo(uint64(3)).Return(expectedResponse, nil).Times(1)
	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	api := API{core: mockCore, lg: logger, ct: collector}

	for _, curr := range testCases {
		r := httptest.NewRequest(curr.method, "/api/v1/actor", nil)
		q := r.URL.Query()
		for key, value := range curr.params {
			q.Add(key, value)
		}
		r.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()

		api.Actor(w, r)
		response, err := getResponse(w)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}
		if response.Status != curr.result.Status {
			t.Errorf("unexpected status: %d, want %d", response.Status, curr.result.Status)
			return
		}
		if !reflect.DeepEqual(response.Body, curr.result.Body) {
			t.Errorf("wanted %v, got %v", curr.result.Body, response.Body)
			return
		}
	}
}

func TestFindFilm(t *testing.T) {
	filmItem := models.FilmItem{Title: "t3"}
	films := []models.FilmItem{filmItem}
	expectedResponse := requests.FilmsResponse{
		Films: films,
		Total: uint64(len(films)),
	}

	testCases := map[string]struct {
		method string
		body   io.Reader
		result *requests.Response
	}{
		"Bad method": {
			method: http.MethodGet,
			result: &requests.Response{Status: http.StatusMethodNotAllowed, Body: nil},
		},
		"bad request error": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusBadRequest, Body: nil},
			body:   nil,
		},
		"Core error": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusInternalServerError, Body: nil},
			body:   createBody(requests.FindFilmRequest{Title: "t1", Genres: nil, Actors: nil}),
		},
		"not found error": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusNotFound, Body: nil},
			body:   createBody(requests.FindFilmRequest{Title: "t2", Genres: nil, Actors: nil}),
		},
		"Ok": {
			method: http.MethodPost,
			result: getExpectedResult(&requests.Response{Status: http.StatusOK, Body: expectedResponse}),
			body:   createBody(requests.FindFilmRequest{Title: "t3", Genres: nil, Actors: nil}),
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCore := mocks.NewMockICore(mockCtrl)
	mockCore.EXPECT().FindFilm(string("t1"), string(""), string(""), float32(0), float32(0), string(""), nil, nil, uint64(0), uint64(0)).Return(nil, fmt.Errorf("core_err")).Times(1)
	mockCore.EXPECT().FindFilm(string("t2"), string(""), string(""), float32(0), float32(0), string(""), nil, nil, uint64(0), uint64(0)).Return(nil, usecase.ErrNotFound).Times(1)
	mockCore.EXPECT().FindFilm(string("t3"), string(""), string(""), float32(0), float32(0), string(""), nil, nil, uint64(0), uint64(0)).Return(films, nil).Times(1)
	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	api := API{core: mockCore, lg: logger, ct: collector}

	for _, curr := range testCases {
		r := httptest.NewRequest(curr.method, "/api/v1/search/film", curr.body)
		w := httptest.NewRecorder()

		api.FindFilm(w, r)
		response, err := getResponse(w)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}
		if response.Status != curr.result.Status {
			t.Errorf("unexpected status: %d, want %d", response.Status, curr.result.Status)
			return
		}
		if !reflect.DeepEqual(response.Body, curr.result.Body) {
			t.Errorf("wanted %v, got %v", curr.result.Body, response.Body)
			return
		}
	}
}

func TestFindActor(t *testing.T) {
	actorItem := models.Character{NameActor: "n1"}
	actors := []models.Character{actorItem}
	expectedResponse := requests.ActorsResponse{
		Actors: actors,
		Total:  1,
	}

	testCases := map[string]struct {
		method string
		body   io.Reader
		result *requests.Response
	}{
		"Bad method": {
			method: http.MethodGet,
			result: &requests.Response{Status: http.StatusMethodNotAllowed, Body: nil},
		},
		"bad request error": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusBadRequest, Body: nil},
			body:   nil,
		},
		"Core error": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusInternalServerError, Body: nil},
			body:   createActorBody(requests.FindActorRequest{Name: "n1", Career: nil, Films: nil}),
		},
		"not found error": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusNotFound, Body: nil},
			body:   createActorBody(requests.FindActorRequest{Name: "n2", Career: nil, Films: nil, Page: 2, PerPage: 1}),
		},
		"Ok": {
			method: http.MethodPost,
			result: getExpectedResult(&requests.Response{Status: http.StatusOK, Body: expectedResponse}),
			body:   createActorBody(requests.FindActorRequest{Name: "n3", Career: nil, Films: nil, Page: 1, PerPage: 1}),
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCore := mocks.NewMockICore(mockCtrl)
	mockCore.EXPECT().FindActor(string("n1"), string(""), nil, nil, string(""), uint64(0), uint64(0)).Return(nil, fmt.Errorf("core_err")).Times(1)
	mockCore.EXPECT().FindActor(string("n2"), string(""), nil, nil, string(""), uint64(1), uint64(1)).Return(nil, usecase.ErrNotFound).Times(1)
	mockCore.EXPECT().FindActor(string("n3"), string(""), nil, nil, string(""), uint64(0), uint64(1)).Return(actors, nil).Times(1)
	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	api := API{core: mockCore, lg: logger, ct: collector}

	for _, curr := range testCases {
		r := httptest.NewRequest(curr.method, "/api/v1/search/actor", curr.body)
		w := httptest.NewRecorder()

		api.FindActor(w, r)
		response, err := getResponse(w)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}
		if response.Status != curr.result.Status {
			t.Errorf("unexpected status: %d, want %d", response.Status, curr.result.Status)
			return
		}
		if !reflect.DeepEqual(response.Body, curr.result.Body) {
			t.Errorf("wanted %v, got %v", curr.result.Body, response.Body)
			return
		}
	}
}

func TestCalendar(t *testing.T) {
	expectedResponse := &requests.CalendarResponse{
		MonthName: "m",
		Days:      nil,
	}

	testCases := []struct {
		testName string
		method   string
		result   *requests.Response
	}{
		{
			testName: "Bad method",
			method:   http.MethodPost,
			result:   &requests.Response{Status: http.StatusMethodNotAllowed, Body: nil},
		},
		{
			testName: "Core error",
			method:   http.MethodGet,
			result:   &requests.Response{Status: http.StatusInternalServerError, Body: nil},
		},
		{
			testName: "Ok",
			method:   http.MethodGet,
			result:   getExpectedResult(&requests.Response{Status: http.StatusOK, Body: expectedResponse}),
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCore := mocks.NewMockICore(mockCtrl)
	mockCore.EXPECT().GetCalendar().Return(nil, fmt.Errorf("core_err")).Times(1)
	mockCore.EXPECT().GetCalendar().Return(expectedResponse, nil).Times(1)
	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	api := API{core: mockCore, lg: logger, ct: collector}

	for _, curr := range testCases {
		r := httptest.NewRequest(curr.method, "/api/v1/calendar", nil)
		w := httptest.NewRecorder()

		api.Calendar(w, r)
		response, err := getResponse(w)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}
		if response.Status != curr.result.Status {
			t.Errorf("unexpected status: %d, want %d", response.Status, curr.result.Status)
			return
		}
		if !reflect.DeepEqual(response.Body, curr.result.Body) {
			t.Errorf("wanted %v, got %v", curr.result.Body, response.Body)
			return
		}
	}
}

func TestFavoriteFilmsAdd(t *testing.T) {
	testCases := map[string]struct {
		method string
		params map[string]string
		result *requests.Response
	}{
		"Bad method": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusMethodNotAllowed, Body: nil},
		},
		"bad request error": {
			method: http.MethodGet,
			params: map[string]string{},
			result: &requests.Response{Status: http.StatusBadRequest, Body: nil},
		},
		"Core error": {
			method: http.MethodGet,
			params: map[string]string{"film_id": "1"},
			result: &requests.Response{Status: http.StatusInternalServerError, Body: nil},
		},
		"found error": {
			method: http.MethodGet,
			params: map[string]string{"film_id": "2"},
			result: &requests.Response{Status: http.StatusNotAcceptable, Body: nil},
		},
		"Ok": {
			method: http.MethodGet,
			params: map[string]string{"film_id": "3"},
			result: &requests.Response{Status: http.StatusOK, Body: nil},
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCore := mocks.NewMockICore(mockCtrl)
	mockCore.EXPECT().FavoriteFilmsAdd(uint64(1), uint64(1)).Return(fmt.Errorf("core_err")).Times(1)
	mockCore.EXPECT().FavoriteFilmsAdd(uint64(1), uint64(2)).Return(usecase.ErrFoundFavorite).Times(1)
	mockCore.EXPECT().FavoriteFilmsAdd(uint64(1), uint64(3)).Return(nil).Times(1)
	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	api := API{core: mockCore, lg: logger, ct: collector}

	for _, curr := range testCases {
		r := httptest.NewRequest(curr.method, "/api/v1/favorite/film/add", nil)
		newReq := r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, uint64(1)))
		q := r.URL.Query()
		for key, value := range curr.params {
			q.Add(key, value)
		}
		newReq.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()

		api.FavoriteFilmsAdd(w, newReq)
		response, err := getResponse(w)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}
		if response.Status != curr.result.Status {
			t.Errorf("unexpected status: %d, want %d", response.Status, curr.result.Status)
			return
		}
		if !reflect.DeepEqual(response.Body, curr.result.Body) {
			t.Errorf("wanted %v, got %v", curr.result.Body, response.Body)
			return
		}
	}
}

func TestFavoriteFilmsRemove(t *testing.T) {
	testCases := map[string]struct {
		method string
		params map[string]string
		result *requests.Response
	}{
		"Bad method": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusMethodNotAllowed, Body: nil},
		},
		"bad request error": {
			method: http.MethodGet,
			params: map[string]string{},
			result: &requests.Response{Status: http.StatusBadRequest, Body: nil},
		},
		"Core error": {
			method: http.MethodGet,
			params: map[string]string{"film_id": "1"},
			result: &requests.Response{Status: http.StatusInternalServerError, Body: nil},
		},
		"Ok": {
			method: http.MethodGet,
			params: map[string]string{"film_id": "3"},
			result: &requests.Response{Status: http.StatusOK, Body: nil},
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCore := mocks.NewMockICore(mockCtrl)
	mockCore.EXPECT().FavoriteFilmsRemove(uint64(1), uint64(1)).Return(fmt.Errorf("core_err")).Times(1)
	mockCore.EXPECT().FavoriteFilmsRemove(uint64(1), uint64(3)).Return(nil).Times(1)
	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	api := API{core: mockCore, lg: logger, ct: collector}

	for _, curr := range testCases {
		r := httptest.NewRequest(curr.method, "/api/v1/favorite/film/remove", nil)
		newReq := r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, uint64(1)))
		q := r.URL.Query()
		for key, value := range curr.params {
			q.Add(key, value)
		}
		newReq.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()

		api.FavoriteFilmsRemove(w, newReq)
		response, err := getResponse(w)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}
		if response.Status != curr.result.Status {
			t.Errorf("unexpected status: %d, want %d", response.Status, curr.result.Status)
			return
		}
		if !reflect.DeepEqual(response.Body, curr.result.Body) {
			t.Errorf("wanted %v, got %v", curr.result.Body, response.Body)
			return
		}
	}
}

func TestFavoriteFilms(t *testing.T) {
	filmItem := models.FilmItem{Title: "t"}
	films := []models.FilmItem{filmItem}

	testCases := map[string]struct {
		method string
		params map[string]string
		result *requests.Response
	}{
		"Bad method": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusMethodNotAllowed, Body: nil},
		},
		"Core error": {
			method: http.MethodGet,
			params: map[string]string{"page": "2", "page_size": "8"},
			result: &requests.Response{Status: http.StatusInternalServerError, Body: nil},
		},
		"Ok": {
			method: http.MethodGet,
			params: map[string]string{"page": "1", "page_size": "8"},
			result: getExpectedResult(&requests.Response{Status: http.StatusOK, Body: films}),
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCore := mocks.NewMockICore(mockCtrl)
	mockCore.EXPECT().FavoriteFilms(uint64(1), uint64(8), uint64(8)).Return(nil, fmt.Errorf("core_err")).Times(1)
	mockCore.EXPECT().FavoriteFilms(uint64(1), uint64(0), uint64(8)).Return(films, nil).Times(1)
	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	api := API{core: mockCore, lg: logger, ct: collector}

	for _, curr := range testCases {
		r := httptest.NewRequest(curr.method, "/api/v1/favorite/films", nil)
		newReq := r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, uint64(1)))
		q := r.URL.Query()
		for key, value := range curr.params {
			q.Add(key, value)
		}
		newReq.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()

		api.FavoriteFilms(w, newReq)
		response, err := getResponse(w)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}
		if response.Status != curr.result.Status {
			t.Errorf("unexpected status: %d, want %d", response.Status, curr.result.Status)
			return
		}
		if !reflect.DeepEqual(response.Body, curr.result.Body) {
			t.Errorf("wanted %v, got %v", curr.result.Body, response.Body)
			return
		}
	}
}

func TestFavoriteActorsAdd(t *testing.T) {
	testCases := map[string]struct {
		method string
		params map[string]string
		result *requests.Response
	}{
		"Bad method": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusMethodNotAllowed, Body: nil},
		},
		"bad request error": {
			method: http.MethodGet,
			params: map[string]string{},
			result: &requests.Response{Status: http.StatusBadRequest, Body: nil},
		},
		"Core error": {
			method: http.MethodGet,
			params: map[string]string{"actor_id": "1"},
			result: &requests.Response{Status: http.StatusInternalServerError, Body: nil},
		},
		"found error": {
			method: http.MethodGet,
			params: map[string]string{"actor_id": "2"},
			result: &requests.Response{Status: http.StatusNotAcceptable, Body: nil},
		},
		"Ok": {
			method: http.MethodGet,
			params: map[string]string{"actor_id": "3"},
			result: &requests.Response{Status: http.StatusOK, Body: nil},
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCore := mocks.NewMockICore(mockCtrl)
	mockCore.EXPECT().FavoriteActorsAdd(uint64(1), uint64(1)).Return(fmt.Errorf("core_err")).Times(1)
	mockCore.EXPECT().FavoriteActorsAdd(uint64(1), uint64(2)).Return(usecase.ErrFoundFavorite).Times(1)
	mockCore.EXPECT().FavoriteActorsAdd(uint64(1), uint64(3)).Return(nil).Times(1)
	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	api := API{core: mockCore, lg: logger, ct: collector}

	for _, curr := range testCases {
		r := httptest.NewRequest(curr.method, "/api/v1/favorite/actor/add", nil)
		newReq := r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, uint64(1)))
		q := r.URL.Query()
		for key, value := range curr.params {
			q.Add(key, value)
		}
		newReq.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()

		api.FavoriteActorsAdd(w, newReq)
		response, err := getResponse(w)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}
		if response.Status != curr.result.Status {
			t.Errorf("unexpected status: %d, want %d", response.Status, curr.result.Status)
			return
		}
		if !reflect.DeepEqual(response.Body, curr.result.Body) {
			t.Errorf("wanted %v, got %v", curr.result.Body, response.Body)
			return
		}
	}
}

func TestFavoriteActorsRemove(t *testing.T) {
	testCases := map[string]struct {
		method string
		params map[string]string
		result *requests.Response
	}{
		"Bad method": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusMethodNotAllowed, Body: nil},
		},
		"bad request error": {
			method: http.MethodGet,
			params: map[string]string{},
			result: &requests.Response{Status: http.StatusBadRequest, Body: nil},
		},
		"Core error": {
			method: http.MethodGet,
			params: map[string]string{"actor_id": "1"},
			result: &requests.Response{Status: http.StatusInternalServerError, Body: nil},
		},
		"Ok": {
			method: http.MethodGet,
			params: map[string]string{"actor_id": "3"},
			result: &requests.Response{Status: http.StatusOK, Body: nil},
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCore := mocks.NewMockICore(mockCtrl)
	mockCore.EXPECT().FavoriteActorsRemove(uint64(1), uint64(1)).Return(fmt.Errorf("core_err")).Times(1)
	mockCore.EXPECT().FavoriteActorsRemove(uint64(1), uint64(3)).Return(nil).Times(1)
	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	api := API{core: mockCore, lg: logger, ct: collector}

	for _, curr := range testCases {
		r := httptest.NewRequest(curr.method, "/api/v1/favorite/actor/remove", nil)
		newReq := r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, uint64(1)))
		q := r.URL.Query()
		for key, value := range curr.params {
			q.Add(key, value)
		}
		newReq.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()

		api.FavoriteActorsRemove(w, newReq)
		response, err := getResponse(w)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}
		if response.Status != curr.result.Status {
			t.Errorf("unexpected status: %d, want %d", response.Status, curr.result.Status)
			return
		}
		if !reflect.DeepEqual(response.Body, curr.result.Body) {
			t.Errorf("wanted %v, got %v", curr.result.Body, response.Body)
			return
		}
	}
}

func TestFavoriteActors(t *testing.T) {
	actorItem := models.Character{NameActor: "n"}
	actors := []models.Character{actorItem}
	response := requests.ActorsResponse{Actors: actors}

	testCases := map[string]struct {
		method string
		params map[string]string
		result *requests.Response
	}{
		"Bad method": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusMethodNotAllowed, Body: nil},
		},
		"Core error": {
			method: http.MethodGet,
			params: map[string]string{"page": "2", "page_size": "8"},
			result: &requests.Response{Status: http.StatusInternalServerError, Body: nil},
		},
		"Ok": {
			method: http.MethodGet,
			params: map[string]string{"page": "1", "page_size": "8"},
			result: getExpectedResult(&requests.Response{Status: http.StatusOK, Body: response}),
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCore := mocks.NewMockICore(mockCtrl)
	mockCore.EXPECT().FavoriteActors(uint64(1), uint64(8), uint64(8)).Return(nil, fmt.Errorf("core_err")).Times(1)
	mockCore.EXPECT().FavoriteActors(uint64(1), uint64(0), uint64(8)).Return(actors, nil).Times(1)
	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	api := API{core: mockCore, lg: logger, ct: collector}

	for _, curr := range testCases {
		r := httptest.NewRequest(curr.method, "/api/v1/favorite/actors", nil)
		newReq := r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, uint64(1)))
		q := r.URL.Query()
		for key, value := range curr.params {
			q.Add(key, value)
		}
		newReq.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()

		api.FavoriteActors(w, newReq)
		response, err := getResponse(w)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}
		if response.Status != curr.result.Status {
			t.Errorf("unexpected status: %d, want %d", response.Status, curr.result.Status)
			return
		}
		if !reflect.DeepEqual(response.Body, curr.result.Body) {
			t.Errorf("wanted %v, got %v", curr.result.Body, response.Body)
			return
		}
	}
}

func TestAddRating(t *testing.T) {
	testCases := map[string]struct {
		method string
		result *requests.Response
		body   io.Reader
	}{
		"Bad method": {
			method: http.MethodGet,
			result: &requests.Response{Status: http.StatusMethodNotAllowed, Body: nil},
		},
		"no body error": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusBadRequest, Body: nil},
			body:   nil,
		},
		"Core error": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusInternalServerError, Body: nil},
			body:   createRatingBody(requests.CommentRequest{FilmId: 1}),
		},
		"found error": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusNotAcceptable, Body: nil},
			body:   createRatingBody(requests.CommentRequest{FilmId: 2}),
		},
		"Ok": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusOK, Body: nil},
			body:   createRatingBody(requests.CommentRequest{FilmId: 3}),
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCore := mocks.NewMockICore(mockCtrl)
	mockCore.EXPECT().AddRating(uint64(1), uint64(1), uint16(0)).Return(false, fmt.Errorf("core_err")).Times(1)
	mockCore.EXPECT().AddRating(uint64(2), uint64(1), uint16(0)).Return(true, nil).Times(1)
	mockCore.EXPECT().AddRating(uint64(3), uint64(1), uint16(0)).Return(false, nil).Times(1)
	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	api := API{core: mockCore, lg: logger, ct: collector}

	for _, curr := range testCases {
		r := httptest.NewRequest(curr.method, "/api/v1/rating/add", curr.body)
		newReq := r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, uint64(1)))

		w := httptest.NewRecorder()

		api.AddRating(w, newReq)
		response, err := getResponse(w)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}
		if response.Status != curr.result.Status {
			t.Errorf("unexpected status: %d, want %d", response.Status, curr.result.Status)
			return
		}
		if !reflect.DeepEqual(response.Body, curr.result.Body) {
			t.Errorf("wanted %v, got %v", curr.result.Body, response.Body)
			return
		}
	}
}

func TestDeleteRating(t *testing.T) {
	testCases := map[string]struct {
		method string
		result *requests.Response
		body   io.Reader
	}{
		"Bad method": {
			method: http.MethodGet,
			result: &requests.Response{Status: http.StatusMethodNotAllowed, Body: nil},
		},
		"no body error": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusBadRequest, Body: nil},
			body:   nil,
		},
		"Core error": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusInternalServerError, Body: nil},
			body:   createDelRatingBody(requests.DeleteCommentRequest{IdUser: 5, IdFilm: 1}),
		},
		"Ok": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusOK, Body: nil},
			body:   createDelRatingBody(requests.DeleteCommentRequest{IdUser: 5, IdFilm: 2}),
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCore := mocks.NewMockICore(mockCtrl)
	mockCore.EXPECT().DeleteRating(uint64(5), uint64(1)).Return(fmt.Errorf("core_err")).Times(1)
	mockCore.EXPECT().DeleteRating(uint64(5), uint64(2)).Return(nil).Times(1)
	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	api := API{core: mockCore, lg: logger, ct: collector}

	for _, curr := range testCases {
		r := httptest.NewRequest(curr.method, "/api/v1/rating/delete", curr.body)
		newReq := r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, uint64(1)))

		w := httptest.NewRecorder()

		api.DeleteRating(w, newReq)
		response, err := getResponse(w)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}
		if response.Status != curr.result.Status {
			t.Errorf("unexpected status: %d, want %d", response.Status, curr.result.Status)
			return
		}
		if !reflect.DeepEqual(response.Body, curr.result.Body) {
			t.Errorf("wanted %v, got %v", curr.result.Body, response.Body)
			return
		}
	}
}

func TestUsersStatistics(t *testing.T) {
	expectItem := requests.UsersStatisticsResponse{GenreId: 1, Count: 1, Avg: 1}
	expect := []requests.UsersStatisticsResponse{expectItem}
	testCases := map[string]struct {
		method string
		result *requests.Response
		userId uint64
	}{
		"Bad method": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusMethodNotAllowed, Body: nil},
			userId: 1,
		},
		"Core error": {
			method: http.MethodGet,
			result: &requests.Response{Status: http.StatusInternalServerError, Body: nil},
			userId: 2,
		},
		"Ok": {
			method: http.MethodGet,
			result: getExpectedResult(&requests.Response{Status: http.StatusOK, Body: expect}),
			userId: 3,
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCore := mocks.NewMockICore(mockCtrl)
	mockCore.EXPECT().UsersStatistics(uint64(2)).Return(nil, fmt.Errorf("core_err")).Times(1)
	mockCore.EXPECT().UsersStatistics(uint64(3)).Return(expect, nil).Times(1)
	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	api := API{core: mockCore, lg: logger, ct: collector}

	for _, curr := range testCases {
		r := httptest.NewRequest(curr.method, "/api/v1/statistics", nil)
		newReq := r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, curr.userId))

		w := httptest.NewRecorder()

		api.UsersStatistics(w, newReq)
		response, err := getResponse(w)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}
		if response.Status != curr.result.Status {
			t.Errorf("unexpected status: %d, want %d", response.Status, curr.result.Status)
			return
		}
		if !reflect.DeepEqual(response.Body, curr.result.Body) {
			t.Errorf("wanted %v, got %v", curr.result.Body, response.Body)
			return
		}
	}
}

func TestTrends(t *testing.T) {
	expectItem := models.FilmItem{Title: "t1"}
	expect := []models.FilmItem{expectItem}
	expectResponse := requests.FilmsResponse{Films: expect, Total: uint64(len(expect))}
	testCases := []struct {
		testName string
		method   string
		result   *requests.Response
	}{
		{
			testName: "Bad method",
			method:   http.MethodPost,
			result:   &requests.Response{Status: http.StatusMethodNotAllowed, Body: nil},
		},
		{
			testName: "Core error",
			method:   http.MethodGet,
			result:   &requests.Response{Status: http.StatusInternalServerError, Body: nil},
		},
		{
			testName: "Ok",
			method:   http.MethodGet,
			result:   getExpectedResult(&requests.Response{Status: http.StatusOK, Body: expectResponse}),
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCore := mocks.NewMockICore(mockCtrl)
	mockCore.EXPECT().Trends().Return(nil, fmt.Errorf("core_err")).Times(1)
	mockCore.EXPECT().Trends().Return(expect, nil).Times(1)

	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	api := API{core: mockCore, lg: logger, ct: collector}

	for _, curr := range testCases {
		r := httptest.NewRequest(curr.method, "/api/v1/trends", nil)
		w := httptest.NewRecorder()

		api.Trends(w, r)
		response, err := getResponse(w)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}
		if response.Status != curr.result.Status {
			t.Errorf("unexpected status: %d, want %d", response.Status, curr.result.Status)
			return
		}
		if !reflect.DeepEqual(response.Body, curr.result.Body) {
			t.Errorf("wanted %v, got %v", curr.result.Body, response.Body)
			return
		}
	}
}

func TestLastSeen(t *testing.T) {
	expectNear := []models.NearFilm{{IdFilm: 1, IdUser: 5}}
	expectBadNear := []models.NearFilm{{IdFilm: 1, IdUser: 3}}
	expect := []models.FilmItem{{Title: "t1"}}
	expectResponse := requests.FilmsResponse{
		Films: expect,
		Total: uint64(len(expect)),
	}
	testCases := map[string]struct {
		method         string
		result         *requests.Response
		userId         uint64
		nearFilmResult []models.NearFilm
		nearFilmErr    error
		lastSeenResult []models.FilmItem
		lastSeenError  error
	}{
		"Bad method": {
			method: http.MethodPost,
			result: &requests.Response{Status: http.StatusMethodNotAllowed, Body: nil},
			userId: 1,
		},
		"GetNearFilms error": {
			method:         http.MethodGet,
			result:         &requests.Response{Status: http.StatusInternalServerError, Body: nil},
			userId:         2,
			nearFilmResult: nil,
			nearFilmErr:    fmt.Errorf("core_err"),
		},
		"GetLastSeen error": {
			method:         http.MethodGet,
			result:         &requests.Response{Status: http.StatusInternalServerError, Body: nil},
			userId:         3,
			nearFilmResult: expectBadNear,
			nearFilmErr:    nil,
			lastSeenResult: nil,
			lastSeenError:  fmt.Errorf("core_err"),
		},
		"not found error": {
			method:         http.MethodGet,
			result:         &requests.Response{Status: http.StatusNotFound, Body: nil},
			userId:         4,
			nearFilmResult: []models.NearFilm{},
			nearFilmErr:    nil,
			lastSeenResult: nil,
			lastSeenError:  usecase.ErrNotFound,
		},
		"Ok": {
			method:         http.MethodGet,
			result:         getExpectedResult(&requests.Response{Status: http.StatusOK, Body: expectResponse}),
			userId:         5,
			nearFilmResult: expectNear,
			nearFilmErr:    nil,
			lastSeenResult: expect,
			lastSeenError:  nil,
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	mockCore := mocks.NewMockICore(mockCtrl)

	api := API{core: mockCore, lg: logger, ct: collector}

	for _, curr := range testCases {
		r := httptest.NewRequest(curr.method, "/api/v1/lasts", nil)
		newReq := r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, curr.userId))

		mockCore.EXPECT().GetNearFilms(newReq.Context(), curr.userId, logger).Return(curr.nearFilmResult, curr.nearFilmErr).MaxTimes(1)
		mockCore.EXPECT().GetLastSeen(curr.nearFilmResult).Return(curr.lastSeenResult, curr.lastSeenError).MaxTimes(1)

		w := httptest.NewRecorder()

		api.LastSeen(w, newReq)
		response, err := getResponse(w)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}
		if response.Status != curr.result.Status {
			t.Errorf("unexpected status: %d, want %d", response.Status, curr.result.Status)
			return
		}
		if !reflect.DeepEqual(response.Body, curr.result.Body) {
			t.Errorf("wanted %v, got %v", curr.result.Body, response.Body)
			return
		}
	}
}
