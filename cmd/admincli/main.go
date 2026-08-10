package main

import (
	"flag"
	"fmt"
	"os"

	"pixabros/internal/auth"
	"pixabros/internal/config"
	"pixabros/internal/db"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: admincli create-admin -username <u> -password <p>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "create-admin":
		createCmd := flag.NewFlagSet("create-admin", flag.ExitOnError)
		username := createCmd.String("username", "", "admin username")
		password := createCmd.String("password", "", "admin password")
		createCmd.Parse(os.Args[2:])

		if *username == "" || *password == "" {
			fmt.Fprintln(os.Stderr, "both -username and -password are required")
			os.Exit(1)
		}
		if err := createAdmin(*username, *password); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		fmt.Println("admin created:", *username)
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", os.Args[1])
		os.Exit(1)
	}
}

func createAdmin(username, password string) error {
	cfg := config.Load()

	conn, err := db.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer conn.Close()

	if err := db.Migrate(conn); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	_, err = conn.Exec(`INSERT INTO admins (username, password_hash) VALUES (?, ?);`, username, hash)
	if err != nil {
		return fmt.Errorf("insert admin: %w", err)
	}
	return nil
}
