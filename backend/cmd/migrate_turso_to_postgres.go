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

func printTableReport(tursoTable, pgTable string, tursoCols, pgCols []string, colMapping [][2]string, tursoDB, pgDB *sql.DB) {
	fmt.Printf("\n====================\nTurso: %-15s → Postgres: %-15s\n====================\n", tursoTable, pgTable)
	fmt.Printf("%-22s | %-22s\n", "Turso Column", "Postgres Column")
	fmt.Println(strings.Repeat("-", 47))
	for _, pair := range colMapping {
		fmt.Printf("%-22s | %-22s\n", pair[0], pair[1])
	}

	fmt.Println("\nTable Data Check:")
	tursoHasData, tursoErr := tableHasData(tursoDB, tursoTable)
	pgHasData, pgErr := tableHasData(pgDB, pgTable)
	fmt.Printf("  Turso table '%s' has data: %v\n", tursoTable, valueOrErr(tursoHasData, tursoErr))
	fmt.Printf("  Postgres table '%s' has data: %v\n", pgTable, valueOrErr(pgHasData, pgErr))
	if tursoErr == nil && pgErr == nil {
		if tursoHasData && !pgHasData {
			fmt.Println("  → Migration needed.")
		} else {
			fmt.Println("  → No migration needed.")
		}
	}
}

func printArticleTagsReport() {
	fmt.Printf("\n====================\nTurso: %-15s → Postgres: %-15s\n====================\n", "article_tags", "article.tag_ids (array)")
	fmt.Printf("%-22s | %-22s\n", "Turso Column", "Postgres Column")
	fmt.Println(strings.Repeat("-", 47))
	fmt.Printf("%-22s | %-22s\n", "article_id", "article.id (links)")
	fmt.Printf("%-22s | %-22s\n", "tag_id", "article.tag_ids (array of tag.id)")
	fmt.Println("\n  This join table is migrated by collecting all tag_ids for each article and storing them in the tag_ids array in the article table.")
}

func valueOrErr(val bool, err error) string {
	if err != nil {
		return "ERR: " + err.Error()
	}
	if val {
		return "YES"
	}
	return "NO"
}

func main() {
	fmt.Println("\n===== Turso to Postgres Migration Plan & Check =====\n")

	tursoDB, err := connectTurso()
	if err != nil {
		log.Fatalf("failed to connect to Turso: %v", err)
	}
	defer tursoDB.Close()

	pgDB, err := connectPostgres()
	if err != nil {
		log.Fatalf("failed to connect to Postgres: %v", err)
	}
	defer pgDB.Close()

	// users → account
	printTableReport(
		"users", "account",
		[]string{"name", "email", "passwordHash", "role", "created_at", "updated_at"},
		[]string{"name", "email", "password_hash", "role", "created_at", "updated_at"},
		[][2]string{
			{"name", "name"},
			{"email", "email"},
			{"passwordHash", "password_hash"},
			{"role", "role"},
			{"created_at", "created_at"},
			{"updated_at", "updated_at"},
		}, tursoDB, pgDB)

	// tags → tag
	printTableReport(
		"tags", "tag",
		[]string{"tag_name", "created_at"},
		[]string{"name", "created_at"},
		[][2]string{
			{"tag_name", "name"},
			{"created_at", "created_at"},
		}, tursoDB, pgDB)

	// articles → article
	printTableReport(
		"articles", "article",
		[]string{"slug", "title", "content", "image", "author", "is_draft", "embedding", "image_generation_request_id", "published_at", "chat_history", "created_at", "updated_at"},
		[]string{"slug", "title", "content", "image_url", "author_id", "is_draft", "embedding", "imagen_request_id", "published_at", "session_memory", "created_at", "updated_at"},
		[][2]string{
			{"slug", "slug"},
			{"title", "title"},
			{"content", "content"},
			{"image", "image_url"},
			{"author", "author_id"},
			{"is_draft", "is_draft"},
			{"embedding", "embedding"},
			{"image_generation_request_id", "imagen_request_id"},
			{"published_at", "published_at"},
			{"chat_history", "session_memory"},
			{"created_at", "created_at"},
			{"updated_at", "updated_at"},
		}, tursoDB, pgDB)

	// article_tags → article.tag_ids (array)
	printArticleTagsReport()

	// about_page → page
	printTableReport(
		"about_page", "page",
		[]string{"title", "content", "meta_description", "profile_image", "last_updated"},
		[]string{"title", "content", "description", "image_url", "updated_at"},
		[][2]string{
			{"title", "title"},
			{"content", "content"},
			{"meta_description", "description"},
			{"profile_image", "image_url"},
			{"last_updated", "updated_at"},
		}, tursoDB, pgDB)

	// contact_page → page
	printTableReport(
		"contact_page", "page",
		[]string{"title", "content", "email_address", "social_links", "meta_description", "last_updated"},
		[]string{"title", "content", "description", "image_url", "updated_at"},
		[][2]string{
			{"title", "title"},
			{"content", "content"},
			{"email_address", "(custom: JSON/meta_data)"},
			{"social_links", "(custom: JSON/meta_data)"},
			{"meta_description", "description"},
			{"last_updated", "updated_at"},
		}, tursoDB, pgDB)

	// projects → project
	printTableReport(
		"projects", "project",
		[]string{"title", "description", "url", "image"},
		[]string{"title", "description", "url", "image_url"},
		[][2]string{
			{"title", "title"},
			{"description", "description"},
			{"url", "url"},
			{"image", "image_url"},
		}, tursoDB, pgDB)

	// image_generation → imagen_request
	printTableReport(
		"image_generation", "imagen_request",
		[]string{"prompt", "provider", "model", "request_id", "output_url", "storage_key", "created_at"},
		[]string{"prompt", "provider", "model_name", "request_id", "output_url", "file_index_id", "created_at"},
		[][2]string{
			{"prompt", "prompt"},
			{"provider", "provider"},
			{"model", "model_name"},
			{"request_id", "request_id"},
			{"output_url", "output_url"},
			{"storage_key", "(custom: file_index_id)"},
			{"created_at", "created_at"},
		}, tursoDB, pgDB)

	fmt.Println("\nMigration check complete.")
	os.Exit(0)
}
