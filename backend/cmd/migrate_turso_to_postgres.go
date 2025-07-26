package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"                                // Postgres driver
	_ "github.com/tursodatabase/libsql-client-go/libsql" // Turso driver
)

/*

Turso Datbase Schema:

CREATE TABLE "__drizzle_migrations" (
                        id SERIAL PRIMARY KEY,
                        hash text NOT NULL,
                        created_at numeric
                );
CREATE TABLE `about_page` (
        `id` integer PRIMARY KEY AUTOINCREMENT NOT NULL,
        `title` text,
        `content` text,
        `profile_image` text,
        `meta_description` text,
        `last_updated` text
);
CREATE TABLE `article_tags` (
        `article_id` integer NOT NULL,
        `tag_id` integer NOT NULL,
        FOREIGN KEY (`article_id`) REFERENCES `articles`(`id`) ON UPDATE no action ON DELETE no action,
        FOREIGN KEY (`tag_id`) REFERENCES `tags`(`tag_id`) ON UPDATE no action ON DELETE no action
);
CREATE UNIQUE INDEX `article_tags_article_id_tag_id_unique` ON `article_tags` (`article_id`,`tag_id`);
CREATE TABLE `articles` (
        `id` integer PRIMARY KEY AUTOINCREMENT NOT NULL,
        `image` text,
        `slug` text NOT NULL,
        `title` text NOT NULL,
        `content` text NOT NULL,
        `author` integer NOT NULL,
        `created_at` integer DEFAULT (unixepoch()) NOT NULL,
        `updated_at` integer DEFAULT (unixepoch()) NOT NULL,
        `is_draft` integer DEFAULT false NOT NULL,
        `embedding` F32_BLOB(1536),
        `image_generation_request_id` text, `published_at` integer, `chat_history` text,
        FOREIGN KEY (`author`) REFERENCES `users`(`id`) ON UPDATE no action ON DELETE no action
);
CREATE UNIQUE INDEX `articles_slug_unique` ON `articles` (`slug`);
CREATE TABLE `contact_page` (
        `id` integer PRIMARY KEY AUTOINCREMENT NOT NULL,
        `title` text,
        `content` text,
        `email_address` text,
        `social_links` text,
        `meta_description` text,
        `last_updated` text
);
CREATE TABLE goose_db_version (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                version_id INTEGER NOT NULL,
                is_applied INTEGER NOT NULL,
                tstamp TIMESTAMP DEFAULT (datetime('now'))
        );
CREATE TABLE `image_generation` (
        `id` integer PRIMARY KEY AUTOINCREMENT NOT NULL,
        `prompt` text,
        `provider` text,
        `model` text,
        `request_id` text,
        `output_url` text,
        `storage_key` text,
        `created_at` integer DEFAULT (unixepoch()) NOT NULL
);
CREATE TABLE `projects` (
        `id` integer PRIMARY KEY AUTOINCREMENT NOT NULL,
        `title` text NOT NULL,
        `description` text NOT NULL,
        `url` text NOT NULL,
        `image` text
);
CREATE TABLE `tags` (
        `tag_id` integer PRIMARY KEY AUTOINCREMENT NOT NULL,
        `tag_name` text
);
CREATE UNIQUE INDEX `tags_tag_name_unique` ON `tags` (`tag_name`);
CREATE TABLE `users` (
        `id` integer PRIMARY KEY AUTOINCREMENT NOT NULL,
        `name` text NOT NULL,
        `email` text NOT NULL,
        `passwordHash` text NOT NULL,
        `role` text DEFAULT 'user' NOT NULL
, `memory` text, `created_at` datetime, `updated_at` datetime, `deleted_at` datetime);
CREATE UNIQUE INDEX `users_email_unique` ON `users` (`email`);



Postgres Database Schema:
in database/migrations/20250723064003_init.sql

*/

func connectTurso() (*sql.DB, error) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DB_URL environment variable not set")
	}
	return sql.Open("libsql", dbURL)
}

func connectPostgres() (*sql.DB, error) {
	pgURL := os.Getenv("PG_DB_URL")
	if pgURL == "" {
		return nil, fmt.Errorf("PG_DB_URL environment variable not set")
	}
	return sql.Open("postgres", pgURL)
}

func listTables(db *sql.DB, dbType string) ([]string, error) {
	var query string
	if dbType == "turso" {
		query = "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%';"
	} else if dbType == "postgres" {
		query = "SELECT table_name FROM information_schema.tables WHERE table_schema='public';"
	} else {
		return nil, fmt.Errorf("unknown dbType: %s", dbType)
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, nil
}

func tableHasData(db *sql.DB, table string) (bool, error) {
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s;", table)
	row := db.QueryRow(query)
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func checkMigrationNeeded() error {
	tursoDB, err := connectTurso()
	if err != nil {
		return fmt.Errorf("failed to connect to Turso: %w", err)
	}
	defer tursoDB.Close()

	pgDB, err := connectPostgres()
	if err != nil {
		return fmt.Errorf("failed to connect to Postgres: %w", err)
	}
	defer pgDB.Close()

	tursoTables, err := listTables(tursoDB, "turso")
	if err != nil {
		return fmt.Errorf("failed to list tables in Turso: %w", err)
	}
	fmt.Println("Turso tables:", tursoTables)

	pgTables, err := listTables(pgDB, "postgres")
	if err != nil {
		return fmt.Errorf("failed to list tables in Postgres: %w", err)
	}
	fmt.Println("Postgres tables:", pgTables)

	pgTableSet := make(map[string]bool)
	for _, t := range pgTables {
		pgTableSet[t] = true
	}

	// Mapping from Turso table to Postgres table
	tableMap := map[string]string{
		"users":            "account",
		"tags":             "tag",
		"articles":         "article",
		"article_tags":     "article", // tags are now an array in article
		"about_page":       "page",
		"contact_page":     "page",
		"projects":         "project",
		"image_generation": "imagen_request",
	}

	type tableCheck struct {
		TursoTable    string
		PGTable       string
		ExistsInPG    bool
		TursoHasData  string
		PGHasData     string
		MigrationNeed string
		Error         string
	}

	var checks []tableCheck
	migrationNeeded := false
	for _, tursoTable := range tursoTables {
		if tursoTable == "__drizzle_migrations" || tursoTable == "goose_db_version" {
			continue // skip migration/meta tables
		}
		chk := tableCheck{TursoTable: tursoTable}
		pgTable, ok := tableMap[tursoTable]
		if !ok {
			chk.PGTable = "(no mapping)"
			chk.ExistsInPG = false
			chk.MigrationNeed = "NO"
			chk.Error = "No PG mapping"
			checks = append(checks, chk)
			continue
		}
		chk.PGTable = pgTable
		chk.ExistsInPG = pgTableSet[pgTable]

		hasData, err := tableHasData(tursoDB, tursoTable)
		if err != nil {
			chk.TursoHasData = "ERR"
			chk.Error = "Turso: " + err.Error()
		} else if hasData {
			chk.TursoHasData = "YES"
		} else {
			chk.TursoHasData = "NO"
		}

		if chk.ExistsInPG {
			pgHasData, err := tableHasData(pgDB, pgTable)
			if err != nil {
				chk.PGHasData = "ERR"
				chk.Error += " | PG: " + err.Error()
			} else if pgHasData {
				chk.PGHasData = "YES"
			} else {
				chk.PGHasData = "NO"
			}
		} else {
			chk.PGHasData = "N/A"
		}

		if chk.TursoHasData == "YES" && (chk.PGHasData == "NO" || chk.PGHasData == "N/A") {
			chk.MigrationNeed = "YES"
			migrationNeeded = true
		} else if chk.MigrationNeed == "" {
			chk.MigrationNeed = "NO"
		}

		checks = append(checks, chk)
	}

	// Print table header
	fmt.Printf("\n%-15s %-15s %-12s %-14s %-14s %-10s %-s\n", "Turso Table", "PG Table", "In Postgres", "Turso Has Data", "PG Has Data", "Migrate?", "Error")
	fmt.Println(strings.Repeat("-", 100))
	for _, chk := range checks {
		fmt.Printf("%-15s %-15s %-12v %-14s %-14s %-10s %-s\n", chk.TursoTable, chk.PGTable, chk.ExistsInPG, chk.TursoHasData, chk.PGHasData, chk.MigrationNeed, chk.Error)
	}

	if migrationNeeded {
		fmt.Println("\nMigration is needed.")
	} else {
		fmt.Println("\nNo migration needed.")
	}
	return nil
}

func printColumnMappings() {
	fmt.Println("\n==== Turso → Postgres Column Mappings ====")

	// Table: users → account
	fmt.Println("\nTurso: users  →  Postgres: account")
	fmt.Printf("%-22s %-22s\n", "Turso Column", "Postgres Column")
	fmt.Println(strings.Repeat("-", 44))
	fmt.Printf("%-22s %-22s\n", "name", "name")
	fmt.Printf("%-22s %-22s\n", "email", "email")
	fmt.Printf("%-22s %-22s\n", "passwordHash", "password_hash")
	fmt.Printf("%-22s %-22s\n", "role", "role")
	fmt.Printf("%-22s %-22s\n", "created_at", "created_at")
	fmt.Printf("%-22s %-22s\n", "updated_at", "updated_at")

	// Table: tags → tag
	fmt.Println("\nTurso: tags  →  Postgres: tag")
	fmt.Printf("%-22s %-22s\n", "Turso Column", "Postgres Column")
	fmt.Println(strings.Repeat("-", 44))
	fmt.Printf("%-22s %-22s\n", "tag_name", "name")
	fmt.Printf("%-22s %-22s\n", "created_at", "created_at")

	// Table: articles → article
	fmt.Println("\nTurso: articles  →  Postgres: article")
	fmt.Printf("%-30s %-22s\n", "Turso Column", "Postgres Column")
	fmt.Println(strings.Repeat("-", 52))
	fmt.Printf("%-30s %-22s\n", "slug", "slug")
	fmt.Printf("%-30s %-22s\n", "title", "title")
	fmt.Printf("%-30s %-22s\n", "content", "content")
	fmt.Printf("%-30s %-22s\n", "image", "image_url")
	fmt.Printf("%-30s %-22s\n", "author", "author_id")
	fmt.Printf("%-30s %-22s\n", "is_draft", "is_draft")
	fmt.Printf("%-30s %-22s\n", "embedding", "embedding")
	fmt.Printf("%-30s %-22s\n", "image_generation_request_id", "imagen_request_id")
	fmt.Printf("%-30s %-22s\n", "published_at", "published_at")
	fmt.Printf("%-30s %-22s\n", "chat_history", "session_memory")
	fmt.Printf("%-30s %-22s\n", "created_at", "created_at")
	fmt.Printf("%-30s %-22s\n", "updated_at", "updated_at")

	// Table: about_page/contact_page → page
	fmt.Println("\nTurso: about_page/contact_page  →  Postgres: page")
	fmt.Printf("%-22s %-22s\n", "Turso Column", "Postgres Column")
	fmt.Println(strings.Repeat("-", 44))
	fmt.Printf("%-22s %-22s\n", "title", "title")
	fmt.Printf("%-22s %-22s\n", "content", "content")
	fmt.Printf("%-22s %-22s\n", "meta_description", "description")
	fmt.Printf("%-22s %-22s\n", "profile_image", "image_url")
	fmt.Printf("%-22s %-22s\n", "last_updated", "updated_at")
	fmt.Printf("%-22s %-22s\n", "email_address", "(custom: JSON/meta_data)")
	fmt.Printf("%-22s %-22s\n", "social_links", "(custom: JSON/meta_data)")

	// Table: projects → project
	fmt.Println("\nTurso: projects  →  Postgres: project")
	fmt.Printf("%-22s %-22s\n", "Turso Column", "Postgres Column")
	fmt.Println(strings.Repeat("-", 44))
	fmt.Printf("%-22s %-22s\n", "title", "title")
	fmt.Printf("%-22s %-22s\n", "description", "description")
	fmt.Printf("%-22s %-22s\n", "url", "url")
	fmt.Printf("%-22s %-22s\n", "image", "image_url")
	fmt.Printf("%-22s %-22s\n", "created_at", "created_at")
	fmt.Printf("%-22s %-22s\n", "updated_at", "updated_at")

	// Table: image_generation → imagen_request
	fmt.Println("\nTurso: image_generation  →  Postgres: imagen_request")
	fmt.Printf("%-22s %-22s\n", "Turso Column", "Postgres Column")
	fmt.Println(strings.Repeat("-", 44))
	fmt.Printf("%-22s %-22s\n", "prompt", "prompt")
	fmt.Printf("%-22s %-22s\n", "provider", "provider")
	fmt.Printf("%-22s %-22s\n", "model", "model_name")
	fmt.Printf("%-22s %-22s\n", "request_id", "request_id")
	fmt.Printf("%-22s %-22s\n", "output_url", "output_url")
	fmt.Printf("%-22s %-22s\n", "storage_key", "(custom: file_index_id)")
	fmt.Printf("%-22s %-22s\n", "created_at", "created_at")

	// Table: article_tags → article.tag_ids
	fmt.Println("\nTurso: article_tags  →  Postgres: article.tag_ids (array)")
	fmt.Printf("%-22s %-22s\n", "Turso Column", "Postgres Column")
	fmt.Println(strings.Repeat("-", 44))
	fmt.Printf("%-22s %-22s\n", "article_id", "article.id (links)")
	fmt.Printf("%-22s %-22s\n", "tag_id", "article.tag_ids (array of tag.id)")
}

func main() {
	fmt.Println("Starting Turso to Postgres migration...")

	printColumnMappings()

	if err := checkMigrationNeeded(); err != nil {
		log.Fatalf("Check failed: %v", err)
	}

	fmt.Println("Migration check complete.")
	os.Exit(0)
}
