package scriproxy

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newRequest(t *testing.T, method, url string) *http.Request {
	req, err := http.NewRequest(method, url, &bytes.Buffer{})

	if err != nil {
		t.Fatal(err)
	}

	return req
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
	t.Run("Host", func(t *testing.T) {
		testRequestRewriter(
			t,
			newRequest(t, "GET", "http://example.com/"),
			`req.host = "foo.com"`,
			[]string{""},
			func(t *testing.T, req *http.Request) {
				assert.Equal(t, "foo.com", req.Host)
			},
		)
	})

	t.Run("Query", func(t *testing.T) {
		t.Run("GetAndSet",
			func(t *testing.T) {
				testRequestRewriter(
					t,
					newRequest(t, "GET", "http://example.com/?foo=bar"),
					`req.url.query.set("baz", req.url.query.get("foo"))`,
					[]string{""},
					func(t *testing.T, req *http.Request) {
						assert.Equal(t, "bar", req.URL.Query().Get("baz"))
					},
				)
			},
		)

		t.Run("Del",
			func(t *testing.T) {
				testRequestRewriter(
					t,
					newRequest(t, "GET", "http://example.com/?foo=bar"),
					`req.url.query.del("foo")`,
					[]string{""},
					func(t *testing.T, req *http.Request) {
						assert.Equal(t, "", req.URL.Query().Get("foo"))
					},
				)
			},
		)
	})

	t.Run("Header", func(t *testing.T) {
		t.Run("GetAndSet", func(t *testing.T) {
			req := newRequest(t, "GET", "http://example.com/")
			req.Header.Set("foo", "bar")

			testRequestRewriter(
				t,
				req,
				`req.header.set("baz", req.header.get("foo"))`,
				[]string{""},
				func(t *testing.T, req *http.Request) {
					assert.Equal(t, "bar", req.Header.Get("baz"))
				},
			)
		})

		t.Run("Del", func(t *testing.T) {
			req := newRequest(t, "GET", "http://example.com/")
			req.Header.Set("foo", "bar")

			testRequestRewriter(
				t,
				req,
				`req.header.del("foo")`,
				[]string{""},
				func(t *testing.T, req *http.Request) {
					assert.Equal(t, "", req.Header.Get("foo"))
				},
			)
		})

		t.Run("Add", func(t *testing.T) {
			testRequestRewriter(
				t,
				newRequest(t, "GET", "http://example.com/"),
				`req.header.add("foo", "bar"); req.header.add("foo", "baz");`,
				[]string{""},
				func(t *testing.T, req *http.Request) {
					assert.Equal(t, 2, len(req.Header["Foo"]))
				},
			)
		})
	})

	t.Run("URL", func(t *testing.T) {
		testRequestRewriter(
			t,
			newRequest(t, "GET", "http://example.com/foo"),
			`
			req.url.scheme = req.url.scheme + "s"
			req.url.host = req.url.host + ":3000"
			req.url.path = req.url.path + "/bar"
			`,
			[]string{""},
			func(t *testing.T, req *http.Request) {
				assert.Equal(t, "https", req.URL.Scheme)
				assert.Equal(t, "example.com:3000", req.URL.Host)
				assert.Equal(t, "/foo/bar", req.URL.Path)
			},
		)
	})

	t.Run("Library", func(t *testing.T) {
		testRequestRewriter(
			t,
			newRequest(t, "GET", "http://example.com/"),
			`
			text := import("text")
			req.url.path = "/" + text.join(["foo", "bar"], "/")
			`,
			[]string{"text"},
			func(t *testing.T, req *http.Request) {
				assert.Equal(t, "/foo/bar", req.URL.Path)
			},
		)
	})
}
