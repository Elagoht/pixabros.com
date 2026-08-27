package httpserver

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pixabros/internal/auth"
	"pixabros/internal/db"
	"pixabros/internal/games"
	"pixabros/internal/render"
	"pixabros/internal/site"
	"pixabros/internal/storage"
)

const (
	wantIndexRobots   = "index, follow, max-image-preview:large, max-snippet:-1, max-video-preview:-1"
	wantNoindexRobots = "noindex, nofollow, noarchive"
)

// directives splits a policy so a test can assert on one of them without
// depending on the order the others are written in.
func directives(policy string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(policy, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, value, _ := strings.Cut(part, " ")
		out[name] = strings.TrimSpace(value)
	}
	return out
}

func TestPolicyFor_PicksThePolicyForTheArea(t *testing.T) {
	for path, want := range map[string]string{
		"/":                          publicCSP,
		"/devlog/a-post":             publicCSP,
		"/assets/site-abc123.css":    publicCSP,
		"/media/og_generated/x.webp": publicCSP,
		"/api/admin/games":           apiCSP,
		"/api/contact":               apiCSP,
		"/play/dungrid-tactics/":     playCSP,
		adminUIPrefix:                panelCSP,
		adminUIPrefix + "devlog":     panelCSP,
	} {
		if got := policyFor(path); got != want {
			t.Errorf("policyFor(%q) picked the wrong policy:\n got %q", path, got)
		}
	}
}

func TestRobotsPolicyFor_PicksThePolicyForTheRoute(t *testing.T) {
	tests := map[string]string{
		"/":                        wantIndexRobots,
		"/games/dungrid-tactics":   wantIndexRobots,
		"/devlog/shipping-dungrid": wantIndexRobots,
		"/apiary":                  wantIndexRobots,
		"/playground":              wantIndexRobots,
		"/I-am-a-pixabrother/":     wantIndexRobots,
		"/assets2/site.css":        wantIndexRobots,
		"/media-library/cover":     wantIndexRobots,

		strings.TrimSuffix(adminUIPrefix, "/"): wantNoindexRobots,
		adminUIPrefix:                          wantNoindexRobots,
		adminUIPrefix + "games":                wantNoindexRobots,
		"/api":                                 wantNoindexRobots,
		"/api/admin/games":                     wantNoindexRobots,
		"/api/contact":                         wantNoindexRobots,
		"/play":                                wantNoindexRobots,
		"/play/dungrid-tactics/":               wantNoindexRobots,
		"/offline":                             wantNoindexRobots,
		"/contact/sent":                        wantNoindexRobots,
		"/sw.js":                               wantNoindexRobots,
		"/api/shell":                           wantNoindexRobots,

		"/assets":                    "",
		"/assets/site-abc123.css":    "",
		"/media":                     "",
		"/media/og_generated/x.webp": "",
		"/manifest.webmanifest":      "",
		"/robots.txt":                "",
		"/llms.txt":                  "",
		"/sitemap.xml":               "",
		"/rss.xml":                   "",
	}

	for path, want := range tests {
		t.Run(path, func(t *testing.T) {
			if got := robotsPolicyFor(path); got != want {
				t.Errorf("robotsPolicyFor(%q) = %q, want %q", path, got, want)
			}
		})
	}
}

// The one third party the site frames is the video player, and naming it is
// the whole reason frame-src is not just 'self'.
func TestPublicCSP_FramesOnlyTheVideoPlayerAndItself(t *testing.T) {
	frameSrc := directives(publicCSP)["frame-src"]

	sources := strings.Fields(frameSrc)
	want := map[string]bool{"'self'": true, "https://www.youtube-nocookie.com": true}
	if len(sources) != len(want) {
		t.Fatalf("frame-src = %q, want exactly %d sources", frameSrc, len(want))
	}
	for _, source := range sources {
		if !want[source] {
			t.Errorf("frame-src allows %q, which is neither this site nor the video player", source)
		}
	}
}

// manifest-src has no fallback of its own beyond default-src, so a policy that
// does not name it denies the manifest -- and a site whose manifest is blocked
// is a site that cannot be installed, silently.
func TestPublicCSP_AllowsItsOwnManifest(t *testing.T) {
	if got := directives(publicCSP)["manifest-src"]; got != "'self'" {
		t.Errorf("public manifest-src = %q, want 'self'", got)
	}
}

// worker-src falls back to child-src and then to default-src, which is 'none'.
// A policy that does not name it refuses to start the service worker, and the
// site loses offline support without a single visible error.
func TestPublicCSP_AllowsItsOwnServiceWorker(t *testing.T) {
	if got := directives(publicCSP)["worker-src"]; got != "'self'" {
		t.Errorf("public worker-src = %q, want 'self'", got)
	}
}

// Anything the policy forgets to name falls through to default-src, so
// default-src 'none' is what makes the rest of the policy a whitelist rather
// than a suggestion.
func TestPolicies_DenyByDefault(t *testing.T) {
	for name, policy := range map[string]string{
		"public": publicCSP,
		"panel":  panelCSP,
		"api":    apiCSP,
	} {
		if got := directives(policy)["default-src"]; got != "'none'" {
			t.Errorf("%s default-src = %q, want 'none'", name, got)
		}
	}
}

// A page that carries no inline script must not permit one: 'unsafe-inline'
// and 'unsafe-eval' in script-src would give away the entire defence.
func TestPolicies_NeverAllowInlineOrEvaluatedScript(t *testing.T) {
	for name, policy := range map[string]string{
		"public": publicCSP,
		"panel":  panelCSP,
		"api":    apiCSP,
		"play":   playCSP,
	} {
		scriptSrc := directives(policy)["script-src"]
		for _, banned := range []string{"'unsafe-inline'", "'unsafe-eval'"} {
			if strings.Contains(scriptSrc, banned) {
				t.Errorf("%s script-src = %q, which allows %s", name, scriptSrc, banned)
			}
		}
	}
}

// The public site is never framed, and the panel least of all: clickjacking a
// screen that holds a session is the case that matters.
func TestPolicies_RefuseToBeFramed(t *testing.T) {
	for name, policy := range map[string]string{
		"public": publicCSP,
		"panel":  panelCSP,
		"api":    apiCSP,
	} {
		if got := directives(policy)["frame-ancestors"]; got != "'none'" {
			t.Errorf("%s frame-ancestors = %q, want 'none'", name, got)
		}
	}
	// A game build is the exception: the console frames it from this origin.
	if got := directives(playCSP)["frame-ancestors"]; got != "'self'" {
		t.Errorf("play frame-ancestors = %q, want 'self' so the console can frame a build", got)
	}
}

// An uploaded build runs inline script and compiles WebAssembly, so a content
// policy here would break games. Restricting who may frame it is the part
// worth keeping, and it must not quietly grow into more.
func TestPlayCSP_OnlyRestrictsFraming(t *testing.T) {
	if got := directives(playCSP); len(got) != 1 {
		t.Errorf("play policy = %q, want frame-ancestors alone", playCSP)
	}
}

// The policies above are only worth anything if they actually reach a
// response, so this goes through the router the server really builds.
func TestNew_SendsThePolicyOnEveryResponse(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	playDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(playDir, "a-game"), 0o755); err != nil {
		t.Fatalf("make play dir: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(playDir, "a-game", "index.html"), []byte("<h1>game</h1>"), 0o644,
	); err != nil {
		t.Fatalf("write game index: %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := render.NewStore(conn, files)
	if _, err := store.RenderAndPersist("index.html", func(string) ([]byte, []string, error) {
		return []byte("<h1>public</h1>"), []string{"public"}, nil
	}); err != nil {
		t.Fatalf("RenderAndPersist() error = %v", err)
	}

	srv := httptest.NewServer(New(Dependencies{
		Admins:    auth.NewAdminRepo(conn),
		Sessions:  auth.NewSessionStore(conn),
		Store:     store,
		Files:     files,
		DB:        conn,
		Games:     games.NewRepo(conn),
		PlayDir:   playDir,
		AssetsDir: t.TempDir(),
	}))
	defer srv.Close()

	for path, want := range map[string]string{
		"/":                   publicCSP,
		"/api/admin/whoami":   apiCSP,
		"/play/a-game/":       playCSP,
		"/I-am-a-pixabro/":    panelCSP,
		"/nothing-lives-here": publicCSP,
		"/api/does-not-exist": apiCSP,
	} {
		resp, err := srv.Client().Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()

		if got := resp.Header.Get("Content-Security-Policy"); got != want {
			t.Errorf("GET %s policy =\n %q\nwant\n %q", path, got, want)
		}
		if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("GET %s X-Content-Type-Options = %q, want nosniff", path, got)
		}
	}
}

func TestNew_SendsRobotsPolicyForRoutesAndLeavesResourcesUnmarked(t *testing.T) {
	srv := newRobotsTestServer(t)

	for path, want := range map[string]string{
		"/":                     wantIndexRobots,
		"/offline":              wantNoindexRobots,
		"/contact/sent":         wantNoindexRobots,
		"/api/admin/whoami":     wantNoindexRobots,
		"/play/a-game/":         wantNoindexRobots,
		adminUIPrefix:           wantNoindexRobots,
		"/sw.js":                wantNoindexRobots,
		"/api/shell":            wantNoindexRobots,
		"/assets/site.js":       "",
		"/media/cover.webp":     "",
		"/manifest.webmanifest": "",
		"/robots.txt":           "",
		"/llms.txt":             "",
		"/sitemap.xml":          "",
		"/rss.xml":              "",
	} {
		resp, err := srv.Client().Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()

		if got := resp.Header.Get("X-Robots-Tag"); got != want {
			t.Errorf("GET %s X-Robots-Tag = %q, want %q", path, got, want)
		}
	}
}

func TestNew_Noindexes404Responses(t *testing.T) {
	srv := newRobotsTestServer(t)

	for _, path := range []string{"/nothing-lives-here", "/assets/missing.js"} {
		resp, err := srv.Client().Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s status = %d, want %d", path, resp.StatusCode, http.StatusNotFound)
		}
		if got := resp.Header.Get("X-Robots-Tag"); got != wantNoindexRobots {
			t.Errorf("GET %s X-Robots-Tag = %q, want %q", path, got, wantNoindexRobots)
		}
	}
}

func newRobotsTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	playDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(playDir, "a-game"), 0o755); err != nil {
		t.Fatalf("make play dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(playDir, "a-game", "index.html"), []byte("game"), 0o644); err != nil {
		t.Fatalf("write game index: %v", err)
	}

	assetsDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(assetsDir, "site.js"), []byte("asset"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}
	mediaDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(mediaDir, "cover.webp"), []byte("media"), 0o644); err != nil {
		t.Fatalf("write media: %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := render.NewStore(conn, files)
	for key, body := range map[string]string{
		"index.html":   "public",
		"offline":      "offline",
		"contact/sent": "sent",
		"robots.txt":   "robots",
		"llms.txt":     "llms",
		"sitemap.xml":  "sitemap",
		"rss.xml":      "rss",
	} {
		if _, err := store.RenderAndPersist(key, func(string) ([]byte, []string, error) {
			return []byte(body), nil, nil
		}); err != nil {
			t.Fatalf("RenderAndPersist(%q) error = %v", key, err)
		}
	}

	plain := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(New(Dependencies{
		Admins:        auth.NewAdminRepo(conn),
		Sessions:      auth.NewSessionStore(conn),
		Store:         store,
		Files:         files,
		DB:            conn,
		Games:         games.NewRepo(conn),
		PlayDir:       playDir,
		AssetsDir:     assetsDir,
		MediaDir:      mediaDir,
		Manifest:      plain,
		ServiceWorker: plain,
		Shell:         plain,
		NotFoundBody:  []byte("<h1>Not found</h1>"),
	}))
	t.Cleanup(srv.Close)
	return srv
}

var (
	errHijack = errors.New("hijack reached underlying writer")
	errPush   = errors.New("push reached underlying writer")
	errRead   = errors.New("read failed")
)

// statusRecordingWriter models net/http's commitment rules closely enough to
// expose middleware mistakes that httptest.ResponseRecorder cannot: 1xx
// responses (except 101) are informational and may precede a final response.
type statusRecordingWriter struct {
	header   http.Header
	statuses []int
	final    int
	body     strings.Builder
}

func (w *statusRecordingWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *statusRecordingWriter) WriteHeader(status int) {
	if w.final != 0 {
		return
	}
	if status < 100 || status > 999 {
		panic("invalid status")
	}
	w.statuses = append(w.statuses, status)
	if status < 100 || status > 199 || status == http.StatusSwitchingProtocols {
		w.final = status
	}
}

func (w *statusRecordingWriter) Write(body []byte) (int, error) {
	if w.final == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.body.Write(body)
}

type statusReaderFromWriter struct{ *statusRecordingWriter }

func (w *statusReaderFromWriter) ReadFrom(src io.Reader) (int64, error) {
	return io.Copy(struct{ io.Writer }{w.statusRecordingWriter}, src)
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errRead }

func TestWithSecurityHeaders_InformationalStatusCanBeFollowedBy404(t *testing.T) {
	underlying := &statusRecordingWriter{}
	handler := withSecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusEarlyHints)
		w.WriteHeader(http.StatusNotFound)
	}))

	handler.ServeHTTP(underlying, httptest.NewRequest(http.MethodGet, "/missing", nil))

	wantStatuses := []int{http.StatusEarlyHints, http.StatusNotFound}
	if got := underlying.statuses; len(got) != len(wantStatuses) || got[0] != wantStatuses[0] || got[1] != wantStatuses[1] {
		t.Errorf("statuses = %v, want %v", got, wantStatuses)
	}
	if got := underlying.Header().Get("X-Robots-Tag"); got != wantNoindexRobots {
		t.Errorf("X-Robots-Tag = %q, want %q", got, wantNoindexRobots)
	}
}

func TestWithSecurityHeaders_InvalidStatusDoesNotCommitWrapper(t *testing.T) {
	underlying := &statusRecordingWriter{}
	handler := withSecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		panicked := false
		func() {
			defer func() { panicked = recover() != nil }()
			w.WriteHeader(99)
		}()
		if !panicked {
			t.Error("WriteHeader(99) did not panic")
		}
		w.WriteHeader(http.StatusNotFound)
	}))

	handler.ServeHTTP(underlying, httptest.NewRequest(http.MethodGet, "/missing", nil))

	if underlying.final != http.StatusNotFound {
		t.Errorf("final status = %d, want %d", underlying.final, http.StatusNotFound)
	}
	if got := underlying.Header().Get("X-Robots-Tag"); got != wantNoindexRobots {
		t.Errorf("X-Robots-Tag = %q, want %q", got, wantNoindexRobots)
	}
}

func TestWithSecurityHeaders_EmptyOrFailedReadFromDoesNotCommitWrapper(t *testing.T) {
	tests := []struct {
		name    string
		reader  io.Reader
		wantErr error
	}{
		{name: "empty EOF", reader: strings.NewReader("")},
		{name: "error before bytes", reader: failingReader{}, wantErr: errRead},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			underlying := &statusReaderFromWriter{statusRecordingWriter: &statusRecordingWriter{}}
			handler := withSecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				readerFrom := w.(io.ReaderFrom)
				n, err := readerFrom.ReadFrom(tc.reader)
				if n != 0 {
					t.Errorf("ReadFrom() bytes = %d, want 0", n)
				}
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("ReadFrom() error = %v, want %v", err, tc.wantErr)
				}
				w.WriteHeader(http.StatusNotFound)
			}))

			handler.ServeHTTP(underlying, httptest.NewRequest(http.MethodGet, "/missing", nil))

			if underlying.final != http.StatusNotFound {
				t.Errorf("final status = %d, want %d", underlying.final, http.StatusNotFound)
			}
			if got := underlying.Header().Get("X-Robots-Tag"); got != wantNoindexRobots {
				t.Errorf("X-Robots-Tag = %q, want %q", got, wantNoindexRobots)
			}
		})
	}
}

type optionalResponseWriter struct {
	header  http.Header
	body    strings.Builder
	flushed bool
}

func (w *optionalResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *optionalResponseWriter) Write(body []byte) (int, error) {
	return w.body.Write(body)
}

func (w *optionalResponseWriter) WriteHeader(int) {}

func (w *optionalResponseWriter) Flush() { w.flushed = true }

func (w *optionalResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, errHijack
}

func (w *optionalResponseWriter) ReadFrom(src io.Reader) (int64, error) {
	return io.Copy(&w.body, src)
}

func (w *optionalResponseWriter) Push(string, *http.PushOptions) error { return errPush }

const (
	testHasFlusher = 1 << iota
	testHasHijacker
	testHasReaderFrom
	testHasPusher
)

func optionalInterfaceMask(w http.ResponseWriter) int {
	mask := 0
	if _, ok := w.(http.Flusher); ok {
		mask |= testHasFlusher
	}
	if _, ok := w.(http.Hijacker); ok {
		mask |= testHasHijacker
	}
	if _, ok := w.(io.ReaderFrom); ok {
		mask |= testHasReaderFrom
	}
	if _, ok := w.(http.Pusher); ok {
		mask |= testHasPusher
	}
	return mask
}

func TestWrapRobotsResponseWriter_PreservesAllOptionalInterfaceMasks(t *testing.T) {
	base := &plainResponseWriter{}
	all := &optionalResponseWriter{}
	tests := []struct {
		name       string
		mask       int
		underlying http.ResponseWriter
	}{
		{name: "none", underlying: base},
		{name: "flusher", mask: testHasFlusher, underlying: struct {
			http.ResponseWriter
			http.Flusher
		}{base, all}},
		{name: "hijacker", mask: testHasHijacker, underlying: struct {
			http.ResponseWriter
			http.Hijacker
		}{base, all}},
		{name: "reader-from", mask: testHasReaderFrom, underlying: struct {
			http.ResponseWriter
			io.ReaderFrom
		}{base, all}},
		{name: "pusher", mask: testHasPusher, underlying: struct {
			http.ResponseWriter
			http.Pusher
		}{base, all}},
		{name: "flusher-hijacker", mask: testHasFlusher | testHasHijacker, underlying: struct {
			http.ResponseWriter
			http.Flusher
			http.Hijacker
		}{base, all, all}},
		{name: "flusher-reader-from", mask: testHasFlusher | testHasReaderFrom, underlying: struct {
			http.ResponseWriter
			http.Flusher
			io.ReaderFrom
		}{base, all, all}},
		{name: "flusher-pusher", mask: testHasFlusher | testHasPusher, underlying: struct {
			http.ResponseWriter
			http.Flusher
			http.Pusher
		}{base, all, all}},
		{name: "hijacker-reader-from", mask: testHasHijacker | testHasReaderFrom, underlying: struct {
			http.ResponseWriter
			http.Hijacker
			io.ReaderFrom
		}{base, all, all}},
		{name: "hijacker-pusher", mask: testHasHijacker | testHasPusher, underlying: struct {
			http.ResponseWriter
			http.Hijacker
			http.Pusher
		}{base, all, all}},
		{name: "reader-from-pusher", mask: testHasReaderFrom | testHasPusher, underlying: struct {
			http.ResponseWriter
			io.ReaderFrom
			http.Pusher
		}{base, all, all}},
		{name: "flusher-hijacker-reader-from", mask: testHasFlusher | testHasHijacker | testHasReaderFrom, underlying: struct {
			http.ResponseWriter
			http.Flusher
			http.Hijacker
			io.ReaderFrom
		}{base, all, all, all}},
		{name: "flusher-hijacker-pusher", mask: testHasFlusher | testHasHijacker | testHasPusher, underlying: struct {
			http.ResponseWriter
			http.Flusher
			http.Hijacker
			http.Pusher
		}{base, all, all, all}},
		{name: "flusher-reader-from-pusher", mask: testHasFlusher | testHasReaderFrom | testHasPusher, underlying: struct {
			http.ResponseWriter
			http.Flusher
			io.ReaderFrom
			http.Pusher
		}{base, all, all, all}},
		{name: "hijacker-reader-from-pusher", mask: testHasHijacker | testHasReaderFrom | testHasPusher, underlying: struct {
			http.ResponseWriter
			http.Hijacker
			io.ReaderFrom
			http.Pusher
		}{base, all, all, all}},
		{name: "all", mask: testHasFlusher | testHasHijacker | testHasReaderFrom | testHasPusher, underlying: all},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := optionalInterfaceMask(tc.underlying); got != tc.mask {
				t.Fatalf("test writer mask = %04b, want %04b", got, tc.mask)
			}
			if got := optionalInterfaceMask(wrapRobotsResponseWriter(tc.underlying)); got != tc.mask {
				t.Errorf("wrapped writer mask = %04b, want %04b", got, tc.mask)
			}
		})
	}
}

func TestWithSecurityHeaders_PreservesOptionalResponseWriterBehavior(t *testing.T) {
	underlying := &optionalResponseWriter{}
	handler := withSecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("wrapped writer lost http.Flusher")
		} else {
			flusher.Flush()
		}

		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Error("wrapped writer lost http.Hijacker")
		} else if _, _, err := hijacker.Hijack(); !errors.Is(err, errHijack) {
			t.Errorf("Hijack() error = %v, want underlying error", err)
		}

		readerFrom, ok := w.(io.ReaderFrom)
		if !ok {
			t.Error("wrapped writer lost io.ReaderFrom")
		} else if _, err := readerFrom.ReadFrom(strings.NewReader("streamed")); err != nil {
			t.Errorf("ReadFrom() error = %v", err)
		}

		pusher, ok := w.(http.Pusher)
		if !ok {
			t.Error("wrapped writer lost http.Pusher")
		} else if err := pusher.Push("/asset.js", nil); !errors.Is(err, errPush) {
			t.Errorf("Push() error = %v, want underlying error", err)
		}
	}))

	handler.ServeHTTP(underlying, httptest.NewRequest(http.MethodGet, "/", nil))

	if !underlying.flushed {
		t.Error("Flush was not forwarded")
	}
	if got := underlying.body.String(); got != "streamed" {
		t.Errorf("streamed body = %q, want %q", got, "streamed")
	}
}

type plainResponseWriter struct{ header http.Header }

func (w *plainResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (*plainResponseWriter) Write(body []byte) (int, error) { return len(body), nil }

func (*plainResponseWriter) WriteHeader(int) {}

func TestWithSecurityHeaders_DoesNotInventOptionalResponseWriterBehavior(t *testing.T) {
	underlying := &plainResponseWriter{}
	handler := withSecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, ok := w.(http.Flusher); ok {
			t.Error("wrapped writer invented http.Flusher")
		}
		if _, ok := w.(http.Hijacker); ok {
			t.Error("wrapped writer invented http.Hijacker")
		}
		if _, ok := w.(io.ReaderFrom); ok {
			t.Error("wrapped writer invented io.ReaderFrom")
		}
		if _, ok := w.(http.Pusher); ok {
			t.Error("wrapped writer invented http.Pusher")
		}
	}))

	handler.ServeHTTP(underlying, httptest.NewRequest(http.MethodGet, "/", nil))
}

// data: and blob: are not in any img-src, and they should not creep in: the
// panel previews an upload by fetching it back from the server after it lands,
// not by rendering the local file. If that ever changes this test is the place
// the change gets noticed.
func TestPolicies_LoadImagesOnlyFromThisSite(t *testing.T) {
	for name, policy := range map[string]string{
		"public": publicCSP,
		"panel":  panelCSP,
	} {
		if got := directives(policy)["img-src"]; got != "'self'" {
			t.Errorf("%s img-src = %q, want 'self'", name, got)
		}
	}
}

// Every path robots.txt disallows is answered with noindex here. Both policies
// are written from site.CrawlerExclusions, so neither can name a path the
// other has never heard of -- and this test is what catches the list itself
// being reshaped into something the matcher no longer honours.
func TestRobotsPolicyFor_CoversEveryCrawlerExclusion(t *testing.T) {
	for _, excluded := range site.CrawlerExclusions() {
		paths := []string{strings.TrimSuffix(excluded, "/")}
		if strings.HasSuffix(excluded, "/") {
			paths = append(paths, excluded+"probe")
		}
		for _, path := range paths {
			if got := robotsPolicyFor(path); got != wantNoindexRobots {
				t.Errorf("robotsPolicyFor(%q) = %q, want %q", path, got, wantNoindexRobots)
			}
		}
	}
}
