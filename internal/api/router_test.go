package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func request(router http.Handler, method, path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(method, path, nil))
	return recorder
}

func TestHealthContract(t *testing.T) {
	response := request(NewRouter(nil), http.MethodGet, "/api/v1/health")
	if response.Code != http.StatusOK || !strings.HasPrefix(response.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("health status/content type: %d %s", response.Code, response.Header())
	}
	var body struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Data.Status != "ok" {
		t.Fatalf("health body = %s, error = %v", response.Body.String(), err)
	}
}

func TestRoutingErrors(t *testing.T) {
	for _, test := range []struct {
		method, path, code string
		status             int
	}{
		{http.MethodGet, "/health", "not_found", 404},
		{http.MethodGet, "/api/v2/health", "not_found", 404},
		{http.MethodGet, "/api/v1/missing", "not_found", 404},
		{http.MethodGet, "/api/v1/health/", "not_found", 404},
		{http.MethodPost, "/api/v1/health", "method_not_allowed", 405},
		{http.MethodHead, "/api/v1/health", "method_not_allowed", 405},
	} {
		t.Run(test.method+test.path, func(t *testing.T) {
			response := request(NewRouter(nil), test.method, test.path)
			assertError(t, response, test.status, test.code)
			if test.status == 405 && response.Header().Get("Allow") != "GET" {
				t.Fatalf("Allow = %q", response.Header().Get("Allow"))
			}
		})
	}
}

func assertError(t *testing.T, response *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	var body ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != status || body.Error.Code != code || body.Error.Message == "" {
		t.Fatalf("error contract: status %d body %+v", response.Code, body)
	}
	if !strings.HasPrefix(response.Header().Get("Content-Type"), "application/json") {
		t.Fatal("error response is not JSON")
	}
}

func TestInternalFailuresAreRedactedAndServerRecovers(t *testing.T) {
	router := NewRouter(nil)
	router.GET("/test/error", func(c *gin.Context) { _ = c.Error(errors.New("private-test-detail")) })
	router.GET("/test/panic", func(c *gin.Context) { panic("private-test-detail") })
	for _, path := range []string{"/test/error", "/test/panic"} {
		response := request(router, http.MethodGet, path)
		assertError(t, response, 500, "internal_error")
		if strings.Contains(response.Body.String(), "private-test-detail") {
			t.Fatal("response disclosed internal failure detail")
		}
		if request(router, http.MethodGet, "/api/v1/health").Code != 200 {
			t.Fatal("health failed after internal failure")
		}
	}
}

func TestWrittenResponsesAreNotOverwritten(t *testing.T) {
	router := NewRouter(nil)
	for _, fail := range []string{"error", "panic"} {
		router.GET("/test/"+fail, func(c *gin.Context) {
			c.String(http.StatusAccepted, "committed")
			if fail == "panic" {
				panic("private-test-detail")
			}
			_ = c.Error(errors.New("private-test-detail"))
		})
		response := request(router, http.MethodGet, "/test/"+fail)
		if response.Code != http.StatusAccepted || response.Body.String() != "committed" {
			t.Fatalf("committed response was corrupted: %d %s", response.Code, response.Body.String())
		}
	}
}
