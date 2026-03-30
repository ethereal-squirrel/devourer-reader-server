package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"github.com/devourer/server/internal/api"
	"github.com/devourer/server/internal/auth"
	"github.com/devourer/server/internal/config"
	"github.com/devourer/server/internal/db"
	"github.com/devourer/server/internal/db/queries"
	"github.com/devourer/server/internal/metadata"
	"github.com/devourer/server/internal/migrate"
	"github.com/devourer/server/internal/scanner"
	"github.com/devourer/server/internal/watcher"
)

func main() {
	godotenv.Load()

	root := &cobra.Command{
		Use:   "devourer-server",
		Short: "Devourer media server",
	}

	root.AddCommand(
		serveCmd(),
		createLibraryCmd(),
		scanLibraryCmd(),
		scanStatusCmd(),
		resetPasswordCmd(),
		migrateCalibreCmd(),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP server and file watcher",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()

			sqlDB, err := db.Open(cfg.DatabasePath)
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer sqlDB.Close()

			if err := db.Initialize(sqlDB, cfg.MigrationsDir); err != nil {
				return fmt.Errorf("initialize database: %w", err)
			}

			providers, err := metadata.LoadProviders(cfg.PluginsPath)
			if err != nil {
				log.Printf("[Main] Warning: could not load metadata providers: %v", err)
			}

			w, err := watcher.New(sqlDB, cfg.AssetsPath, cfg.PluginsPath, providers)
			if err != nil {
				return fmt.Errorf("create watcher: %w", err)
			}
			if err := w.Start(); err != nil {
				log.Printf("[Main] Watcher start error: %v", err)
			}
			defer w.Stop()

			r := api.NewServer(sqlDB, cfg, w)
			addr := fmt.Sprintf(":%s", cfg.Port)
			log.Printf("[Server] Devourer running on %s", addr)
			log.Printf("[Server] Database: %s", cfg.DatabasePath)
			log.Printf("[Server] Assets:   %s", cfg.AssetsPath)
			return r.Run(addr)
		},
	}
}

func createLibraryCmd() *cobra.Command {
	var name, path, libType, provider, apiKey string
	cmd := &cobra.Command{
		Use:   "create-library",
		Short: "Create a new library",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" || path == "" || libType == "" || provider == "" {
				return fmt.Errorf("--name, --path, --type and --provider are required")
			}
			cfg := config.Load()
			sqlDB, err := openDB(cfg)
			if err != nil {
				return err
			}
			defer sqlDB.Close()

			import_json := fmt.Sprintf(`{"provider":%q,"api_key":%q}`, provider, apiKey)
			lib, err := queries.CreateLibrary(sqlDB, name, path, libType,
				[]byte(import_json))
			if err != nil {
				return fmt.Errorf("create library: %w", err)
			}
			log.Printf("[Command] Created library %q (id=%d)", lib.Name, lib.ID)

			providers, _ := metadata.LoadProviders(cfg.PluginsPath)
			scanCfg := &scanner.Config{DB: sqlDB, PluginsPath: cfg.PluginsPath, Providers: providers}
			scanner.ScanLibrary(scanCfg, lib.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Library name")
	cmd.Flags().StringVar(&path, "path", "", "Library root path")
	cmd.Flags().StringVar(&libType, "type", "book", "Library type: book|manga")
	cmd.Flags().StringVar(&provider, "provider", "googlebooks", "Metadata provider key")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "Provider API key (optional)")
	return cmd
}

func scanLibraryCmd() *cobra.Command {
	var id int64
	cmd := &cobra.Command{
		Use:   "scan-library",
		Short: "Trigger a library scan",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == 0 {
				return fmt.Errorf("--id is required")
			}
			cfg := config.Load()
			sqlDB, err := openDB(cfg)
			if err != nil {
				return err
			}
			defer sqlDB.Close()

			providers, _ := metadata.LoadProviders(cfg.PluginsPath)
			scanCfg := &scanner.Config{DB: sqlDB, PluginsPath: cfg.PluginsPath, Providers: providers}
			result, err := scanner.ScanLibrary(scanCfg, id)
			if err != nil {
				return err
			}
			log.Printf("[Command] %s", result.Message)
			return nil
		},
	}
	cmd.Flags().Int64Var(&id, "id", 0, "Library ID")
	return cmd
}

func scanStatusCmd() *cobra.Command {
	var id int64
	cmd := &cobra.Command{
		Use:   "scan-status",
		Short: "Show scan status for a library",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == 0 {
				return fmt.Errorf("--id is required")
			}
			status := scanner.GetScanStatus(id)
			log.Printf("[Command] Scan status: %+v", status)
			return nil
		},
	}
	cmd.Flags().Int64Var(&id, "id", 0, "Library ID")
	return cmd
}

func resetPasswordCmd() *cobra.Command {
	var email, password string
	cmd := &cobra.Command{
		Use:   "reset-password",
		Short: "Reset a user's password",
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" || password == "" {
				return fmt.Errorf("--username and --password are required")
			}
			cfg := config.Load()
			sqlDB, err := openDB(cfg)
			if err != nil {
				return err
			}
			defer sqlDB.Close()

			resp, err := auth.ResetPassword(sqlDB, email, password)
			if err != nil {
				return err
			}
			log.Printf("[Command] %v", resp["message"])
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "username", "", "User email / username")
	cmd.Flags().StringVar(&password, "password", "", "New password")
	return cmd
}

func migrateCalibreCmd() *cobra.Command {
	var calibrePath, libraryName, provider string
	cmd := &cobra.Command{
		Use:   "migrate-calibre",
		Short: "Import a Calibre library",
		RunE: func(cmd *cobra.Command, args []string) error {
			if calibrePath == "" || libraryName == "" {
				return fmt.Errorf("--path and --name are required")
			}
			if provider == "" {
				provider = "googlebooks"
			}
			cfg := config.Load()
			sqlDB, err := openDB(cfg)
			if err != nil {
				return err
			}
			defer sqlDB.Close()

			return migrate.MigrateCalibre(sqlDB, calibrePath, libraryName, provider, 0)
		},
	}
	cmd.Flags().StringVar(&calibrePath, "path", "", "Path to Calibre library directory")
	cmd.Flags().StringVar(&libraryName, "name", "", "Name for the new Devourer library")
	cmd.Flags().StringVar(&provider, "provider", "googlebooks", "Metadata provider key")
	return cmd
}

func openDB(cfg *config.Config) (*sql.DB, error) {
	sqlDB, err := db.Open(cfg.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.Initialize(sqlDB, cfg.MigrationsDir); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("initialize database: %w", err)
	}
	return sqlDB, nil
}
