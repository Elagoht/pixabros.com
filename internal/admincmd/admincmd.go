// Package admincmd holds the operator commands that live in the server binary.
//
// They used to be a second binary, which meant a deployment was two artefacts
// and `./pixabros reset-password ...` silently started a server instead of
// resetting anything -- the flags were simply ignored. One binary, one place
// to look.
package admincmd

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"

	"pixabros/internal/auth"
	"pixabros/internal/config"
	"pixabros/internal/db"
)

const usage = `usage:
  pixabros                                        start the server
  pixabros create-admin   -username <u> -password <p>
  pixabros reset-password -username <u> -password <p>
  pixabros redraw-og-images                       redraw the generated devlog cards`

// Run executes an operator command.
//
// It reports whether the arguments were a command at all: with none, the
// caller goes on to start the server. An unrecognised argument is an error
// rather than a silent fall-through, because a mistyped command that quietly
// boots a server looks like it worked.
func Run(args []string) (handled bool) {
	if len(args) == 0 {
		return false
	}

	switch args[0] {
	case "create-admin":
		username, password := parseCredentials("create-admin", args[1:])
		if err := createAdmin(username, password); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		fmt.Println("admin created:", username)
	case "reset-password":
		username, password := parseCredentials("reset-password", args[1:])
		if err := resetPassword(username, password); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		fmt.Println("password reset:", username)
	case "redraw-og-images":
		if err := redrawOGImages(); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		fmt.Println(usage)
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", args[0])
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}
	return true
}

func parseCredentials(name string, args []string) (username, password string) {
	cmd := flag.NewFlagSet(name, flag.ExitOnError)
	u := cmd.String("username", "", "admin username")
	p := cmd.String("password", "", "admin password")
	cmd.Parse(args)

	if *u == "" || *p == "" {
		fmt.Fprintln(os.Stderr, "both -username and -password are required")
		os.Exit(1)
	}
	return *u, *p
}

// openDB opens the configured database and applies migrations, so every
// subcommand works against a database that may not exist yet.
func openDB() (*sql.DB, error) {
	cfg := config.Load()

	conn, err := db.Open(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Migrate(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return conn, nil
}

func hashNewPassword(password string) (string, error) {
	if err := auth.ValidatePassword(password); err != nil {
		return "", fmt.Errorf("invalid password: %w", err)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return hash, nil
}

func createAdmin(username, password string) error {
	conn, err := openDB()
	if err != nil {
		return err
	}
	defer conn.Close()

	hash, err := hashNewPassword(password)
	if err != nil {
		return err
	}

	if _, err := conn.Exec(
		`INSERT INTO admins (username, password_hash) VALUES (?, ?);`, username, hash,
	); err != nil {
		return fmt.Errorf("insert admin: %w", err)
	}
	return nil
}

// resetPassword is the out-of-band recovery path for a locked-out admin:
// the panel has no self-service reset because admins have no email address.
// Every existing session for the admin is dropped along with the old
// password, so a leaked session token cannot outlive the credential it was
// issued against.
func resetPassword(username, password string) error {
	conn, err := openDB()
	if err != nil {
		return err
	}
	defer conn.Close()

	admins := auth.NewAdminRepo(conn)
	admin, err := admins.FindByUsername(username)
	if errors.Is(err, auth.ErrAdminNotFound) {
		return fmt.Errorf("no admin named %q", username)
	}
	if err != nil {
		return fmt.Errorf("look up admin: %w", err)
	}

	hash, err := hashNewPassword(password)
	if err != nil {
		return err
	}

	if err := admins.UpdatePasswordHash(admin.ID, hash); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if err := auth.NewSessionStore(conn).DeleteAllForAdmin(admin.ID); err != nil {
		return fmt.Errorf("invalidate sessions: %w", err)
	}
	return nil
}
