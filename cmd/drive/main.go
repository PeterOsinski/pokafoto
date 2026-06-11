package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/drive/drive/internal/config"
	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/server"
	"github.com/drive/drive/internal/store"
	"github.com/google/uuid"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "admin":
			adminCmd(os.Args[2:])
			return
		case "import":
			if err := runImport(os.Args[2:]); err != nil {
				fmt.Fprintf(os.Stderr, "import error: %v\n", err)
				os.Exit(1)
			}
			return
		case "help", "--help", "-h":
			printUsage()
			return
		}
	}

	cfg := config.Load()
	if err := run(cfg); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config) error {
	if err := os.MkdirAll(filepath.Dir(cfg.Database.Path), 0755); err != nil {
		return fmt.Errorf("create database directory: %w", err)
	}

	db, err := store.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := db.RunMigrations(); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	userStore := store.NewUserStore(db)
	count, err := userStore.Count()
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}

	if count == 0 {
		slog.Warn("no admin user found. Create one with: drive admin create")
	}

	if cfg.Auth.JWTSecret == "" {
		cfg.Auth.JWTSecret = generateJWTSecret()
		slog.Info("auto-generated JWT secret")
	}

	if err := os.MkdirAll(cfg.OriginalsDir(), 0755); err != nil {
		return fmt.Errorf("create originals dir: %w", err)
	}
	if err := os.MkdirAll(cfg.ThumbnailsDir(), 0755); err != nil {
		return fmt.Errorf("create thumbnails dir: %w", err)
	}

	srv := server.New(cfg, db)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-shutdown
		slog.Info("shutting down...")
		srv.Shutdown()
		os.Exit(0)
	}()

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	slog.Info("starting server", "addr", addr)
	return srv.Start(addr)
}

func adminCmd(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: drive admin <command>")
		fmt.Println("Commands:")
		fmt.Println("  create    Create the first admin user")
		return
	}

	switch args[0] {
	case "create":
		createAdmin()
	default:
		fmt.Printf("Unknown admin command: %s\n", args[0])
		os.Exit(1)
	}
}

func createAdmin() {
	cfg := config.Load()

	if err := os.MkdirAll(filepath.Dir(cfg.Database.Path), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating data directory: %v\n", err)
		os.Exit(1)
	}

	db, err := store.Open(cfg.Database.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.RunMigrations(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running migrations: %v\n", err)
		os.Exit(1)
	}

	userStore := store.NewUserStore(db)
	count, err := userStore.Count()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error counting users: %v\n", err)
		os.Exit(1)
	}

	if count > 0 {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("There are existing users. Create another admin? (y/N): ")
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)
		if response != "y" && response != "Y" {
			fmt.Println("Aborted.")
			return
		}
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)
	if username == "" {
		fmt.Fprintln(os.Stderr, "Username is required")
		os.Exit(1)
	}

	fmt.Print("Password: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)
	if password == "" {
		fmt.Fprintln(os.Stderr, "Password is required")
		os.Exit(1)
	}

	if len(password) < 8 {
		fmt.Fprintln(os.Stderr, "Password must be at least 8 characters")
		os.Exit(1)
	}

	var displayName *string
	fmt.Print("Display Name (optional): ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name != "" {
		displayName = &name
	}

	user, err := userStore.Create(username, password, model.RoleAdmin, displayName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating admin user: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Admin user created successfully: %s (id: %s)\n", user.Username, user.ID)
}

func printUsage() {
	fmt.Println("Drive - Self-hosted photo & file backup")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  drive              Start the server")
	fmt.Println("  drive admin create Create the first admin user")
	fmt.Println("  drive import       Bulk import files from a local directory")
	fmt.Println("  drive help         Show this help")
}

func generateJWTSecret() string {
	return uuid.New().String() + uuid.New().String()
}
