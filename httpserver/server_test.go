package httpserver_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/tscolari/gofakes/httpserver"
)

func makeRequest(t *testing.T, server *httpserver.Server, method, path string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, server.Addr()+path, bytes.NewBuffer([]byte{}))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	c := http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	return resp
}

func compareRequest(t *testing.T, actual *http.Request, method, path string) {
	t.Helper()

	if actual.Method != method {
		t.Fatalf("Expected request method to be %s, it was %s", method, actual.Method)
	}

	if actual.URL.Path != path {
		t.Fatalf("Expected request path to be %s, it was %s", path, actual.URL.Path)
	}
}

func compareResponse(t *testing.T, resp *http.Response, expectedStatus int, expectedBody []byte) {
	if resp.StatusCode != expectedStatus {
		t.Fatalf("Expected response code %d, got %d", resp.StatusCode, expectedStatus)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Unexpected err: %s", err)
	}
	if res := bytes.Compare(body, expectedBody); res != 0 {
		t.Fatalf("Expected body:\n%s\nGot:\n%s", expectedBody, body)
	}
}

func TestRegisterPayload(t *testing.T) {
	cases := []struct {
		Name         string
		Method, Path string
		StatusCode   int
		Payload      []byte
	}{
		{"Post200", "POST", "/hello", 200, []byte("hello post")},
		{"Get200", "GET", "/hello", 200, []byte("hello get")},
		{"Delete200", "DELETE", "/hello", 200, []byte("hello delete")},
		{"Post201", "POST", "/another", 201, []byte("another post")},
	}

	for _, tc := range cases {
		server := httpserver.New()
		server.Start()
		defer server.Stop()

		t.Run(tc.Name, func(t *testing.T) {
			server.RegisterPayload(tc.Method, tc.Path, tc.StatusCode, tc.Payload)
			resp := makeRequest(t, server, tc.Method, tc.Path)
			compareResponse(t, resp, tc.StatusCode, tc.Payload)
		})
	}
}

func TestRegisterHandler(t *testing.T) {
	cases := []struct {
		Name         string
		Method, Path string
		StatusCode   int
		Payload      []byte
	}{
		{"Post200", "POST", "/hello", 200, []byte("hello post")},
		{"Get200", "GET", "/hello", 200, []byte("hello get")},
		{"Delete200", "DELETE", "/hello", 200, []byte("hello delete")},
		{"Post201", "POST", "/another", 201, []byte("another post")},
	}

	for _, tc := range cases {
		server := httpserver.New()
		server.Start()
		defer server.Stop()

		t.Run(tc.Name, func(t *testing.T) {
			handler := func(rw http.ResponseWriter, r *http.Request) {
				rw.WriteHeader(tc.StatusCode)
				rw.Write(tc.Payload)
			}

			server.RegisterHandler(tc.Method, tc.Path, handler)
			resp := makeRequest(t, server, tc.Method, tc.Path)
			compareResponse(t, resp, tc.StatusCode, tc.Payload)
		})
	}
}

func TestNotFound(t *testing.T) {
	server := httpserver.New()
	server.Start()
	defer server.Stop()

	t.Run("PathNotFound", func(t *testing.T) {
		resp := makeRequest(t, server, "POST", "/hello")
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("Expected status code to be %d but it was %d", http.StatusNotFound, resp.StatusCode)
		}
	})

	t.Run("MethodNotFound", func(t *testing.T) {
		server.RegisterPayload("POST", "/hello", 200, []byte{})

		resp := makeRequest(t, server, "GET", "/hello")
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("Expected status code to be %d but it was %d", http.StatusMethodNotAllowed, resp.StatusCode)
		}
	})
}

func TestReset(t *testing.T) {
	server := httpserver.New()
	server.Start()
	defer server.Stop()
	server.RegisterPayload("POST", "/hello", http.StatusOK, []byte{})
	server.HandlerStub(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(200)
	})

	resp := makeRequest(t, server, "POST", "/hello")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code to be %d but it was %d", http.StatusOK, resp.StatusCode)
	}

	server.Reset()

	resp = makeRequest(t, server, "POST", "/hello")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status code to be %d but it was %d", http.StatusNotFound, resp.StatusCode)
	}

	if server.RequestCount() != 1 {
		t.Fatalf("Expected request count to be %d, it was %d", 1, server.RequestCount())
	}
}

func TestRequestCount(t *testing.T) {
	server := httpserver.New()
	server.Start()
	defer server.Stop()
	server.RegisterPayload("POST", "/hello", http.StatusOK, []byte{})

	makeRequest(t, server, "POST", "/hello")
	makeRequest(t, server, "POST", "/hello")
	makeRequest(t, server, "GET", "/404")

	if server.RequestCount() != 3 {
		t.Fatalf("Expected request count to be %d, it was %d", 3, server.RequestCount())
	}
}

func TestRequestNum(t *testing.T) {
	server := httpserver.New()
	server.Start()
	defer server.Stop()
	server.RegisterPayload("POST", "/hello", http.StatusOK, []byte{})
	server.RegisterPayload("GET", "/world", http.StatusOK, []byte{})

	makeRequest(t, server, "POST", "/hello")
	makeRequest(t, server, "GET", "/world")

	if server.RequestCount() != 2 {
		t.Fatalf("Expected request count to be %d, it was %d", 2, server.RequestCount())
	}

	compareRequest(t, server.RequestNum(0), "POST", "/hello")
	compareRequest(t, server.RequestNum(1), "GET", "/world")
}

func TestHandlerStub(t *testing.T) {
	server := httpserver.New()
	server.Start()
	defer server.Stop()

	var request *http.Request
	server.HandlerStub(func(rw http.ResponseWriter, r *http.Request) {
		request = r
		rw.Write([]byte("world"))
	})

	makeRequest(t, server, "POST", "/hello")
	compareRequest(t, request, "POST", "/hello")

	t.Run("Precedence", func(t *testing.T) {
		server.RegisterPayload("POST", "/hello", http.StatusOK, []byte("hello"))

		resp := makeRequest(t, server, "POST", "/hello")
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Unexpected err: %s", err)
		}
		if res := bytes.Compare(body, []byte("world")); res != 0 {
			t.Fatalf("Expected body:\n%s\nGot:\n%s", "world", body)
		}
	})
}
