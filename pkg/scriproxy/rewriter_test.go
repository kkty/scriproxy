package scriproxy

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newMockRequest(t *testing.T) *http.Request {
	mockRequest, err := http.NewRequest(
		"GET",
		"http://example.com:3000/foo?bar=baz",
		&bytes.Buffer{},
	)

	mockRequest.Header.Set("bar", "baz")

	if err != nil {
		t.Fatal(err)
	}

	return mockRequest
}

func testRequestRewriter(
	t *testing.T,
	req *http.Request,
	script string,
	libraries []string,
	assertFn func(*testing.T, *http.Request),
) {
	requestRewriter, err := NewRequestRewriter([]byte(script), libraries)

	if err != nil {
		t.Fatal(err)
	}

	if err := requestRewriter.Rewrite(req); err != nil {
		t.Fatal(err)
	}

	assertFn(t, req)
}

func TestRequestRewriter(t *testing.T) {
	t.Run("RewriteHost", func(t *testing.T) {
		testRequestRewriter(
			t,
			newMockRequest(t),
			`req.host = "foo.com"`,
			[]string{""},
			func(t *testing.T, req *http.Request) {
				assert.Equal(t, "foo.com", req.Host)
			},
		)
	})

	t.Run("RewriteQuery", func(t *testing.T) {
		testRequestRewriter(
			t,
			newMockRequest(t),
			`req.url.query.set("qux", req.url.query.get("bar"))`,
			[]string{""},
			func(t *testing.T, req *http.Request) {
				assert.Equal(t, "baz", req.URL.Query().Get("qux"))
			},
		)
	})

	t.Run("RewriteHeader", func(t *testing.T) {
		testRequestRewriter(
			t,
			newMockRequest(t),
			`req.header.set("qux", req.header.get("bar"))`,
			[]string{""},
			func(t *testing.T, req *http.Request) {
				assert.Equal(t, "baz", req.Header.Get("qux"))
			},
		)
	})

	t.Run("RewriteURL", func(t *testing.T) {
		testRequestRewriter(
			t,
			newMockRequest(t),
			`
			req.url.scheme = req.url.scheme + "s"
			req.url.host = "foo.com"
			req.url.path = "bar"
			`,
			[]string{""},
			func(t *testing.T, req *http.Request) {
				assert.Equal(t, "https", req.URL.Scheme)
				assert.Equal(t, "foo.com", req.URL.Host)
				assert.Equal(t, "bar", req.URL.Path)
			},
		)
	})

	t.Run("UseLibrary", func(t *testing.T) {
		testRequestRewriter(
			t,
			newMockRequest(t),
			`
			text := import("text")
			req.url.path = text.join(["foo", "bar"], "/")
			`,
			[]string{"text"},
			func(t *testing.T, req *http.Request) {
				assert.Equal(t, "foo/bar", req.URL.Path)
			},
		)
	})
}
