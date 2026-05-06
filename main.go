package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/sessions"
)

// Post represents a blog post
type Post struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Author    string `json:"author"`
	Category  string `json:"category"`
	Tags      string `json:"tags"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// PageData for template rendering
type PageData struct {
	Title           string
	Name            string
	Posts           []Post
	Post            *Post
	User            *User
	Comments        []Comment
	AllComments     []Comment
	Error           string
	Success         string
	IsAdmin         bool
	Query           string
	Category        string
	Sort            string
	PendingCount    int
	ApprovedCount   int
	RejectedCount   int
	PendingComments int
	Slides          []Carousel
	TotalCount      int
	ActiveCount     int
	InactiveCount   int
	Slide           *Carousel
	ActiveSlides    int
	TotalSlides     int
	Users           []User
	TotalUsers      int
	AdminCount      int
	UserCount       int
	TotalPosts      int
	TotalComments   int
	DBHost          string
	DBPort          int
	DBName          string
	CSRFToken       string
	Flash           FlashMessage
}

// Global repository
var postRepo *PostRepository
var carouselRepo *CarouselRepository
var Sess *sessions.Sessions

// renderView is a helper that ensures CSRFToken and Flash are always set in PageData
func renderView(ctx iris.Context, name string, data PageData) error {
	if data.CSRFToken == "" {
		data.CSRFToken = ctx.Values().GetStringDefault("csrfToken", "")
	}
	if data.Flash.Type == "" {
		flash := ctx.Values().Get("flash")
		if f, ok := flash.(FlashMessage); ok {
			data.Flash = f
		}
	}
	return ctx.View(name, data)
}

func main() {
	// Initialize configuration
	InitConfig()

	// Initialize database
	if err := InitDB(); err != nil {
		log.Printf("Warning: Database connection failed: %v", err)
		log.Println("Falling back to in-memory storage...")
	} else {
		postRepo = NewPostRepository(GetDB())
		carouselRepo = NewCarouselRepository(GetDB())
		defer CloseDB()
	}

	app := iris.New()

	// Session configuration
	Sess = sessions.New(sessions.Config{
		Cookie:     "blogsession",
		Expires:    24 * time.Hour,
			})

	// View configuration
	tmpl := iris.HTML("./views", ".html")
	tmpl.AddFunc("split", strings.Split)
	tmpl.AddFunc("csrfField", func(token string) string {
		return `<input type="hidden" name="csrf_token" value="` + token + `">`
	})
	tmpl.Reload(true); app.RegisterView(tmpl)

	// Static files
	app.HandleDir("/static", "./public")

	// Middleware
	app.Use(func(ctx iris.Context) {
		ctx.Values().Set("version", "2.1")

		// Set CSRF token for all requests - single source of truth
		token := SetCSRFToken(ctx)
		ctx.Values().Set("csrfToken", token)

		// Get flash message
		flash := GetFlash(ctx)
		ctx.Values().Set("flash", flash)

		ctx.Next()
	})

	// Apply CSRF protection to POST/PUT/DELETE
	app.Use(CSRFMiddleware())

	// ========================================
	// Frontend Routes
	// ========================================

	// Home page with search
	app.Get("/", func(ctx iris.Context) {
		query := ctx.URLParamDefault("q", "")
		category := ctx.URLParamDefault("category", "")
		sort := ctx.URLParamDefault("sort", "newest")

		var posts []Post
		var err error

		if postRepo != nil {
			if query != "" || category != "" {
				posts, err = postRepo.Search(query, category, sort)
			} else {
				posts, err = postRepo.GetAll()
			}
		}

		if err != nil {
			log.Printf("[ERROR] Get posts failed: %v", err)
			posts = []Post{}
		}

		session := Sess.Start(ctx)
		var user *User
		if userID := session.GetInt64Default(SessionUserID, 0); userID > 0 {
			user, _ = GetUserByID(userID)
		}

		// Get carousel slides
		var slides []Carousel
		if carouselRepo != nil {
			slides, _ = carouselRepo.GetActive()
		}

		renderView(ctx, "home/index.html", PageData{
			Title:    "My Blog - Home",
			Name:     "Visitor",
			Posts:    posts,
			User:     user,
			Query:    query,
			Category: category,
			Sort:     sort,
			Slides:   slides,
		})
	})

	// About page
	app.Get("/about", func(ctx iris.Context) {
		renderView(ctx, "home/about.html", PageData{
			Title: "About - My Blog",
		})
	})

	// Single post page with comments
	app.Get("/post/{id:int}", func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		post, err := postRepo.GetByID(id)
		if err != nil {
			ctx.StatusCode(404)
			ctx.ViewData("Error", ErrMsgNotFound)
			renderView(ctx, "error.html", PageData{})
			return
		}

		comments, _ := GetCommentsByPostID(id)

		session := Sess.Start(ctx)
		var user *User
		if userID := session.GetInt64Default(SessionUserID, 0); userID > 0 {
			user, _ = GetUserByID(userID)
		}

		renderView(ctx, "home/post.html", PageData{
			Title:    post.Title,
			Post:     post,
			Comments: comments,
			User:     user,
		})
	})

	// Add comment (requires login)
	app.Post("/post/{id:int}/comment", func(ctx iris.Context) {
		session := Sess.Start(ctx)
		userID := session.GetInt64Default(SessionUserID, 0)
		if userID == 0 {
			SetFlash(ctx, "error", ErrMsgUnauthorized)
			ctx.Redirect("/login")
			return
		}

		postID := ctx.Params().GetInt64Default("id", 0)
		content := strings.TrimSpace(ctx.FormValue("content"))

		if content == "" {
			SetFlash(ctx, "error", ErrMsgInvalidInput)
			ctx.Redirect("/post/" + ctx.Params().Get("id"))
			return
		}

		_, err := CreateComment(postID, userID, content)
		if err != nil {
			log.Printf("[ERROR] Create comment failed: %v", err)
			SetFlash(ctx, "error", ErrMsgSystemError)
		} else {
			SetFlash(ctx, "success", MsgCommentPending)
		}

		ctx.Redirect("/post/" + ctx.Params().Get("id"))
	})

	// ========================================
	// Auth Routes
	// ========================================

	// Login page
	app.Get("/login", func(ctx iris.Context) {
		session := Sess.Start(ctx)
		if userID := session.GetInt64Default(SessionUserID, 0); userID > 0 {
			ctx.Redirect("/")
			return
		}

		err := renderView(ctx, "auth/login.html", PageData{
			Title: "Login",
		})
		if err != nil {
			log.Printf("[ERROR] Template render error (login): %v", err)
		}
	})

	// Login handler
	app.Post("/login", func(ctx iris.Context) {
		username := ctx.FormValue("username")
		password := ctx.FormValue("password")

		user, err := GetUserByUsername(username)
		if err != nil || !CheckPassword(password, user.Password) {
			renderView(ctx, "auth/login.html", PageData{
				Title: "Login",
				Error: ErrMsgInvalidCredentials,
			})
			return
		}

		// Regenerate session on login to prevent session fixation
		session := Sess.Start(ctx)
		session.Destroy()
		session = Sess.Start(ctx)

		session.Set(SessionUserID, user.ID)
		session.Set(SessionUsername, user.Username)

		SetFlash(ctx, "success", MsgLoginSuccess)
		ctx.Redirect("/")
	})

	// Register page
	app.Get("/register", func(ctx iris.Context) {
		session := Sess.Start(ctx)
		if userID := session.GetInt64Default(SessionUserID, 0); userID > 0 {
			ctx.Redirect("/")
			return
		}

		renderView(ctx, "auth/register.html", PageData{
			Title: "Register",
		})
	})

	// Register handler
	app.Post("/register", func(ctx iris.Context) {
		username := strings.TrimSpace(ctx.FormValue("username"))
		email := strings.TrimSpace(ctx.FormValue("email"))
		password := ctx.FormValue("password")
		confirm := ctx.FormValue("confirm")

		if password != confirm {
			renderView(ctx, "auth/register.html", PageData{
				Title: "Register",
				Error: ErrMsgPasswordMismatch,
			})
			return
		}

		user, err := CreateUser(username, email, password)
		if err != nil {
			errMsg := ErrMsgSystemError
			if err == ErrUserExists {
				errMsg = ErrMsgUserExists
			}
			renderView(ctx, "auth/register.html", PageData{
				Title: "Register",
				Error: errMsg,
			})
			return
		}

		// Regenerate session on register
		session := Sess.Start(ctx)
		session.Destroy()
		session = Sess.Start(ctx)

		session.Set(SessionUserID, user.ID)
		session.Set(SessionUsername, user.Username)

		SetFlash(ctx, "success", MsgRegisterSuccess)
		ctx.Redirect("/")
	})

	// Logout
	app.Get("/logout", func(ctx iris.Context) {
		session := Sess.Start(ctx)
		session.Destroy()
		ctx.Redirect("/")
	})

	// ========================================
	// API Routes
	// ========================================

	app.Get("/api/posts", func(ctx iris.Context) {
		posts, err := postRepo.GetAll()
		if err != nil {
			ctx.StatusCode(500)
			ctx.JSON(map[string]string{"error": err.Error()})
			return
		}
		ctx.JSON(posts)
	})

	// ========================================
	// Admin Routes
	// ========================================

	// Admin login page
	app.Get("/admin/login", func(ctx iris.Context) {
		session := Sess.Start(ctx)
		if session.GetBooleanDefault(SessionIsAdmin, false) {
			ctx.Redirect("/admin")
			return
		}

		renderView(ctx, "admin/login.html", PageData{
			Title: "Admin Login",
		})
	})

	// Admin login handler
	app.Post("/admin/login", func(ctx iris.Context) {
		username := ctx.FormValue("username")
		password := ctx.FormValue("password")

		// 优先：用户名+密码方式（users 表中 role=admin 的用户用自己的账号登录）
		if username != "" {
			user, err := GetUserByUsername(username)
			if err == nil && user.Role == RoleAdmin && CheckPassword(password, user.Password) {
				session := Sess.Start(ctx)
				session.Set(SessionIsAdmin, true)
				session.Set("admin_username", username)
				ctx.Redirect("/admin")
				return
			}
			// 用户名存在但密码不对 → 也检查全局后台密码（支持用全局密码登录）
			if username != "" {
				storedPassword, err := GetAdminPassword()
				if err == nil && CheckPassword(password, storedPassword) {
					session := Sess.Start(ctx)
					session.Set(SessionIsAdmin, true)
					session.Set("admin_username", username)
					ctx.Redirect("/admin")
					return
				}
			}
		} else {
			// 无用户名：只用全局后台密码验证（兼容旧版）
			storedPassword, err := GetAdminPassword()
			if err == nil && CheckPassword(password, storedPassword) {
				session := Sess.Start(ctx)
				session.Set(SessionIsAdmin, true)
				session.Set("admin_username", "admin")
				ctx.Redirect("/admin")
				return
			}
		}

		renderView(ctx, "admin/login.html", PageData{
			Title: "Admin Login",
			Error: ErrMsgInvalidCredentials,
		})
	})

	// Admin logout
	app.Get("/admin/logout", func(ctx iris.Context) {
		session := Sess.Start(ctx)
		session.Delete(SessionIsAdmin)
		ctx.Redirect("/admin/login")
	})

	// Admin middleware - check authentication
	adminAuth := func(ctx iris.Context) {
		session := Sess.Start(ctx)
		if !session.GetBooleanDefault(SessionIsAdmin, false) {
			ctx.Redirect("/admin/login")
			return
		}
		ctx.Next()
	}

	// Admin dashboard (protected)
	app.Get("/admin", adminAuth, func(ctx iris.Context) {
		posts, err := postRepo.GetAll()
		if err != nil {
			log.Printf("[ERROR] Get posts failed: %v", err)
			posts = []Post{}
		}

		// Get statistics
		pendingComments, _ := CountPendingComments()
		totalUsers, _ := CountUsers()
		adminCount, _ := CountUsersByRole(RoleAdmin)
		userCount, _ := CountUsersByRole(RoleUser)

		var activeSlides int
		if carouselRepo != nil {
			slides, _ := carouselRepo.GetActive()
			activeSlides = len(slides)
		}

		renderView(ctx, "admin/index.html", PageData{
			Title:           "Admin Dashboard",
			Posts:           posts,
			PendingComments: pendingComments,
			TotalUsers:      totalUsers,
			AdminCount:      adminCount,
			UserCount:       userCount,
			ActiveSlides:    activeSlides,
		})
	})

	// New post form (protected)
	app.Get("/admin/new", adminAuth, func(ctx iris.Context) {
		renderView(ctx, "admin/new.html", PageData{
			Title: "New Post",
		})
	})

	// Create post (protected)
	app.Post("/admin", adminAuth, func(ctx iris.Context) {
		title := strings.TrimSpace(ctx.FormValue("title"))
		content := ctx.FormValue("content")
		author := strings.TrimSpace(ctx.FormValue("author"))
		category := strings.TrimSpace(ctx.FormValue("category"))
		tags := ctx.FormValue("tags")

		if title == "" {
			title = "Untitled"
		}
		if author == "" {
			author = "Admin"
		}
		if category == "" {
			category = "Tech"
		}

		_, err := postRepo.Create(&Post{
			Title:    title,
			Content:  content,
			Author:   author,
			Category: category,
			Tags:     tags,
		})
		if err != nil {
			log.Printf("[ERROR] Create post failed: %v", err)
			SetFlash(ctx, "error", ErrMsgSystemError)
		} else {
			SetFlash(ctx, "success", MsgPostCreated)
		}

		ctx.Redirect("/admin")
	})

	// Edit form (protected)
	app.Get("/admin/edit/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		post, err := postRepo.GetByID(id)
		if err != nil {
			ctx.StatusCode(404)
			ctx.ViewData("Error", ErrMsgNotFound)
			renderView(ctx, "error.html", PageData{})
			return
		}
		renderView(ctx, "admin/edit.html", PageData{
			Title: "Edit Post",
			Post:  post,
		})
	})

	// Update post (protected)
	app.Post("/admin/edit/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		title := strings.TrimSpace(ctx.FormValue("title"))
		content := ctx.FormValue("content")
		author := strings.TrimSpace(ctx.FormValue("author"))
		category := strings.TrimSpace(ctx.FormValue("category"))
		tags := ctx.FormValue("tags")

		err := postRepo.Update(&Post{
			ID:       id,
			Title:    title,
			Content:  content,
			Author:   author,
			Category: category,
			Tags:     tags,
		})
		if err != nil {
			log.Printf("[ERROR] Update post failed: %v", err)
			SetFlash(ctx, "error", ErrMsgSystemError)
		} else {
			SetFlash(ctx, "success", MsgPostUpdated)
		}

		ctx.Redirect("/admin")
	})

	// Delete post (protected)
	app.Post("/admin/delete/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		err := postRepo.Delete(id)
		if err != nil {
			log.Printf("[ERROR] Delete post failed: %v", err)
			SetFlash(ctx, "error", ErrMsgSystemError)
		} else {
			SetFlash(ctx, "success", MsgPostDeleted)
		}
		ctx.Redirect("/admin")
	})

	// ========================================
	// Comment Management (Admin only)
	// ========================================

	// View all comments
	app.Get("/admin/comments", adminAuth, func(ctx iris.Context) {
		comments, err := GetAllComments()
		if err != nil {
			log.Printf("[ERROR] Get comments failed: %v", err)
			comments = []Comment{}
		}

		pendingCount := 0
		approvedCount := 0
		rejectedCount := 0
		for _, c := range comments {
			switch c.Status {
			case StatusPending:
				pendingCount++
			case StatusApproved:
				approvedCount++
			case StatusRejected:
				rejectedCount++
			}
		}

		pendingComments, _ := CountPendingComments()

		renderView(ctx, "admin/comments.html", PageData{
			Title:           "Comments Management",
			AllComments:     comments,
			PendingCount:    pendingCount,
			ApprovedCount:   approvedCount,
			RejectedCount:   rejectedCount,
			PendingComments: pendingComments,
		})
	})

	// Approve comment
	app.Post("/admin/comment/approve/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		ApproveComment(id)
		ctx.Redirect("/admin/comments")
	})

	// Reject comment
	app.Post("/admin/comment/reject/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		RejectComment(id)
		ctx.Redirect("/admin/comments")
	})

	// Delete comment
	app.Post("/admin/comment/delete/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		DeleteComment(id)
		ctx.Redirect("/admin/comments")
	})

	// ========================================
	// Carousel Management Routes
	// ========================================

	// Carousel list
	app.Get("/admin/carousel", adminAuth, func(ctx iris.Context) {
		slides, _ := carouselRepo.GetAll()

		var activeCount, inactiveCount int
		for _, s := range slides {
			if s.IsActive {
				activeCount++
			} else {
				inactiveCount++
			}
		}

		pendingComments, _ := CountPendingComments()

		renderView(ctx, "admin/carousel.html", PageData{
			Title:           "Carousel Management",
			Slides:          slides,
			TotalCount:      len(slides),
			ActiveCount:     activeCount,
			InactiveCount:   inactiveCount,
			PendingComments: pendingComments,
		})
	})

	// New carousel form
	app.Get("/admin/carousel/new", adminAuth, func(ctx iris.Context) {
		pendingComments, _ := CountPendingComments()
		renderView(ctx, "admin/carousel_new.html", PageData{
			Title:           "New Carousel",
			PendingComments: pendingComments,
		})
	})

	// Create carousel
	app.Post("/admin/carousel/new", adminAuth, func(ctx iris.Context) {
		slide := &Carousel{
			Title:      ctx.FormValue("title"),
			Subtitle:   ctx.FormValue("subtitle"),
			ButtonText: ctx.FormValue("button_text"),
			ButtonLink: ctx.FormValue("button_link"),
			ImageURL:   ctx.FormValue("image_url"),
			SortOrder:  parseInt(ctx.FormValue("sort_order"), 1),
			IsActive:   ctx.FormValue("is_active") == "on",
		}

		_, err := carouselRepo.Create(slide)
		if err != nil {
			log.Printf("[ERROR] Create carousel failed: %v", err)
			SetFlash(ctx, "error", ErrMsgSystemError)
		}

		ctx.Redirect("/admin/carousel")
	})

	// Edit carousel form
	app.Get("/admin/carousel/edit/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		slide, err := carouselRepo.GetByID(id)
		if err != nil {
			ctx.StatusCode(404)
			ctx.ViewData("Error", ErrMsgNotFound)
			renderView(ctx, "error.html", PageData{})
			return
		}

		pendingComments, _ := CountPendingComments()
		renderView(ctx, "admin/carousel_edit.html", PageData{
			Title:           "Edit Carousel",
			Slide:           slide,
			PendingComments: pendingComments,
		})
	})

	// Update carousel
	app.Post("/admin/carousel/edit/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)

		slide := &Carousel{
			ID:         id,
			Title:      ctx.FormValue("title"),
			Subtitle:   ctx.FormValue("subtitle"),
			ButtonText: ctx.FormValue("button_text"),
			ButtonLink: ctx.FormValue("button_link"),
			ImageURL:   ctx.FormValue("image_url"),
			SortOrder:  parseInt(ctx.FormValue("sort_order"), 1),
			IsActive:   ctx.FormValue("is_active") == "on",
		}

		err := carouselRepo.Update(slide)
		if err != nil {
			log.Printf("[ERROR] Update carousel failed: %v", err)
			SetFlash(ctx, "error", ErrMsgSystemError)
		}

		ctx.Redirect("/admin/carousel")
	})

	// Toggle carousel active status
	app.Post("/admin/carousel/toggle/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		active := ctx.URLParamDefault("active", "true") == "true"

		err := carouselRepo.ToggleActive(id, active)
		if err != nil {
			log.Printf("[ERROR] Toggle carousel failed: %v", err)
		}

		ctx.Redirect("/admin/carousel")
	})

	// Delete carousel
	app.Post("/admin/carousel/delete/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		err := carouselRepo.Delete(id)
		if err != nil {
			log.Printf("[ERROR] Delete carousel failed: %v", err)
		}
		ctx.Redirect("/admin/carousel")
	})

	// ========================================
	// User Management Routes
	// ========================================

	// User list
	app.Get("/admin/users", adminAuth, func(ctx iris.Context) {
		users, _ := GetAllUsers()
		totalUsers, _ := CountUsers()
		adminCount, _ := CountUsersByRole(RoleAdmin)
		userCount, _ := CountUsersByRole(RoleUser)
		pendingComments, _ := CountPendingComments()

		renderView(ctx, "admin/users.html", PageData{
			Title:           "User Management",
			Users:           users,
			TotalUsers:      totalUsers,
			AdminCount:      adminCount,
			UserCount:       userCount,
			PendingComments: pendingComments,
		})
	})

	// Promote user to admin
	app.Post("/admin/user/promote/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		err := UpdateUserRole(id, RoleAdmin)
		if err != nil {
			log.Printf("[ERROR] Promote user failed: %v", err)
		}
		ctx.Redirect("/admin/users")
	})

	// Demote admin to user
	app.Post("/admin/user/demote/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		err := UpdateUserRole(id, RoleUser)
		if err != nil {
			log.Printf("[ERROR] Demote user failed: %v", err)
		}
		ctx.Redirect("/admin/users")
	})

	// Delete user
	app.Post("/admin/user/delete/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		err := DeleteUser(id)
		if err != nil {
			log.Printf("[ERROR] Delete user failed: %v", err)
		}
		ctx.Redirect("/admin/users")
	})

	// ========================================
	// Settings Routes
	// ========================================

	// Settings page
	app.Get("/admin/settings", adminAuth, func(ctx iris.Context) {
		totalPosts, _ := postRepo.Count()
		totalUsers, _ := CountUsers()
		var totalComments int
		DB.QueryRow("SELECT COUNT(*) FROM comments").Scan(&totalComments)
		pendingComments, _ := CountPendingComments()

		var totalSlides, activeSlides int
		if carouselRepo != nil {
			allSlides, _ := carouselRepo.GetAll()
			totalSlides = len(allSlides)
			activeSlidesList, _ := carouselRepo.GetActive()
			activeSlides = len(activeSlidesList)
		}

		renderView(ctx, "admin/settings.html", PageData{
			Title:           "Settings",
			TotalPosts:      totalPosts,
			TotalUsers:      totalUsers,
			TotalComments:   totalComments,
			PendingComments: pendingComments,
			TotalSlides:     totalSlides,
			ActiveSlides:    activeSlides,
			DBHost:          AppConfig.DBHost,
			DBPort:          AppConfig.DBPort,
			DBName:          AppConfig.DBName,
		})
	})

	// Update admin password
	app.Post("/admin/settings/password", adminAuth, func(ctx iris.Context) {
		currentPassword := ctx.FormValue("current_password")
		newPassword := ctx.FormValue("new_password")
		confirmPassword := ctx.FormValue("confirm_password")

		storedPassword, err := GetAdminPassword()
		if err != nil {
			SetFlash(ctx, "error", ErrMsgSystemError)
			ctx.Redirect("/admin/settings")
			return
		}

		if !CheckPassword(currentPassword, storedPassword) {
			SetFlash(ctx, "error", ErrMsgInvalidCredentials)
			ctx.Redirect("/admin/settings")
			return
		}

		if newPassword != confirmPassword {
			SetFlash(ctx, "error", ErrMsgPasswordMismatch)
			ctx.Redirect("/admin/settings")
			return
		}

		if len(newPassword) < 6 {
			SetFlash(ctx, "error", "Password must be at least 6 characters")
			ctx.Redirect("/admin/settings")
			return
		}

		hashedPassword, err := HashPassword(newPassword)
		if err != nil {
			SetFlash(ctx, "error", ErrMsgSystemError)
			ctx.Redirect("/admin/settings")
			return
		}

		err = UpdateAdminPassword(hashedPassword)
		if err != nil {
			log.Printf("[ERROR] Update password failed: %v", err)
			SetFlash(ctx, "error", ErrMsgSystemError)
		} else {
			SetFlash(ctx, "success", MsgPasswordChanged)
		}

		ctx.Redirect("/admin/settings")
	})

	// Start server
	log.Println("========================================")
	log.Println("  Blog Server Started (v2.1)")
	log.Printf("  Frontend: http://localhost:%s", AppConfig.ServerPort)
	log.Printf("  Admin:    http://localhost:%s/admin", AppConfig.ServerPort)
	log.Printf("  API:      http://localhost:%s/api/posts", AppConfig.ServerPort)
	log.Println("========================================")

	if err := app.Listen(":" + AppConfig.ServerPort); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// Helper functions
func parseInt(s string, defaultValue int) int {
	var result int
	if _, err := fmt.Sscanf(s, "%d", &result); err != nil {
		return defaultValue
	}
	return result
}
