package sqlite

import (
	"net/url"
	"testing"
	"time"
)

func TestSQLiteDrivePathIsNotURIHost(t *testing.T) {
	for _, path := range []string{"C:/workspace/test database.sqlite", "D:/work/évidence.sqlite"} {
		parsed, e := url.Parse(sqliteDSN(path, time.Second))
		if e != nil {
			t.Fatal(e)
		}
		if parsed.Scheme != "file" || parsed.Host != "" || parsed.Opaque != (&url.URL{Path: path}).EscapedPath() || parsed.Path != "" {
			t.Fatalf("drive became URI authority: %s", parsed.String())
		}
		if parsed.Query().Get("_txlock") != "immediate" || len(parsed.Query()["_pragma"]) != 4 {
			t.Fatal("connection settings lost")
		}
	}
}
