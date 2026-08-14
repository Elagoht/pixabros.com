package gamearchive

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Scan reads a build that is already extracted on disk and produces the
// manifest an upload would have produced for it.
//
// It exists because the manifest is recorded at upload time, so every build
// that predates that feature has none -- and a game with no manifest shows no
// offline control at all, which is indistinguishable from a game whose control
// is working. Re-uploading each build by hand would move tens of megabytes to
// recover something already sitting on the server's own disk.
//
// It deliberately shares skip() and collector with Extract rather than
// reimplementing the rules, and a test asserts the two produce identical
// Builds for identical content: a backfilled manifest that disagreed with an
// uploaded one would advertise a download whose contents are not what the
// browser goes on to fetch.
func Scan(dir string) (Build, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return Build{}, fmt.Errorf("read build directory: %w", err)
	}
	if !info.IsDir() {
		return Build{}, fmt.Errorf("%s is not a directory", dir)
	}
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err != nil {
		return Build{}, fmt.Errorf("build at %s is missing index.html at its root", dir)
	}

	into := &collector{}
	err = filepath.WalkDir(dir, func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			// A whole debris directory is pruned here rather than having every
			// file inside it rejected one by one.
			if p != dir && skip(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		if skip(rel) {
			return nil
		}

		bytes, hash, err := hashFile(p)
		if err != nil {
			return err
		}
		into.add(rel, bytes, hash)
		return nil
	})
	if err != nil {
		return Build{}, err
	}
	return into.result(), nil
}

// hashFile streams a file through sha256 rather than reading it whole: a build
// carries single files of tens of megabytes, and the backfill walks every one
// of them at startup.
func hashFile(path string) (int64, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer f.Close()

	sum := sha256.New()
	n, err := io.Copy(sum, f)
	if err != nil {
		return 0, "", err
	}
	return n, hex.EncodeToString(sum.Sum(nil)), nil
}
