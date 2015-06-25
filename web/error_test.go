package web

import (
	"net/http/httptest"
	"testing"
)

func TestWriteError(t *testing.T) {
	errors := []*Error{ErrBadRequest, ErrMissingPartRequest, ErrAuth, ErrCredentials, ErrNotAcceptable,
		ErrUnsupportedMediaType, ErrCSRF, ErrForbidden, ErrInternalServer}
	for _, e := range errors {
		w := httptest.NewRecorder()
		WriteError(w, e)
		if w.Code != e.Status {
			t.Fatalf("Wrong status code %d, expected %d", w.Code, e.Status)
		}
	}
}
