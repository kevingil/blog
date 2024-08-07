package server

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/kevingil/blog/internal/controllers"
	"github.com/kevingil/blog/internal/helpers"
)

func Serve() {
	// Create a new engine by passing the template folder
	// and template extension using <engine>.New(dir, ext string)
	engine := html.New("./internal/templates", ".gohtml")
	// Add your helper functions to the template's global function map.
	engine.AddFunc("until", helpers.Until)
	engine.AddFunc("date", helpers.Date)
	engine.AddFunc("shortDate", helpers.ShortDate)
	engine.AddFunc("v", helpers.V)
	engine.AddFunc("mdToHTML", helpers.MdToHTML)
	engine.AddFunc("truncate", helpers.Truncate)
	engine.AddFunc("draft", helpers.Draft)
	engine.AddFunc("ValidateEmail", helpers.ValidateEmail)
	engine.AddFunc("sub", func(a, b int) int {
		return a - b
	})
	engine.AddFunc("add", func(a, b int) int {
		return a + b
	})

	engine.Debug(true)

	app := fiber.New(fiber.Config{
		Views:       engine,
		ViewsLayout: "layout",
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			// Status code defaults to 500
			code := fiber.StatusInternalServerError

			// Retrieve the custom status code if it's a *fiber.Error
			var e *fiber.Error
			if errors.As(err, &e) {
				code = e.Code
			}

			// Redirect to custom 404 page
			if code == fiber.StatusNotFound {
				return ctx.Redirect("/404", fiber.StatusMovedPermanently)
			}

			// Redirect error page with error info
			errorMessage := err.Error()
			return ctx.Redirect(fmt.Sprintf("/error?code=%d&message=%s", code, errorMessage), fiber.StatusMovedPermanently)
		},
	})

	// Serve static files
	app.Static("/", "./web")

	// Error pages
	app.Get("/404", func(ctx *fiber.Ctx) error {
		layout := "layout"
		if ctx.Get("HX-Request") == "true" {
			layout = ""
		}
		return ctx.Status(fiber.StatusNotFound).Render("404", fiber.Map{}, layout)

	})

	app.Get("/error", func(ctx *fiber.Ctx) error {
		code := ctx.QueryInt("code", fiber.StatusInternalServerError)
		message := ctx.Query("message", "No error information.")
		layout := "layout"
		if ctx.Get("HX-Request") == "true" {
			layout = ""
		}

		return ctx.Status(code).Render("error", fiber.Map{
			"Code":    code,
			"Message": message,
		}, layout)
	})

	// User login, logout, register
	app.Get("/login", controllers.LoginPage)
	app.Post("/login", controllers.AuthenticateUser)
	app.Get("/logout", controllers.Logout)
	app.Get("/register", controllers.Register)

	// View posts, preview drafts
	app.Get("/blog", controllers.BlogPage)

	// Services
	app.Get("/blog/partial/recent", controllers.RecentPostsPartial)

	// View posts, preview drafts
	app.Get("/blog/:slug", controllers.BlogPostPage)
	app.Get("/post/:slug", controllers.RedirectDeprecatedUrlPrefix)

	// User admin
	app.Get("/admin", controllers.AdminPage)
	app.Get("/analytics/visits", controllers.GetSiteVisits)
	app.Get("/analytics/list-top-pages", controllers.ListTopPages)
	app.Get("/analytics/site-visits-chart", controllers.GetSiteVisitsChart)

	// Edit articles, delete, or create new
	// View posts, preview drafts
	app.Get("/admin/articles", controllers.EditArticlesPage)
	app.Post("/admin/articles", controllers.EditArticle)
	app.Post("/admin/articles", controllers.DeleteArticle)
	app.Get("/admin/articles/edit", controllers.EditArticlePage)

	// User Profile
	// Edit about me, skills, and projects
	app.Get("/admin/profile", controllers.EditProfilePage)
	app.Post("/admin/profile", controllers.EditProfile)

	// Homepage projects
	app.Get("/admin/projects", controllers.EditProjects)
	app.Post("/admin/projects", controllers.EditProjects)

	// Files
	app.Get("/admin/files", controllers.AdminFilesPage)
	app.Get("/admin/files/content", controllers.FilesContent)
	app.Post("/admin/files/upload", controllers.HandleFileUpload)
	app.Post("/admin/files/delete", controllers.HandleFileDelete)
	app.Post("/admin/files/directory", controllers.UpdateDirectory)
	app.Post("/admin/files/directory/new", controllers.CreateNewDirectory)

	// Pages
	app.Get("/about", controllers.About)
	app.Get("/contact", controllers.Contact)

	// Catch-all route for index
	app.Get("/", controllers.Index)

	log.Printf("Your app is running on port %s.", os.Getenv("PORT"))
	log.Fatal(app.Listen(":" + os.Getenv("PORT")))
}
