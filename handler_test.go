package batch

import (
	"bytes"
	"net/http"
	"regexp"
	"strings"
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
	if resp, ok := h.routes[r.Method+":"+r.URL.Path]; ok {
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

func getTestRequest(method, url, body string) *http.Request {
	r, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		panic(err.Error)
	}
	return r
}

func (rw *rw) BodyStr() string {
	if rw.buf != nil && rw.buf.Len() > 0 {
		return rw.buf.String()
	}
	return ""
}

func TestServeHTTP(t *testing.T) {

	{ // Serves the inner HTTP handler's content if no match
		th := &testHandler{
			routes: map[string]*testResponse{
				"GET:/a": &testResponse{206, "A", map[string]string{"Header-Key": "v1"}},
			},
		}
		matchRequest := func(req *http.Request) bool { return false }
		handler := &Handler{
			NormalHandler: th,
			MatchRequest:  matchRequest,
		}
		var rw rw
		handler.ServeHTTP(&rw, getTestRequest("GET", "http://localhost/a", ""))

		want := &testResponse{206, "A", map[string]string{"Header-Key": "v1"}}
		if want, got := want.Status, rw.status; want != got {
			t.Errorf("Wrong status returned for non-batch request. Want %d, got %d", want, got)
		}
		if want, got := want.Body, rw.BodyStr(); want != got {
			t.Errorf("Wrong body returned for non-batch request. Want %s, got %s", want, got)
		}
		if want, got := want.Headers, rw.header; len(want) != len(got) {
			t.Errorf("Wrong headers returned for non-batch request. Want %v, got %v", want, got)
		} else {
			for key, want := range want {
				got := got[key]
				if len(got) != 1 {
					t.Errorf("Wrong headers returned for non-batch request. Want %v, got %v", want, got)
				} else if got[0] != want {
					t.Errorf("Wrong headers returned for non-batch request. Want %v, got %v", want, got)
					break
				}
			}
		}
	}

	// If batching:

	{ // Returns 500 status if body reader returns error
		// TODO: implement test
	}

	{ // Returns 400 status if invalid JSON
		// TODO: implement test
	}

	{ // Validates request
		// TODO: implement test
	}

	{ // runs the batched requests and writes their response
		th := &testHandler{
			routes: map[string]*testResponse{
				"GET:/a": &testResponse{200, "A", map[string]string{"H1": "hA"}},
				"PUT:/b": &testResponse{201, "B", map[string]string{"H1": "hB"}},
			},
		}
		matchRequest := func(req *http.Request) bool { return true }
		handler := &Handler{
			NormalHandler: th,
			MatchRequest:  matchRequest,
		}

		json :=
			`[
				{ "method": "GET", "path":"/a"},
				{ "method": "PUT", "path":"/b"}
			]`
		body := strings.NewReader(json)
		r, err := http.NewRequest("POST", "http://localhost/api/compose", body)
		if err != nil {
			t.Fatalf("%v", err)
		}

		var rw rw
		handler.ServeHTTP(&rw, r)
		if rw.status != 200 {
			t.Fatalf("ServeHTTP for batch request wrote status %d, want 200", rw.status)
		}
		if len(rw.header) > 0 {
			t.Fatalf("ServeHTTP for batch request wrote header. Wnat no headers", rw.header)
		}

		want := `[
			{
				"method": "GET",
				"path": "/a",
				"status": 200,
				"headers": {
					"H1": ["hA"]
				},
				"body": "A"
			},
			{
				"method": "PUT",
				"path": "/b",
				"status": 201,
				"headers": {
					"H1": ["hB"]
				},
				"body": "B"
			}
		]`
		want = regexp.MustCompile(`[\n\t ]*`).ReplaceAllString(want, "")
		if got := rw.BodyStr(); want != got {
			t.Errorf("ServeHTTP for batch request wrote wrong body. Want %s, got %s", want, got)
		}
	}
}

func TestGetResponses(t *testing.T) {
	th := &testHandler{
		routes: map[string]*testResponse{
			"GET:/a": &testResponse{200, "A", nil},
			"PUT:/b": &testResponse{200, "B", nil},
		},
	}
	handler := &Handler{
		NormalHandler: th,
	}

	r, err := http.NewRequest("POST", "http://localhost/api/batch", bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("Error creating request: %v", err)
	}
	req := []*request{
		{Method: "GET", Path: "/a"},
		{Method: "GET", Path: "/1"},
		{Method: "PUT", Path: "/b"},
		{Method: "PUT", Path: "/2"},
	}
	res := handler.getResponses(r, req)

	want := []*response{
		{Method: "GET", Path: "/a", Status: 200, Body: "A"},
		{Method: "GET", Path: "/1", Status: 404, Body: ""},
		{Method: "PUT", Path: "/b", Status: 200, Body: "B"},
		{Method: "PUT", Path: "/2", Status: 404, Body: ""},
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
