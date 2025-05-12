package server

import (
	"blog-agent-go/backend/services"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/golang-jwt/jwt/v5"

	"gorm.io/gorm"
)

func (s *FiberServer) RegisterRoutes() {
	// Apply CORS middleware
	s.App.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS,PATCH",
		AllowHeaders:     "Accept,Authorization,Content-Type",
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Pages routes
	pages := s.App.Group("/pages")
	pages.Get("/about", s.GetAboutPageHandler)
	pages.Get("/contact", s.GetContactPageHandler)

	// Auth routes
	auth := s.App.Group("/auth")
	auth.Post("/login", s.LoginHandler)
	auth.Post("/register", s.RegisterHandler)
	auth.Post("/logout", s.LogoutHandler)
	auth.Put("/account", s.UpdateAccountHandler)
	auth.Put("/password", s.UpdatePasswordHandler)
	auth.Delete("/account", s.DeleteAccountHandler)

	// Blog routes
	blog := s.App.Group("/blog")
	blog.Post("/generate", s.GenerateArticleHandler)
	blog.Get("/:id/chat-history", s.GetArticleChatHistoryHandler)
	blog.Put("/:id/update", s.UpdateArticleWithContextHandler)
	blog.Get("/articles/:slug", s.GetArticleDataHandler)
	blog.Get("/articles/:slug/metadata", s.GetArticleMetadataHandler)
	blog.Get("/articles/:id/recommended", s.GetRecommendedArticlesHandler)
	blog.Get("/articles/dashboard", s.GetDashboardArticlesHandler)
	blog.Delete("/articles/:id", s.DeleteArticleHandler)

	// Add new blog routes
	blog.Get("/articles", s.GetArticlesHandler)
	blog.Get("/articles/search", s.SearchArticlesHandler)
	blog.Get("/tags/popular", s.GetPopularTagsHandler)

	// Image generation routes
	images := s.App.Group("/images")
	images.Post("/generate", s.GenerateArticleImageHandler)
	images.Get("/:requestId", s.GetImageGenerationHandler)
	images.Get("/:requestId/status", s.GetImageGenerationStatusHandler)

	// Storage routes
	storage := s.App.Group("/storage")
	storage.Get("/files", s.ListFilesHandler)
	storage.Post("/upload", s.UploadFileHandler)
	storage.Delete("/:key", s.DeleteFileHandler)
	storage.Post("/folders", s.CreateFolderHandler)
	storage.Put("/folders", s.UpdateFolderHandler)

	s.App.Get("/", s.HelloWorldHandler)

	s.App.Get("/health", s.healthHandler)

	s.App.Get("/websocket", websocket.New(s.websocketHandler))
}

func (s *FiberServer) HelloWorldHandler(c *fiber.Ctx) error {
	resp := fiber.Map{
		"message": "Hello World",
	}

	return c.JSON(resp)
}

func (s *FiberServer) healthHandler(c *fiber.Ctx) error {
	// TODO: Implement proper health check
	return c.JSON(fiber.Map{
		"status": "ok",
	})
}

func (s *FiberServer) websocketHandler(con *websocket.Conn) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for {
			_, _, err := con.ReadMessage()
			if err != nil {
				cancel()
				log.Println("Receiver Closing", err)
				break
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			payload := fmt.Sprintf("server timestamp: %d", time.Now().UnixNano())
			if err := con.WriteMessage(websocket.TextMessage, []byte(payload)); err != nil {
				log.Printf("could not write to socket: %v", err)
				return
			}
			time.Sleep(time.Second * 2)
		}
	}
}

func (s *FiberServer) LoginHandler(c *fiber.Ctx) error {
	var req services.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		fmt.Println("Error parsing request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	resp, err := s.authService.Login(req)
	if err != nil {
		fmt.Println("Error logging in:", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	//fmt.Println("Login response:", resp)
	return c.JSON(resp)
}

func (s *FiberServer) RegisterHandler(c *fiber.Ctx) error {
	var req services.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := s.authService.Register(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User registered successfully",
	})
}

func (s *FiberServer) LogoutHandler(c *fiber.Ctx) error {
	// Since we're using JWT tokens, we don't need to do anything on the server side
	// The client should remove the token from their storage
	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

func (s *FiberServer) UpdateAccountHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	// Remove "Bearer " prefix
	token = token[7:]

	// Validate token and get user ID
	validToken, err := s.authService.ValidateToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid token",
		})
	}

	claims := validToken.Claims.(jwt.MapClaims)
	userID := uint(claims["sub"].(float64))

	var req services.UpdateAccountRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := s.authService.UpdateAccount(userID, req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Account updated successfully",
	})
}

func (s *FiberServer) UpdatePasswordHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	// Remove "Bearer " prefix
	token = token[7:]

	// Validate token and get user ID
	validToken, err := s.authService.ValidateToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid token",
		})
	}

	claims := validToken.Claims.(jwt.MapClaims)
	userID := uint(claims["sub"].(float64))

	var req services.UpdatePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := s.authService.UpdatePassword(userID, req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Password updated successfully",
	})
}

func (s *FiberServer) DeleteAccountHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	// Remove "Bearer " prefix
	token = token[7:]

	// Validate token and get user ID
	validToken, err := s.authService.ValidateToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid token",
		})
	}

	claims := validToken.Claims.(jwt.MapClaims)
	userID := uint(claims["sub"].(float64))

	var req struct {
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := s.authService.DeleteAccount(userID, req.Password); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Account deleted successfully",
	})
}

// Blog handlers
func (s *FiberServer) GenerateArticleHandler(c *fiber.Ctx) error {
	var req struct {
		Prompt  string `json:"prompt"`
		Title   string `json:"title"`
		IsDraft bool   `json:"is_draft"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get user ID from session
	userID := c.Locals("userID").(int64)

	article, err := s.blogService.GenerateArticle(c.Context(), req.Prompt, req.Title, userID, req.IsDraft)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(article)
}

func (s *FiberServer) GetArticleChatHistoryHandler(c *fiber.Ctx) error {
	articleID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid article ID",
		})
	}

	history, err := s.blogService.GetArticleChatHistory(c.Context(), int64(articleID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(history)
}

func (s *FiberServer) UpdateArticleWithContextHandler(c *fiber.Ctx) error {
	articleID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid article ID",
		})
	}

	article, err := s.blogService.UpdateArticleWithContext(c.Context(), int64(articleID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(article)
}

// Image generation handlers
func (s *FiberServer) GenerateArticleImageHandler(c *fiber.Ctx) error {
	var req struct {
		Prompt         string `json:"prompt"`
		ArticleID      int64  `json:"article_id"`
		GeneratePrompt bool   `json:"generate_prompt"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	imageGen, err := s.imageService.GenerateArticleImage(c.Context(), req.Prompt, req.ArticleID, req.GeneratePrompt)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(imageGen)
}

func (s *FiberServer) GetImageGenerationHandler(c *fiber.Ctx) error {
	requestID := c.Params("requestId")
	if requestID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request ID",
		})
	}

	imageGen, err := s.imageService.GetImageGeneration(c.Context(), requestID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(imageGen)
}

func (s *FiberServer) GetImageGenerationStatusHandler(c *fiber.Ctx) error {
	requestID := c.Params("requestId")
	if requestID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request ID",
		})
	}

	status, err := s.imageService.GetImageGenerationStatus(c.Context(), requestID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(status)
}

// Storage handlers
func (s *FiberServer) ListFilesHandler(c *fiber.Ctx) error {
	prefix := c.Query("prefix")
	files, folders, err := s.storageService.ListFiles(c.Context(), prefix)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"files":   files,
		"folders": folders,
	})
}

func (s *FiberServer) UploadFileHandler(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file upload",
		})
	}

	key := c.FormValue("key")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Key is required",
		})
	}

	data, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	defer data.Close()

	buf := make([]byte, file.Size)
	_, err = data.Read(buf)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := s.storageService.UploadFile(c.Context(), key, buf); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "File uploaded successfully",
	})
}

func (s *FiberServer) DeleteFileHandler(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Key is required",
		})
	}

	if err := s.storageService.DeleteFile(c.Context(), key); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "File deleted successfully",
	})
}

func (s *FiberServer) CreateFolderHandler(c *fiber.Ctx) error {
	var req struct {
		Path string `json:"path"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := s.storageService.CreateFolder(c.Context(), req.Path); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Folder created successfully",
	})
}

func (s *FiberServer) UpdateFolderHandler(c *fiber.Ctx) error {
	var req struct {
		OldPath string `json:"old_path"`
		NewPath string `json:"new_path"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := s.storageService.UpdateFolder(c.Context(), req.OldPath, req.NewPath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Folder updated successfully",
	})
}

func (s *FiberServer) GetArticlesHandler(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	tag := c.Query("tag", "")

	response, err := s.blogService.GetArticles(page, tag)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(response)
}

func (s *FiberServer) SearchArticlesHandler(c *fiber.Ctx) error {
	query := c.Query("query")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Query parameter is required",
		})
	}

	page := c.QueryInt("page", 1)
	tag := c.Query("tag", "")

	response, err := s.blogService.SearchArticles(query, page, tag)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(response)
}

func (s *FiberServer) GetPopularTagsHandler(c *fiber.Ctx) error {
	tags, err := s.blogService.GetPopularTags()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"tags": tags,
	})
}

func (s *FiberServer) GetArticleMetadataHandler(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Slug is required",
		})
	}

	metadata, err := s.blogService.GetArticleMetadata(slug)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(metadata)
}

func (s *FiberServer) GetArticleDataHandler(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Slug is required",
		})
	}

	data, err := s.blogService.GetArticleData(slug)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Article not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(data)
}

func (s *FiberServer) GetRecommendedArticlesHandler(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid article ID",
		})
	}

	articles, err := s.blogService.GetRecommendedArticles(int64(id))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(articles)
}

func (s *FiberServer) GetDashboardArticlesHandler(c *fiber.Ctx) error {
	articles, err := s.blogService.GetDashboardArticles()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(articles)
}

func (s *FiberServer) DeleteArticleHandler(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid article ID",
		})
	}

	if err := s.blogService.DeleteArticle(int64(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}

func (s *FiberServer) GetAboutPageHandler(c *fiber.Ctx) error {
	page, err := s.pagesService.GetAboutPage()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if page == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "About page not found",
		})
	}

	return c.JSON(page)
}

func (s *FiberServer) GetContactPageHandler(c *fiber.Ctx) error {
	page, err := s.pagesService.GetContactPage()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if page == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Contact page not found",
		})
	}

	return c.JSON(page)
}
