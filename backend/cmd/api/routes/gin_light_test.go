package routes

import (
	gin "github.com/numbergroup/gin-light"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestGinLightCompatibility guards the application handler type boundary.
func TestGinLightCompatibility(t *testing.T) {
	am := &authMiddleware{}
	r := gin.New()
	r.GET("/protected", am.Handle, func(c *gin.Context) { t.Error("unauthorized request reached handler") })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "missing authorization token") {
		t.Fatalf("body = %s", w.Body.String())
	}
}
