package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lib/pq"
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

func migrateUsers(tursoDB, pgDB *sql.DB) (inserted, skipped, errored int, userIDMap map[int]string) {
	fmt.Println("\nMigrating users...")
	userIDMap = make(map[int]string) // Turso id -> Postgres UUID
	rows, err := tursoDB.Query("SELECT id, name, email, passwordHash, role, created_at, updated_at FROM users")
	if err != nil {
		fmt.Printf("  Error querying Turso users: %v\n", err)
		return 0, 0, 1, userIDMap
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name, email, passwordHash, role string
		var createdAt, updatedAt interface{}
		var pgID string
		err := rows.Scan(&id, &name, &email, &passwordHash, &role, &createdAt, &updatedAt)
		if err != nil {
			fmt.Printf("  Error scanning Turso user: %v\n", err)
			errored++
			continue
		}

		// Check for duplicate by email
		err = pgDB.QueryRow("SELECT id FROM account WHERE email = $1", email).Scan(&pgID)
		if err == sql.ErrNoRows {
			// Insert
			err = pgDB.QueryRow(`INSERT INTO account (name, email, password_hash, role, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
				name, email, passwordHash, role, createdAt, updatedAt).Scan(&pgID)
			if err != nil {
				fmt.Printf("  Error inserting user %s: %v\n", email, err)
				errored++
				continue
			}
			fmt.Printf("  Inserted: %s\n", email)
			inserted++
		} else if err != nil {
			fmt.Printf("  Error checking Postgres user: %v\n", err)
			errored++
			continue
		} else {
			fmt.Printf("  Skipped (exists): %s\n", email)
			skipped++
		}
		userIDMap[id] = pgID
	}
	return
}

func migrateTags(tursoDB, pgDB *sql.DB) (inserted, skipped, errored int, tagIDMap map[int]int) {
	fmt.Println("\nMigrating tags...")
	tagIDMap = make(map[int]int)
	rows, err := tursoDB.Query("SELECT tag_id, tag_name FROM tags")
	if err != nil {
		fmt.Printf("  Error querying Turso tags: %v\n", err)
		return 0, 0, 1, tagIDMap
	}
	defer rows.Close()

	for rows.Next() {
		var tagID int
		var tagName string
		err := rows.Scan(&tagID, &tagName)
		if err != nil {
			fmt.Printf("  Error scanning Turso tag: %v\n", err)
			errored++
			continue
		}

		// Check for duplicate by name
		var newID int
		err = pgDB.QueryRow("SELECT id FROM tag WHERE name = $1", tagName).Scan(&newID)
		if err == sql.ErrNoRows {
			// Insert
			err = pgDB.QueryRow("INSERT INTO tag (name) VALUES ($1) RETURNING id", tagName).Scan(&newID)
			if err != nil {
				fmt.Printf("  Error inserting tag %s: %v\n", tagName, err)
				errored++
				continue
			}
			fmt.Printf("  Inserted: %s\n", tagName)
			inserted++
		} else if err != nil {
			fmt.Printf("  Error checking Postgres tag: %v\n", err)
			errored++
			continue
		} else {
			fmt.Printf("  Skipped (exists): %s\n", tagName)
			skipped++
		}
		tagIDMap[tagID] = newID
	}
	return
}

func migrateArticles(tursoDB, pgDB *sql.DB, userIDMap map[int]string, tagIDMap map[int]int, reqIDMap map[string]string) (inserted, skipped, errored int, articleIDMap map[int]string) {
	fmt.Println("\nMigrating articles...")
	articleIDMap = make(map[int]string)
	rows, err := tursoDB.Query(`SELECT id, slug, title, content, image, author, is_draft, embedding, image_generation_request_id, published_at, chat_history, created_at, updated_at FROM articles`)
	if err != nil {
		fmt.Printf("  Error querying Turso articles: %v\n", err)
		return 0, 0, 1, articleIDMap
	}
	defer rows.Close()

	for rows.Next() {
		var id, author int
		var slug, title, content, image, imageGenReqID string
		var isDraft bool
		var embedding, chatHistory []byte
		var publishedAt, createdAt, updatedAt interface{}
		err := rows.Scan(&id, &slug, &title, &content, &image, &author, &isDraft, &embedding, &imageGenReqID, &publishedAt, &chatHistory, &createdAt, &updatedAt)
		if err != nil {
			fmt.Printf("  Error scanning Turso article: %v\n", err)
			errored++
			continue
		}

		// Map author_id
		pgAuthorID := userIDMap[author]
		// Map image_generation_request_id if possible
		var pgImageGenReqID interface{} = nil
		if imageGenReqID != "" && reqIDMap != nil {
			if newReqID, ok := reqIDMap[imageGenReqID]; ok {
				pgImageGenReqID = newReqID
			}
		}

		// Check for duplicate by slug
		var newID string
		err = pgDB.QueryRow("SELECT id FROM article WHERE slug = $1", slug).Scan(&newID)
		if err == sql.ErrNoRows {
			// Insert
			err = pgDB.QueryRow(`INSERT INTO article (slug, title, content, image_url, author_id, is_draft, embedding, imagen_request_id, published_at, session_memory, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id`,
				slug, title, content, image, pgAuthorID, isDraft, embedding, pgImageGenReqID, publishedAt, chatHistory, createdAt, updatedAt).Scan(&newID)
			if err != nil {
				fmt.Printf("  Error inserting article %s: %v\n", slug, err)
				errored++
				continue
			}
			fmt.Printf("  Inserted: %s\n", slug)
			inserted++
		} else if err != nil {
			fmt.Printf("  Error checking Postgres article: %v\n", err)
			errored++
			continue
		} else {
			fmt.Printf("  Skipped (exists): %s\n", slug)
			skipped++
		}
		articleIDMap[id] = newID
	}
	return
}

func migrateArticleTags(tursoDB, pgDB *sql.DB, articleIDMap map[int]string, tagIDMap map[int]int) (updated, errored int) {
	fmt.Println("\nMigrating article_tags to article.tag_ids array...")
	// For each article, collect tag_ids and update article.tag_ids
	rows, err := tursoDB.Query("SELECT article_id, tag_id FROM article_tags")
	if err != nil {
		fmt.Printf("  Error querying Turso article_tags: %v\n", err)
		return 0, 1
	}
	defer rows.Close()

	articleTags := make(map[int][]int)
	for rows.Next() {
		var articleID, tagID int
		err := rows.Scan(&articleID, &tagID)
		if err != nil {
			fmt.Printf("  Error scanning article_tag: %v\n", err)
			errored++
			continue
		}
		articleTags[articleID] = append(articleTags[articleID], tagID)
	}

	for oldArticleID, tagIDs := range articleTags {
		newArticleID, ok := articleIDMap[oldArticleID]
		if !ok {
			continue // Article not migrated
		}
		var newTagIDs []int
		for _, oldTagID := range tagIDs {
			if newTagID, ok := tagIDMap[oldTagID]; ok {
				newTagIDs = append(newTagIDs, newTagID)
			}
		}
		if len(newTagIDs) == 0 {
			continue
		}
		_, err := pgDB.Exec("UPDATE article SET tag_ids = $1 WHERE id = $2", pq.Array(newTagIDs), newArticleID)
		if err != nil {
			fmt.Printf("  Error updating article %s tag_ids: %v\n", newArticleID, err)
			errored++
			continue
		}
		updated++
	}
	return
}

func migratePages(tursoDB, pgDB *sql.DB, tableName string) (inserted, skipped, errored int) {
	fmt.Printf("\nMigrating %s...\n", tableName)
	rows, err := tursoDB.Query(fmt.Sprintf("SELECT title, content, meta_description, profile_image, last_updated FROM %s", tableName))
	if err != nil {
		fmt.Printf("  Error querying Turso %s: %v\n", tableName, err)
		return 0, 0, 1
	}
	defer rows.Close()

	for rows.Next() {
		var title, content, metaDescription, profileImage, lastUpdated string
		err := rows.Scan(&title, &content, &metaDescription, &profileImage, &lastUpdated)
		if err != nil {
			fmt.Printf("  Error scanning %s: %v\n", tableName, err)
			errored++
			continue
		}
		// Check for duplicate by title
		var exists bool
		err = pgDB.QueryRow("SELECT EXISTS (SELECT 1 FROM page WHERE title = $1)", title).Scan(&exists)
		if err != nil {
			fmt.Printf("  Error checking Postgres page: %v\n", err)
			errored++
			continue
		}
		if exists {
			fmt.Printf("  Skipped (exists): %s\n", title)
			skipped++
			continue
		}
		_, err = pgDB.Exec(`INSERT INTO page (title, content, description, image_url, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			title, content, metaDescription, profileImage, lastUpdated)
		if err != nil {
			fmt.Printf("  Error inserting page %s: %v\n", title, err)
			errored++
			continue
		}
		fmt.Printf("  Inserted: %s\n", title)
		inserted++
	}
	return
}

func migrateProjects(tursoDB, pgDB *sql.DB) (inserted, skipped, errored int) {
	fmt.Println("\nMigrating projects...")
	rows, err := tursoDB.Query("SELECT title, description, url, image FROM projects")
	if err != nil {
		fmt.Printf("  Error querying Turso projects: %v\n", err)
		return 0, 0, 1
	}
	defer rows.Close()

	for rows.Next() {
		var title, description, url, image string
		err := rows.Scan(&title, &description, &url, &image)
		if err != nil {
			fmt.Printf("  Error scanning project: %v\n", err)
			errored++
			continue
		}
		// Check for duplicate by title
		var exists bool
		err = pgDB.QueryRow("SELECT EXISTS (SELECT 1 FROM project WHERE title = $1)", title).Scan(&exists)
		if err != nil {
			fmt.Printf("  Error checking Postgres project: %v\n", err)
			errored++
			continue
		}
		if exists {
			fmt.Printf("  Skipped (exists): %s\n", title)
			skipped++
			continue
		}
		_, err = pgDB.Exec(`INSERT INTO project (title, description, url, image_url) VALUES ($1, $2, $3, $4)`,
			title, description, url, image)
		if err != nil {
			fmt.Printf("  Error inserting project %s: %v\n", title, err)
			errored++
			continue
		}
		fmt.Printf("  Inserted: %s\n", title)
		inserted++
	}
	return
}

func migrateImageGeneration(tursoDB, pgDB *sql.DB) (inserted, skipped, errored int) {
	fmt.Println("\nMigrating image_generation...")
	rows, err := tursoDB.Query("SELECT prompt, provider, model, request_id, output_url, storage_key, created_at FROM image_generation")
	if err != nil {
		fmt.Printf("  Error querying Turso image_generation: %v\n", err)
		return 0, 0, 1
	}
	defer rows.Close()

	for rows.Next() {
		var prompt, provider, model, requestID, outputURL, storageKey string
		var createdAt interface{}
		err := rows.Scan(&prompt, &provider, &model, &requestID, &outputURL, &storageKey, &createdAt)
		if err != nil {
			fmt.Printf("  Error scanning image_generation: %v\n", err)
			errored++
			continue
		}
		// Check for duplicate by request_id
		var exists bool
		err = pgDB.QueryRow("SELECT EXISTS (SELECT 1 FROM imagen_request WHERE request_id = $1)", requestID).Scan(&exists)
		if err != nil {
			fmt.Printf("  Error checking Postgres imagen_request: %v\n", err)
			errored++
			continue
		}
		if exists {
			fmt.Printf("  Skipped (exists): %s\n", requestID)
			skipped++
			continue
		}
		_, err = pgDB.Exec(`INSERT INTO imagen_request (prompt, provider, model_name, request_id, output_url, created_at) VALUES ($1, $2, $3, $4, $5, $6)`,
			prompt, provider, model, requestID, outputURL, createdAt)
		if err != nil {
			fmt.Printf("  Error inserting imagen_request %s: %v\n", requestID, err)
			errored++
			continue
		}
		fmt.Printf("  Inserted: %s\n", requestID)
		inserted++
	}
	return
}

func main() {
	checkFlag := flag.Bool("check", false, "Print migration plan and check (default)")
	runFlag := flag.Bool("run", false, "Run the migration (will prompt for confirmation)")
	flag.Parse()

	if !*checkFlag && !*runFlag {
		fmt.Println("\nUsage: go run migrate_turso_to_postgres.go [flags]\n")
		fmt.Println("Flags:")
		fmt.Println("  -check   Print migration plan and check (default)")
		fmt.Println("  -run     Run the migration (will prompt for confirmation)")
		os.Exit(0)
	}

	if *runFlag {
		fmt.Println("\nWARNING: You are about to run the migration. This will modify your Postgres database.")
		fmt.Print("Are you sure you want to continue? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		resp, _ := reader.ReadString('\n')
		if resp != "y\n" && resp != "Y\n" {
			fmt.Println("Aborted.")
			os.Exit(0)
		}

		fmt.Println("\n===== Running Turso to Postgres Migration =====\n")

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

		// USERS
		uInserted, uSkipped, uErrored, userIDMap := migrateUsers(tursoDB, pgDB)
		fmt.Printf("\nUsers: %d inserted, %d skipped, %d errored\n", uInserted, uSkipped, uErrored)

		// TAGS
		tInserted, tSkipped, tErrored, tagIDMap := migrateTags(tursoDB, pgDB)
		fmt.Printf("\nTags: %d inserted, %d skipped, %d errored\n", tInserted, tSkipped, tErrored)

		// IMAGE_GENERATION (for mapping request_id)
		var rows *sql.Rows
		var reqIDMap map[string]string
		igInserted, igSkipped, igErrored := migrateImageGeneration(tursoDB, pgDB)
		fmt.Printf("\nImage Generation: %d inserted, %d skipped, %d errored\n", igInserted, igSkipped, igErrored)
		// Build reqIDMap for articles
		reqIDMap = make(map[string]string)
		rows, err = pgDB.Query("SELECT request_id, id FROM imagen_request")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var reqID, pgID string
				_ = rows.Scan(&reqID, &pgID)
				if reqID != "" {
					reqIDMap[reqID] = pgID
				}
			}
		}

		// ARTICLES
		var aInserted, aSkipped, aErrored int
		var articleIDMap map[int]string
		aInserted, aSkipped, aErrored, articleIDMap = migrateArticles(tursoDB, pgDB, userIDMap, tagIDMap, reqIDMap)
		fmt.Printf("\nArticles: %d inserted, %d skipped, %d errored\n", aInserted, aSkipped, aErrored)

		// ARTICLE_TAGS
		var atUpdated, atErrored int
		atUpdated, atErrored = migrateArticleTags(tursoDB, pgDB, articleIDMap, tagIDMap)
		fmt.Printf("\nArticle Tags: %d updated, %d errored\n", atUpdated, atErrored)

		// ABOUT_PAGE
		var abInserted, abSkipped, abErrored int
		abInserted, abSkipped, abErrored = migratePages(tursoDB, pgDB, "about_page")
		fmt.Printf("\nAbout Page: %d inserted, %d skipped, %d errored\n", abInserted, abSkipped, abErrored)

		// CONTACT_PAGE
		var coInserted, coSkipped, coErrored int
		coInserted, coSkipped, coErrored = migratePages(tursoDB, pgDB, "contact_page")
		fmt.Printf("\nContact Page: %d inserted, %d skipped, %d errored\n", coInserted, coSkipped, coErrored)

		// PROJECTS
		var pInserted, pSkipped, pErrored int
		pInserted, pSkipped, pErrored = migrateProjects(tursoDB, pgDB)
		fmt.Printf("\nProjects: %d inserted, %d skipped, %d errored\n", pInserted, pSkipped, pErrored)

		// IMAGE_GENERATION
		igInserted, igSkipped, igErrored = migrateImageGeneration(tursoDB, pgDB)
		fmt.Printf("\nImage Generation: %d inserted, %d skipped, %d errored\n", igInserted, igSkipped, igErrored)

		fmt.Println("\nMigration complete.")
		os.Exit(0)
	}

	// Default: check
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
