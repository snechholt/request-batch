package batch

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

type Handler struct {
	NormalHandler http.Handler
	MatchRequest  func(r *http.Request) bool
}

type request struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type response struct {
	Method  string              `json:"method"`
	Path    string              `json:"path"`
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers"`
	Body    string              `json:"body"`
}

func (c *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !c.MatchRequest(r) {
		c.NormalHandler.ServeHTTP(w, r)
		return
	}
	var req []*request
	if b, err := ioutil.ReadAll(r.Body); err != nil {
		w.WriteHeader(500)
		return
	} else if err = json.Unmarshal(b, &req); err != nil {
		w.WriteHeader(400)
		return
	}
	for _, req := range req {
		if req.Method == "" || req.Path == "" {
			w.WriteHeader(400)
			return
		}
	}
	r.Body.Close()
	r.Body = nil
	c.serveBatch(w, r, req)
}

func (c *Handler) serveBatch(w http.ResponseWriter, r *http.Request, req []*request) {
	res := c.getResponses(r, req)
	if b, err := json.Marshal(res); err != nil {
		w.WriteHeader(500)
	} else {
		w.WriteHeader(200)
		w.Write(b)
	}
}

func (c *Handler) getResponses(r *http.Request, req []*request) []*response {

	res := make([]*response, len(req))

	var wg sync.WaitGroup
	wg.Add(len(req))
	var m sync.Mutex
	for i, req := range req {
		/* go */ func(req *request, i int) {
			batchResponse := c.getResponse(r, req)
			m.Lock()
			res[i] = batchResponse
			m.Unlock()
			wg.Done()
		}(req, i)
	}
	wg.Wait()
	return res
}

func (h *Handler) getResponse(r *http.Request, req *request) *response {
	batchReq, err := createHTTPRequest(r, req)
	if err != nil {
		panic(err)
	}
	var w rw
	h.NormalHandler.ServeHTTP(&w, batchReq)
	var body string
	if w.buf != nil && w.buf.Len() > 0 {
		body = w.buf.String()
	}
	return &response{
		Method:  req.Method,
		Path:    req.Path,
		Status:  w.status,
		Headers: w.header,
		Body:    body,
	}
}

type rw struct {
	header http.Header
	buf    *bytes.Buffer
	status int
}

func (w *rw) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *rw) Write(p []byte) (int, error) {
	if w.buf == nil {
		w.buf = new(bytes.Buffer)
	}
	w.buf.Write(p)
	return len(p), nil
}

func (w *rw) WriteHeader(status int) {
	w.status = status
}

func createHTTPRequest(mainRequest *http.Request, r *request) (*http.Request, error) {
	req, err := http.NewRequest(r.Method, r.Path, strings.NewReader(""))
	req.URL.Scheme = mainRequest.URL.Scheme
	req.URL.Host = mainRequest.URL.Host
	return req, err
}
