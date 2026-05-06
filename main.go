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
	Title        string
	Name         string
	Posts        []Post
	Post         *Post
	User         *User
	Comments     []Comment
	AllComments  []Comment // For admin
	Error        string
	Success      string
	IsAdmin      bool
	Query        string   // Search query
	Category     string   // Search category
	Sort         string   // Sort order
	PendingCount    int
	ApprovedCount   int
	RejectedCount   int
	PendingComments int // For admin header badge
	// Carousel
	Slides      []Carousel
	TotalCount    int
	ActiveCount   int
	InactiveCount int
	Slide         *Carousel
	ActiveSlides  int // For dashboard
	TotalSlides   int // For settings
	// Users
	Users       []User
	TotalUsers  int
	AdminCount  int
	UserCount   int
	// Settings
	TotalPosts    int
	TotalComments int
	DBHost        string
	DBPort        int
	DBName        string
}

// Global repository
var postRepo *PostRepository
var carouselRepo *CarouselRepository
var sess *sessions.Sessions

func main() {
	// Initialize database
	config := DefaultConfig()
	if err := InitDB(config); err != nil {
		log.Printf("Warning: Database connection failed: %v", err)
		log.Println("Falling back to in-memory storage...")
	} else {
		postRepo = NewPostRepository(GetDB())
		carouselRepo = NewCarouselRepository(GetDB())
		defer CloseDB()
	}

	app := iris.New()

	// Session configuration
	sess = sessions.New(sessions.Config{
		Cookie:  "blogsession",
		Expires: 24 * time.Hour,
	})

	// View configuration
	tmpl := iris.HTML("./views", ".html")
	tmpl.AddFunc("split", strings.Split)
	app.RegisterView(tmpl)

	// Static files
	app.HandleDir("/static", "./public")

	// Middleware
	app.Use(func(ctx iris.Context) {
		ctx.Values().Set("version", "2.0")
		ctx.Next()
	})

	// ============================================================
	// Frontend Routes
	// ============================================================

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
			ctx.StatusCode(500)
			ctx.WriteString("Database error: " + err.Error())
			return
		}

		session := sess.Start(ctx)
		var user *User
		if userID := session.GetInt64Default("userID", 0); userID > 0 {
			user, _ = GetUserByID(userID)
		}

		// Get carousel slides
		var slides []Carousel
		if carouselRepo != nil {
			slides, _ = carouselRepo.GetActive()
		}

		ctx.View("home/index.html", PageData{
			Title:    "我的博客 - 首页",
			Name:     "访客",
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
		ctx.View("home/about.html", PageData{
			Title: "关于 - 我的博客",
		})
	})

	// Single post page with comments
	app.Get("/post/{id:int}", func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		post, err := getPostByID(id)
		if err != nil {
			ctx.StatusCode(404)
			ctx.WriteString("Post not found")
			return
		}

		comments, _ := GetCommentsByPostID(id)

		session := sess.Start(ctx)
		var user *User
		if userID := session.GetInt64Default("userID", 0); userID > 0 {
			user, _ = GetUserByID(userID)
		}

		ctx.View("home/post.html", PageData{
			Title:    post.Title,
			Post:     post,
			Comments: comments,
			User:     user,
		})
	})

	// Add comment (requires login)
	app.Post("/post/{id:int}/comment", func(ctx iris.Context) {
		session := sess.Start(ctx)
		userID := session.GetInt64Default("userID", 0)
		if userID == 0 {
			ctx.Redirect("/login")
			return
		}

		postID := ctx.Params().GetInt64Default("id", 0)
		content := ctx.FormValue("content")

		if content != "" {
			_, err := CreateComment(postID, userID, content)
			if err != nil {
				log.Printf("[ERROR] Create comment failed: %v", err)
			}
		}

		ctx.Redirect("/post/" + ctx.Params().Get("id"))
	})

	// ============================================================
	// Auth Routes
	// ============================================================

	// Login page
	app.Get("/login", func(ctx iris.Context) {
		session := sess.Start(ctx)
		if userID := session.GetInt64Default("userID", 0); userID > 0 {
			ctx.Redirect("/")
			return
		}

		ctx.View("auth/login.html", PageData{
			Title: "登录",
		})
	})

	// Login handler
	app.Post("/login", func(ctx iris.Context) {
		username := ctx.FormValue("username")
		password := ctx.FormValue("password")

		user, err := GetUserByUsername(username)
		if err != nil || !CheckPassword(password, user.Password) {
			ctx.View("auth/login.html", PageData{
				Title: "登录",
				Error: "用户名或密码错误",
			})
			return
		}

		session := sess.Start(ctx)
		session.Set("userID", user.ID)
		session.Set("username", user.Username)

		ctx.Redirect("/")
	})

	// Register page
	app.Get("/register", func(ctx iris.Context) {
		session := sess.Start(ctx)
		if userID := session.GetInt64Default("userID", 0); userID > 0 {
			ctx.Redirect("/")
			return
		}

		ctx.View("auth/register.html", PageData{
			Title: "注册",
		})
	})

	// Register handler
	app.Post("/register", func(ctx iris.Context) {
		username := ctx.FormValue("username")
		email := ctx.FormValue("email")
		password := ctx.FormValue("password")
		confirm := ctx.FormValue("confirm")

		if password != confirm {
			ctx.View("auth/register.html", PageData{
				Title: "注册",
				Error: "两次输入的密码不一致",
			})
			return
		}

		user, err := CreateUser(username, email, password)
		if err != nil {
			ctx.View("auth/register.html", PageData{
				Title: "注册",
				Error: "用户名或邮箱已被注册",
			})
			return
		}

		session := sess.Start(ctx)
		session.Set("userID", user.ID)
		session.Set("username", user.Username)

		ctx.Redirect("/")
	})

	// Logout
	app.Get("/logout", func(ctx iris.Context) {
		session := sess.Start(ctx)
		session.Delete("userID")
		session.Delete("username")
		ctx.Redirect("/")
	})

	// ============================================================
	// API Routes
	// ============================================================

	app.Get("/api/posts", func(ctx iris.Context) {
		posts, err := getPosts()
		if err != nil {
			ctx.StatusCode(500)
			ctx.JSON(map[string]string{"error": err.Error()})
			return
		}
		ctx.JSON(posts)
	})

	// ============================================================
	// Admin Routes
	// ============================================================

	// Admin login page
	app.Get("/admin/login", func(ctx iris.Context) {
		session := sess.Start(ctx)
		if session.GetBooleanDefault("isAdmin", false) {
			ctx.Redirect("/admin")
			return
		}
		ctx.View("admin/login.html", PageData{
			Title: "管理员登录",
		})
	})

	// Admin login handler
	app.Post("/admin/login", func(ctx iris.Context) {
		password := ctx.FormValue("password")
		
		// Get stored admin password
		storedPassword, err := GetAdminPassword()
		if err != nil {
			ctx.View("admin/login.html", PageData{
				Title: "管理员登录",
				Error: "系统错误，请稍后重试",
			})
			return
		}
		
		if CheckPassword(password, storedPassword) {
			session := sess.Start(ctx)
			session.Set("isAdmin", true)
			ctx.Redirect("/admin")
			return
		}
		ctx.View("admin/login.html", PageData{
			Title: "管理员登录",
			Error: "密码错误",
		})
	})

	// Admin logout
	app.Get("/admin/logout", func(ctx iris.Context) {
		session := sess.Start(ctx)
		session.Delete("isAdmin")
		ctx.Redirect("/admin/login")
	})

	// Admin middleware - check password
	adminAuth := func(ctx iris.Context) {
		session := sess.Start(ctx)
		if !session.GetBooleanDefault("isAdmin", false) {
			ctx.Redirect("/admin/login")
			return
		}
		ctx.Next()
	}

	// Admin dashboard (protected)
	app.Get("/admin", adminAuth, func(ctx iris.Context) {
		posts, err := getPosts()
		if err != nil {
			ctx.ViewData("Title", "Admin Dashboard")
			ctx.ViewData("Posts", []Post{})
			ctx.View("admin/index.html")
			return
		}

		// Get statistics
		pendingComments, _ := CountPendingComments()
		totalUsers, _ := CountUsers()
		adminCount, _ := CountUsersByRole("admin")
		userCount, _ := CountUsersByRole("user")
		
		// Get active slides count
		var activeSlides int
		if carouselRepo != nil {
			slides, _ := carouselRepo.GetActive()
			activeSlides = len(slides)
		}

		log.Printf("[DEBUG] Admin: rendering %d posts", len(posts))
		ctx.View("admin/index.html", PageData{
			Title:           "管理后台",
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
		ctx.View("admin/new.html", PageData{
			Title: "New Post",
		})
	})

	// Create post (protected)
	app.Post("/admin", adminAuth, func(ctx iris.Context) {
		title := ctx.FormValue("title")
		content := ctx.FormValue("content")
		author := ctx.FormValue("author")
		category := ctx.FormValue("category")
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

		_, err := createPost(title, content, author, category, tags)
		if err != nil {
			log.Printf("[ERROR] Create post failed: %v", err)
		}

		ctx.Redirect("/admin")
	})

	// Edit form (protected)
	app.Get("/admin/edit/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		log.Printf("[DEBUG] Edit form for id=%d", id)
		post, err := getPostByID(id)
		if err != nil {
			ctx.StatusCode(404)
			ctx.WriteString("Post not found")
			return
		}
		ctx.View("admin/edit.html", PageData{
			Title: "Edit Post",
			Post:  post,
		})
	})

	// Update post (protected)
	app.Post("/admin/edit/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		title := ctx.FormValue("title")
		content := ctx.FormValue("content")
		author := ctx.FormValue("author")
		category := ctx.FormValue("category")
		tags := ctx.FormValue("tags")

		updatePost(id, title, content, author, category, tags)
		ctx.Redirect("/admin")
	})

	// Delete post (protected)
	app.Get("/admin/delete/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		deletePost(id)
		ctx.Redirect("/admin")
	})

	// ============================================================
	// Comment Management (Admin only)
	// ============================================================

	// View all comments
	app.Get("/admin/comments", adminAuth, func(ctx iris.Context) {
		comments, err := GetAllComments()
		if err != nil {
			log.Printf("[ERROR] Get comments failed: %v", err)
		}

		// Count by status
		pendingCount := 0
		approvedCount := 0
		rejectedCount := 0
		for _, c := range comments {
			switch c.Status {
			case "pending":
				pendingCount++
			case "approved":
				approvedCount++
			case "rejected":
				rejectedCount++
			}
		}

		// Get pending count for header badge
		pendingComments, _ := CountPendingComments()

		ctx.View("admin/comments.html", PageData{
			Title:           "评论管理",
			AllComments:     comments,
			PendingCount:    pendingCount,
			ApprovedCount:   approvedCount,
			RejectedCount:   rejectedCount,
			PendingComments: pendingComments,
		})
	})

	// Approve comment
	app.Get("/admin/comment/approve/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		ApproveComment(id)
		ctx.Redirect("/admin/comments")
	})

	// Reject comment
	app.Get("/admin/comment/reject/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		RejectComment(id)
		ctx.Redirect("/admin/comments")
	})

	// Delete comment
	app.Get("/admin/comment/delete/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		DeleteComment(id)
		ctx.Redirect("/admin/comments")
	})

	// ============================================================
	// Carousel Management Routes
	// ============================================================

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

		// Get pending comments count for badge
		pendingComments, _ := CountPendingComments()

		ctx.View("admin/carousel.html", PageData{
			Title:          "轮播图管理",
			Slides:         slides,
			TotalCount:     len(slides),
			ActiveCount:    activeCount,
			InactiveCount:  inactiveCount,
			PendingComments: pendingComments,
		})
	})

	// New carousel form
	app.Get("/admin/carousel/new", adminAuth, func(ctx iris.Context) {
		pendingComments, _ := CountPendingComments()
		ctx.View("admin/carousel_new.html", PageData{
			Title:           "新建轮播图",
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
		}

		ctx.Redirect("/admin/carousel")
	})

	// Edit carousel form
	app.Get("/admin/carousel/edit/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		slide, err := carouselRepo.GetByID(id)
		if err != nil {
			ctx.StatusCode(404)
			ctx.WriteString("Slide not found")
			return
		}

		pendingComments, _ := CountPendingComments()
		ctx.View("admin/carousel_edit.html", PageData{
			Title:           "编辑轮播图",
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
		}

		ctx.Redirect("/admin/carousel")
	})

	// Toggle carousel active status
	app.Get("/admin/carousel/toggle/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		active := ctx.URLParamDefault("active", "true") == "true"
		
		err := carouselRepo.ToggleActive(id, active)
		if err != nil {
			log.Printf("[ERROR] Toggle carousel failed: %v", err)
		}

		ctx.Redirect("/admin/carousel")
	})

	// Delete carousel
	app.Get("/admin/carousel/delete/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		err := carouselRepo.Delete(id)
		if err != nil {
			log.Printf("[ERROR] Delete carousel failed: %v", err)
		}
		ctx.Redirect("/admin/carousel")
	})

	// ============================================================
	// User Management Routes
	// ============================================================

	// User list
	app.Get("/admin/users", adminAuth, func(ctx iris.Context) {
		users, _ := GetAllUsers()
		totalUsers, _ := CountUsers()
		adminCount, _ := CountUsersByRole("admin")
		userCount, _ := CountUsersByRole("user")
		pendingComments, _ := CountPendingComments()

		ctx.View("admin/users.html", PageData{
			Title:           "用户管理",
			Users:           users,
			TotalUsers:      totalUsers,
			AdminCount:      adminCount,
			UserCount:       userCount,
			PendingComments: pendingComments,
		})
	})

	// Promote user to admin
	app.Get("/admin/user/promote/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		err := UpdateUserRole(id, "admin")
		if err != nil {
			log.Printf("[ERROR] Promote user failed: %v", err)
		}
		ctx.Redirect("/admin/users")
	})

	// Demote admin to user
	app.Get("/admin/user/demote/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		err := UpdateUserRole(id, "user")
		if err != nil {
			log.Printf("[ERROR] Demote user failed: %v", err)
		}
		ctx.Redirect("/admin/users")
	})

	// Delete user
	app.Get("/admin/user/delete/{id:int}", adminAuth, func(ctx iris.Context) {
		id := ctx.Params().GetInt64Default("id", 0)
		err := DeleteUser(id)
		if err != nil {
			log.Printf("[ERROR] Delete user failed: %v", err)
		}
		ctx.Redirect("/admin/users")
	})

	// ============================================================
	// Settings Routes
	// ============================================================

	// Settings page
	app.Get("/admin/settings", adminAuth, func(ctx iris.Context) {
		totalPosts, _ := postRepo.Count()
		totalUsers, _ := CountUsers()
		var totalComments int
		DB.QueryRow("SELECT COUNT(*) FROM comments").Scan(&totalComments)
		pendingComments, _ := CountPendingComments()
		
		// Get slides count
		var totalSlides, activeSlides int
		if carouselRepo != nil {
			allSlides, _ := carouselRepo.GetAll()
			totalSlides = len(allSlides)
			activeSlidesList, _ := carouselRepo.GetActive()
			activeSlides = len(activeSlidesList)
		}
		
		config := DefaultConfig()

		ctx.View("admin/settings.html", PageData{
			Title:           "系统设置",
			TotalPosts:      totalPosts,
			TotalUsers:      totalUsers,
			TotalComments:   totalComments,
			PendingComments: pendingComments,
			TotalSlides:     totalSlides,
			ActiveSlides:    activeSlides,
			DBHost:          config.Host,
			DBPort:          config.Port,
			DBName:          config.Database,
		})
	})

	// Update admin password
	app.Post("/admin/settings/password", adminAuth, func(ctx iris.Context) {
		currentPassword := ctx.FormValue("current_password")
		newPassword := ctx.FormValue("new_password")
		confirmPassword := ctx.FormValue("confirm_password")

		// Get stored password
		storedPassword, err := GetAdminPassword()
		if err != nil {
			ctx.Redirect("/admin/settings")
			return
		}

		// Verify current password
		if !CheckPassword(currentPassword, storedPassword) {
			ctx.Redirect("/admin/settings")
			return
		}

		// Check new password confirmation
		if newPassword != confirmPassword {
			ctx.Redirect("/admin/settings")
			return
		}

		// Validate new password length
		if len(newPassword) < 6 {
			ctx.Redirect("/admin/settings")
			return
		}

		// Hash and update password
		hashedPassword, err := HashPassword(newPassword)
		if err != nil {
			ctx.Redirect("/admin/settings")
			return
		}

		err = UpdateAdminPassword(hashedPassword)
		if err != nil {
			log.Printf("[ERROR] Update password failed: %v", err)
		}

		ctx.Redirect("/admin/settings")
	})

	// Start server
	log.Println("========================================")
	log.Println("  Blog Server Started (v2.0)")
	log.Println("  Frontend: http://localhost:8080")
	log.Println("  Admin:    http://localhost:8080/admin")
	log.Println("  API:      http://localhost:8080/api/posts")
	// Debug API to check comments
	app.Get("/api/debug/comments", func(ctx iris.Context) {
		rows, err := DB.Query(`SELECT id, post_id, user_id, content, status, created_at FROM comments ORDER BY id DESC LIMIT 20`)
		if err != nil {
			ctx.JSON(iris.Map{"error": err.Error()})
			return
		}
		defer rows.Close()

		var comments []iris.Map
		for rows.Next() {
			var id, postID, userID int64
			var content, status string
			var createdAt time.Time
			rows.Scan(&id, &postID, &userID, &content, &status, &createdAt)
			comments = append(comments, iris.Map{
				"id":         id,
				"post_id":    postID,
				"user_id":    userID,
				"content":    content,
				"status":     status,
				"created_at": createdAt,
			})
		}
		ctx.JSON(iris.Map{"comments": comments, "count": len(comments)})
	})

	// Debug API to check users
	app.Get("/api/debug/users", func(ctx iris.Context) {
		rows, err := DB.Query(`SELECT id, username, email, role FROM users ORDER BY id DESC LIMIT 20`)
		if err != nil {
			ctx.JSON(iris.Map{"error": err.Error()})
			return
		}
		defer rows.Close()

		var users []iris.Map
		for rows.Next() {
			var id int64
			var username, email, role string
			rows.Scan(&id, &username, &email, &role)
			users = append(users, iris.Map{
				"id":       id,
				"username": username,
				"email":    email,
				"role":     role,
			})
		}
		ctx.JSON(iris.Map{"users": users, "count": len(users)})
	})

	// Debug: cleanup orphan comments
	app.Get("/api/debug/cleanup", func(ctx iris.Context) {
		// Delete comments where user_id doesn't exist in users table
		result, err := DB.Exec(`DELETE c FROM comments c LEFT JOIN users u ON c.user_id = u.id WHERE u.id IS NULL`)
		if err != nil {
			ctx.JSON(iris.Map{"error": err.Error()})
			return
		}
		deleted, _ := result.RowsAffected()
		ctx.JSON(iris.Map{"deleted_orphan_comments": deleted})
	})

	// Debug: get all comments with user join
	app.Get("/api/debug/comments-all", func(ctx iris.Context) {
		comments, err := GetAllComments()
		if err != nil {
			ctx.JSON(iris.Map{"error": err.Error()})
			return
		}
		var result []iris.Map
		for _, c := range comments {
			result = append(result, iris.Map{
				"id":         c.ID,
				"post_id":    c.PostID,
				"user_id":    c.UserID,
				"username":   c.Username,
				"content":    c.Content,
				"status":     c.Status,
				"created_at": c.CreatedAt,
			})
		}
		ctx.JSON(iris.Map{"comments": result, "count": len(result)})
	})

	log.Println("========================================")

	app.Listen(":8080")
}

// ============================================================
// Helper functions
// ============================================================

func parseInt(s string, defaultValue int) int {
	var result int
	if _, err := fmt.Sscanf(s, "%d", &result); err != nil {
		return defaultValue
	}
	return result
}

func getPosts() ([]Post, error) {
	if postRepo != nil {
		return postRepo.GetAll()
	}
	return []Post{}, nil
}

func getPostByID(id int64) (*Post, error) {
	log.Printf("[DEBUG] getPostByID: id=%d, postRepo=%v", id, postRepo != nil)
	if postRepo != nil {
		return postRepo.GetByID(id)
	}
	return nil, nil
}

func createPost(title, content, author, category, tags string) (int64, error) {
	if postRepo != nil {
		return postRepo.Create(&Post{
			Title:    title,
			Content:  content,
			Author:   author,
			Category: category,
			Tags:     tags,
		})
	}
	return 0, nil
}

func updatePost(id int64, title, content, author, category, tags string) error {
	if postRepo != nil {
		return postRepo.Update(&Post{
			ID:       id,
			Title:    title,
			Content:  content,
			Author:   author,
			Category: category,
			Tags:     tags,
		})
	}
	return nil
}

func deletePost(id int64) error {
	if postRepo != nil {
		return postRepo.Delete(id)
	}
	return nil
}
