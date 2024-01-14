package usecase

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/go-park-mail-ru/2023_2_Vkladyshi/comments/mocks"
	"github.com/golang/mock/gomock"
)

func TestAddComment(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockObj := mocks.NewMockICommentRepo(mockCtrl)
	mockObj.EXPECT().HasUsersComment(uint64(1), uint64(1)).Return(false, nil)
	mockObj.EXPECT().HasUsersComment(uint64(0), uint64(1)).Return(false, fmt.Errorf("repo_error"))
	mockObj.EXPECT().HasUsersComment(uint64(2), uint64(1)).Return(true, nil)
	mockObj.EXPECT().HasUsersComment(uint64(2), uint64(2)).Return(false, nil)
	mockObj.EXPECT().AddComment(uint64(1), uint64(1), uint16(1), string("t")).Return(nil)
	mockObj.EXPECT().AddComment(uint64(2), uint64(2), uint16(1), string("t")).Return(fmt.Errorf("repo_error"))

	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))
	core := Core{comments: mockObj, lg: logger}

	found, err := core.AddComment(1, 1, 1, "t")
	if err != nil {
		t.Errorf("waited no errors")
		return
	}
	if found {
		t.Errorf("waited not found")
		return
	}

	found, err = core.AddComment(1, 0, 1, "t")
	if err == nil {
		t.Errorf("waited error")
		return
	}
	if found {
		t.Errorf("waited not found")
		return
	}

	found, err = core.AddComment(1, 2, 1, "t")
	if err != nil {
		t.Errorf("waited no errors")
		return
	}
	if !found {
		t.Errorf("waited to find")
		return
	}

	found, err = core.AddComment(2, 2, 1, "t")
	if err == nil {
		t.Errorf("waited find error")
		return
	}
	if found {
		t.Errorf("waited not found")
		return
	}
}

func TestDeleteComment(t *testing.T) {
	testCases := map[string]struct {
		err error
	}{
		"repo error": {
			err: fmt.Errorf("repo err"),
		},
		"OK": {
			err: nil,
		},
	}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockObj := mocks.NewMockICommentRepo(mockCtrl)

	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	core := Core{comments: mockObj, lg: logger}

	for _, curr := range testCases {
		mockObj.EXPECT().DeleteComment(uint64(1), uint64(1)).Return(curr.err).Times(1)

		err := core.DeleteComment(1, 1)
		if !errors.Is(err, curr.err) {
			t.Errorf("Unexpected error. wanted %s, got %s", curr.err, err)
		}
	}
}
