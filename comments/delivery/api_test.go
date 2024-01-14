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

	"github.com/go-park-mail-ru/2023_2_Vkladyshi/comments/mocks"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/middleware"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/requests"
	"github.com/golang/mock/gomock"
	"github.com/mailru/easyjson"
)

func getResponse(w *httptest.ResponseRecorder) (*requests.Response, error) {
	var response requests.Response

	body, _ := io.ReadAll(w.Body)
	err := easyjson.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("cant unmarshal jsone")
	}

	return &response, nil
}

func createDelRatingBody(req requests.DeleteCommentRequest) io.Reader {
	jsonReq, _ := easyjson.Marshal(req)

	body := bytes.NewBuffer(jsonReq)
	return body
}

func createBody(req requests.CommentRequest) io.Reader {
	jsonReq, _ := easyjson.Marshal(req)

	body := bytes.NewBuffer(jsonReq)
	return body
}

var collector *requests.Collector = requests.GetCollector()

func TestComment(t *testing.T) {
	testCases := map[string]struct {
		method string
		params map[string]string
		result requests.Response
	}{
		"Bad method": {
			method: http.MethodPost,
			params: map[string]string{},
			result: requests.Response{Status: http.StatusMethodNotAllowed, Body: nil},
		},
		"No Film": {
			method: http.MethodGet,
			params: map[string]string{},
			result: requests.Response{Status: http.StatusBadRequest, Body: nil},
		},
		"Core error": {
			method: http.MethodGet,
			params: map[string]string{"film_id": "0"},
			result: requests.Response{Status: http.StatusInternalServerError, Body: nil},
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCore := mocks.NewMockICore(mockCtrl)
	mockCore.EXPECT().GetFilmComments(uint64(0), uint64(0), uint64(10)).Return(nil, fmt.Errorf("core_err")).Times(1)
	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	api := API{core: mockCore, lg: logger, ct: collector}

	for _, curr := range testCases {
		r := httptest.NewRequest(curr.method, "/api/v1/comment", nil)
		q := r.URL.Query()
		for key, value := range curr.params {
			q.Add(key, value)
		}
		r.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()

		api.Comment(w, r)
		response, err := getResponse(w)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}
		if response.Status != curr.result.Status {
			t.Errorf("unexpected status: %d", response.Status)
			return
		}
		if !reflect.DeepEqual(response.Body, curr.result.Body) {
			t.Errorf("wanted %v, got %v", curr.result.Body, response.Body)
			return
		}
	}
}

func TestCommentAdd(t *testing.T) {
	testCases := map[string]struct {
		method string
		result requests.Response
		body   io.Reader
	}{
		"Bad method": {
			method: http.MethodGet,
			result: requests.Response{Status: http.StatusMethodNotAllowed, Body: nil},
			body:   nil,
		},
		"no body error": {
			method: http.MethodPost,
			result: requests.Response{Status: http.StatusBadRequest, Body: nil},
			body:   nil,
		},
		"add comment error": {
			method: http.MethodPost,
			result: requests.Response{Status: http.StatusInternalServerError, Body: nil},
			body:   createBody(requests.CommentRequest{Rating: 10, FilmId: 1, Text: ""}),
		},
		"found error": {
			method: http.MethodPost,
			result: requests.Response{Status: http.StatusNotAcceptable, Body: nil},
			body:   createBody(requests.CommentRequest{Rating: 10, FilmId: 2, Text: ""}),
		},
		"Ok": {
			method: http.MethodPost,
			result: requests.Response{Status: http.StatusOK, Body: nil},
			body:   createBody(requests.CommentRequest{Rating: 10, FilmId: 3, Text: ""}),
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCore := mocks.NewMockICore(mockCtrl)
	mockCore.EXPECT().AddComment(uint64(1), uint64(1), uint16(10), string("")).Return(false, fmt.Errorf("core_err")).Times(1)
	mockCore.EXPECT().AddComment(uint64(2), uint64(1), uint16(10), string("")).Return(true, nil).Times(1)
	mockCore.EXPECT().AddComment(uint64(3), uint64(1), uint16(10), string("")).Return(false, nil).Times(1)
	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	api := API{core: mockCore, lg: logger, ct: collector}

	for _, curr := range testCases {
		r := httptest.NewRequest(curr.method, "/api/v1/comment/add", curr.body)
		newReq := r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, uint64(1)))
		w := httptest.NewRecorder()

		api.AddComment(w, newReq)
		response, err := getResponse(w)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}
		if response.Status != curr.result.Status {
			fmt.Println(api.lg)
			t.Errorf("unexpected status: %d, wanted: %d", response.Status, curr.result.Status)
			return
		}
		if response.Body != nil {
			t.Errorf("unexpected body %v", response.Body)
			return
		}
	}
}

func TestDeleteComment(t *testing.T) {
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
	mockCore.EXPECT().DeleteComment(uint64(5), uint64(1)).Return(fmt.Errorf("core_err")).Times(1)
	mockCore.EXPECT().DeleteComment(uint64(5), uint64(2)).Return(nil).Times(1)
	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	api := API{core: mockCore, lg: logger, ct: collector}

	for _, curr := range testCases {
		r := httptest.NewRequest(curr.method, "/api/v1/comment/delete", curr.body)
		newReq := r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, uint64(1)))

		w := httptest.NewRecorder()

		api.DeleteComment(w, newReq)
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
