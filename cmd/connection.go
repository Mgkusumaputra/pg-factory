package cmd

import (
	"fmt"
	"net/url"
	"strings"
)

// buildPostgresURL assembles a PostgreSQL connection URL with proper URL
// escaping for username, password, and database name.
func buildPostgresURL(user, pass string, port int, db string) string {
	u := &url.URL{
		Scheme: "postgresql",
		Host:   fmt.Sprintf("localhost:%d", port),
		Path:   "/" + strings.TrimPrefix(db, "/"),
		User:   url.UserPassword(user, pass),
	}
	return u.String()
}
