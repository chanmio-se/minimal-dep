package cookie

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func testConfig() Session {
	return Session{Name: "session", Secure: true, MaxAge: 3600, SameSite: http.SameSiteLaxMode}
}

func TestSetWritesSecureAttributes(t *testing.T) {
	rec := httptest.NewRecorder()
	Set(rec, testConfig(), "token")
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %+v", cookies)
	}
	c := cookies[0]
	if c.Name != "session" || c.Value != "token" || !c.HttpOnly || !c.Secure ||
		c.Path != "/" || c.MaxAge != 3600 || c.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie = %+v", c)
	}
}

func TestClearExpiresCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	Clear(rec, testConfig())
	c := rec.Result().Cookies()[0]
	if c.MaxAge != -1 || !c.HttpOnly || !c.Secure {
		t.Fatalf("cleared cookie = %+v", c)
	}
}

func TestRead(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "token"})
	if value, ok := Read(req, "session"); !ok || value != "token" {
		t.Fatalf("Read() = %q, %v", value, ok)
	}
	if _, ok := Read(req, "missing"); ok {
		t.Fatal("Read() must fail for a missing cookie")
	}
}
