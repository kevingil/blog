# Backend Styling Guide

## 1. Directory Structure

Adhere strictly to this layout:

```
backend/
├── pkg/
│   ├── api/
│   │   ├── dto/            # Request/response types for API
│   │   └── <domain>/
│   │       ├── handlers.go  # Route handlers: parse request, call service, return response
│   │       └── routes.go    # Route definitions, middleware assignment
│   ├── core/
│   │   └── <domain>/
│   │       ├── service.go   # Business logic service with dependency injection
│   │       └── <domain>.go  # Domain type aliases (optional)
│   ├── database/
│   │   ├── models/          # GORM model definitions (database schema)
│   │   ├── repository/      # Repository interfaces AND implementations
│   │   └── database.go      # Connection setup, singleton accessor
│   └── types/               # Shared domain types (GORM-agnostic)
├── testutil/
│   └── mocks/               # Mock implementations for testing (MockXxxRepository)
└── main.go                  # Entry point & server initialization
```

## 2. Layer Responsibilities

### Handlers (pkg/api/<domain>/handlers.go)

Use Fiber handlers. Functions should:
1. Extract data from `*fiber.Ctx` (params, body, query)
2. Validate request using `validation.ValidateStruct()`
3. Get service via `getService()` singleton pattern
4. Call service methods
5. Return response using `response.Success()`, `response.Created()`, or `response.Error()`

```go
var (
    serviceInstance *article.Service
    serviceOnce     sync.Once
)

func getService() *article.Service {
    serviceOnce.Do(func() {
        db := database.DB()
        articleRepo := repository.NewArticleRepository(db)
        accountRepo := repository.NewAccountRepository(db)
        tagRepo := repository.NewTagRepository(db)
        serviceInstance = article.NewService(articleRepo, accountRepo, tagRepo)
    })
    return serviceInstance
}

func CreateArticle(c *fiber.Ctx) error {
    var req dto.CreateArticleRequest
    if err := c.BodyParser(&req); err != nil {
        return response.Error(c, core.InvalidInputError("Invalid request body"))
    }
    if err := validation.ValidateStruct(req); err != nil {
        return response.Error(c, err)
    }

    svc := getService()
    result, err := svc.Create(c.Context(), req)
    if err != nil {
        return response.Error(c, err)
    }
    return response.Created(c, result)
}
```

### Services (pkg/core/<domain>/service.go)

Act as the "Traffic Controller." Services:
- Are struct-based with injected repository interfaces
- Contain framework-agnostic business logic (no Fiber imports)
- Call repository methods for data access
- Return domain errors from `pkg/core/errors.go`

```go
type Service struct {
    articleRepo repository.ArticleRepository
    accountRepo repository.AccountRepository
    tagRepo     repository.TagRepository
}

func NewService(
    articleRepo repository.ArticleRepository,
    accountRepo repository.AccountRepository,
    tagRepo repository.TagRepository,
) *Service {
    return &Service{
        articleRepo: articleRepo,
        accountRepo: accountRepo,
        tagRepo:     tagRepo,
    }
}

func (s *Service) Create(ctx context.Context, req dto.CreateRequest) (*types.Article, error) {
    article := &types.Article{
        Title: req.Title,
        Slug:  generateSlug(req.Title),
    }
    if err := s.articleRepo.Save(ctx, article); err != nil {
        return nil, err
    }
    return article, nil
}
```

### Repositories (pkg/database/repository/)

Repository interfaces AND implementations live together. Repositories:
- Define the interface (exported)
- Implement with unexported struct
- Return interface from constructor
- Convert between models and types
- Handle database-specific error mapping

```go
// Interface definition
type ArticleRepository interface {
    FindByID(ctx context.Context, id uuid.UUID) (*types.Article, error)
    FindBySlug(ctx context.Context, slug string) (*types.Article, error)
    Save(ctx context.Context, article *types.Article) error
    Update(ctx context.Context, article *types.Article) error
    Delete(ctx context.Context, id uuid.UUID) error
    List(ctx context.Context, opts types.ArticleListOptions) ([]types.Article, int64, error)
}

// Unexported implementation
type articleRepository struct {
    db *gorm.DB
}

// Constructor returns interface
func NewArticleRepository(db *gorm.DB) ArticleRepository {
    return &articleRepository{db: db}
}

func (r *articleRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.Article, error) {
    var model models.Article
    if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, core.ErrNotFound
        }
        return nil, err
    }
    return articleModelToType(&model), nil
}
```

## 3. Implementation Standards

### Dependency Injection

Pass repository interfaces into the service via constructor:

```go
// In core/<domain>/service.go
func NewService(repo repository.ArticleRepository) *Service {
    return &Service{repo: repo}
}

// In api/<domain>/handlers.go - use singleton pattern
func getService() *article.Service {
    serviceOnce.Do(func() {
        repo := repository.NewArticleRepository(database.DB())
        serviceInstance = article.NewService(repo)
    })
    return serviceInstance
}
```

### Error Handling

Use domain errors from `pkg/core/errors.go`:

```go
// In service
if article == nil {
    return nil, core.ErrNotFound
}

// In handler - automatically maps to HTTP status
return response.Error(c, err)
```

### Validation

Use struct tags for validation in request DTOs:

```go
type CreateArticleRequest struct {
    Title   string   `json:"title" validate:"required,min=1,max=200"`
    Content string   `json:"content" validate:"required"`
    Tags    []string `json:"tags" validate:"max=10,dive,max=50"`
}
```

### Testing

Use `testify` with mock implementations:

```go
func TestService_Create(t *testing.T) {
    ctx := context.Background()
    mockRepo := new(mocks.MockArticleRepository)
    svc := article.NewService(mockRepo)

    t.Run("creates article successfully", func(t *testing.T) {
        mockRepo.On("Save", ctx, mock.AnythingOfType("*types.Article")).Return(nil).Once()

        result, err := svc.Create(ctx, dto.CreateRequest{Title: "Test"})

        assert.NoError(t, err)
        assert.Equal(t, "Test", result.Title)
        mockRepo.AssertExpectations(t)
    })
}
```

### Reference Implementation

The `pkg/core/article/` package is the canonical example of this architecture:
- `service.go` - Struct-based service with repository DI
- `pkg/database/repository/article.go` - Interface + implementation
- `testutil/mocks/article_repository.go` - Mock implementation

## 4. Common Patterns

### List with Pagination

```go
type ListOptions struct {
    Page    int
    PerPage int
    Search  string
}

func (s *Service) List(ctx context.Context, opts ListOptions) ([]types.Article, int64, error) {
    return s.repo.List(ctx, opts)
}
```

### Model to Type Conversion

Keep conversion functions in repositories:

```go
func articleModelToType(m *models.Article) *types.Article {
    return &types.Article{
        ID:        m.ID,
        Title:     m.Title,
        CreatedAt: m.CreatedAt,
    }
}

func articleTypeToModel(a *types.Article) *models.Article {
    return &models.Article{
        ID:        a.ID,
        Title:     a.Title,
    }
}
```

## 5. What NOT to Do

- **Don't** call `database.DB()` directly in services - use injected repository
- **Don't** import Fiber in core packages - keep business logic framework-agnostic
- **Don't** define repository interfaces in core packages - define in repository package
- **Don't** duplicate interface definitions - one interface per repository
- **Don't** skip validation in mutating handlers
- **Don't** return raw GORM errors - map to domain errors
- **Don't** use pointer to interface (`*Repository`) - use interface directly
