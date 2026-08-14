package httpserver

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pixabros/internal/auth"
	"pixabros/internal/db"
	"pixabros/internal/games"
	"pixabros/internal/render"
	"pixabros/internal/storage"
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
