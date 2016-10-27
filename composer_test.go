package batch

import (
	"bytes"
	"net/http"
	"testing"
)

type testResponse struct {
	Status  int
	Body    string
	Headers map[string]string
}

type testHandler struct {
	routes map[string]*testResponse
}

func (h *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if resp, ok := h.routes[r.URL.Path]; ok {
		for key, value := range resp.Headers {
			w.Header().Set(key, value)
		}
		w.WriteHeader(resp.Status)
		if resp.Body != "" {
			w.Write([]byte(resp.Body))
		}
	} else {
		w.WriteHeader(404)
	}
}

func TestGetResponses(t *testing.T) {
	th := &testHandler{
		routes: map[string]*testResponse{
			"/api/a": &testResponse{200, "A", nil},
			"/api/b": &testResponse{200, "B", nil},
		},
	}
	handler := &Handler{th}

	r, err := http.NewRequest("POST", "http://localhost/api/batch", bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("Error creating request: %v", err)
	}
	req := []*request{
		{Path: "/api/a"},
		{Path: "/wasd1"},
		{Path: "/api/b"},
		{Path: "/wasd2"},
	}
	res := handler.getResponses(r, req)

	want := []*response{
		{Status: 200, Body: "A"},
		{Status: 404, Body: ""},
		{Status: 200, Body: "B"},
		{Status: 404, Body: ""},
	}

	if nGot, nWant := len(res), len(want); nGot != nWant {
		t.Errorf("Wrong number of responses. Got %d, want %d", nGot, nWant)
	} else {
		for i, got := range res {
			want := want[i]
			if want.Status != got.Status {
				t.Errorf("Wrong status for response to %s. Got %d, want %d", req[i].Path, got.Status, want.Status)
			}
			if want.Body != got.Body {
				t.Errorf("Wrong body for response to %s. Got '%s', want '%s'", req[i].Path, got.Body, want.Body)
			}
		}
	}
}
