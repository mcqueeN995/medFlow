package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/medflow/backend/internal/config"
	"github.com/medflow/backend/internal/handler" // Исправлено на handlers (множественное число)
	"github.com/medflow/backend/internal/middleware"
	"github.com/medflow/backend/internal/models"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func SetupRouter(
	cfg *config.Config,
	authHandler *handler.AuthHandler,
	forumHandler *handler.ForumHandler,
	userHandler *handler.UserHandler,
	libraryHandler *handler.LibraryHandler,
	uploadHandler *handler.UploadHandler,
	cardHandler *handler.CardHandler,
	poiHandler *handler.POIHandler,
	adminHandler *handler.AdminHandler,
	pushHandler *handler.PushHandler,
) *gin.Engine {
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	router.Use(middleware.RequestID())
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS(middleware.DefaultCORSConfig()))
	router.Use(middleware.RateLimiter(middleware.DefaultRateLimiterConfig()))
	router.Use(middleware.Metrics())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// /metrics - Prometheus scrape-таргет (см. infra/prometheus/prometheus.yml,
	// профиль monitoring). Вне /api/v1 - как /health, это инфраструктурный
	// эндпоинт, не часть публичного API-контракта.
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	api := router.Group("/api/v1")

	auth := api.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", authHandler.Logout)

		// TODO: Добавить после реализации отправки Email / Redis
		// auth.POST("/forgot-password", authHandler.ForgotPassword)
		// auth.POST("/reset-password", authHandler.ResetPassword)
		// auth.GET("/verify-email", authHandler.VerifyEmail)
	}

	// --- USERS (protected - ни один /users-эндпоинт не имеет security: [] в openapi.yaml) ---
	users := api.Group("/users")
	users.Use(middleware.AuthRequired(cfg))
	{
		users.GET("/me", userHandler.Me)
		users.PATCH("/me", userHandler.UpdateMe)
		users.DELETE("/me", userHandler.DeleteMe)
		users.GET("/:id", userHandler.PublicProfile)
	}

	// --- LIBRARY ---
	// Каталог и /source публичны (security: [] в openapi.yaml), а /download
	// требует входа - у него нет переопределения security на уровне операции.
	library := api.Group("/library")
	{
		library.GET("/textbooks", libraryHandler.ListTextbooks)
		library.GET("/textbooks/:id", libraryHandler.GetTextbook)
		library.GET("/textbooks/:id/download", middleware.AuthRequired(cfg), libraryHandler.Download)
		library.GET("/textbooks/:id/source", libraryHandler.Source)
	}

	// --- CARDS (protected - весь модуль закрыт, security: [] нигде не
	// переопределён для /cards/* в openapi.yaml) ---
	cards := api.Group("/cards")
	cards.Use(middleware.AuthRequired(cfg))
	{
		cards.POST("/tasks", cardHandler.CreateTask)
		cards.GET("/tasks", cardHandler.ListTasks)
		cards.GET("/tasks/:id", cardHandler.GetTask)
		cards.GET("/tasks/:id/cards", cardHandler.ListTaskCards)
		cards.GET("/review", cardHandler.Review)
		cards.POST("/:id/rate", cardHandler.RateCard)
		cards.POST("/:id/report", cardHandler.ReportCard)
		cards.GET("/stats", cardHandler.Stats)
	}

	// --- UPLOAD (protected) ---
	upload := api.Group("/upload")
	upload.Use(middleware.AuthRequired(cfg))
	{
		upload.POST("", uploadHandler.Upload)
	}

	// --- MAP (public, security: [] в openapi.yaml) ---
	mapGroup := api.Group("/map")
	{
		mapGroup.GET("/poi", poiHandler.ListPOI)
	}

	// --- ADMIN POI (admin only) ---
	adminPOI := api.Group("/admin/poi")
	adminPOI.Use(middleware.AuthRequired(cfg), middleware.RequireRole(models.RoleAdmin))
	{
		adminPOI.GET("", poiHandler.AdminListPOI)
		adminPOI.POST("", poiHandler.AdminCreatePOI)
		adminPOI.PATCH("/:id", poiHandler.AdminUpdatePOI)
		adminPOI.DELETE("/:id", poiHandler.AdminDeletePOI)
	}

	// --- THREADS (protected - весь форум требует входа, security: [] нигде
	// не переопределён в openapi.yaml, в отличие от Library/Map) ---
	threads := api.Group("/threads")
	threads.Use(middleware.AuthRequired(cfg))
	{
		threads.GET("", forumHandler.ListThreads)
		threads.POST("", forumHandler.CreateThread)
		threads.GET("/:id", forumHandler.GetThread)
		threads.PATCH("/:id", forumHandler.UpdateThread)
		threads.DELETE("/:id", forumHandler.DeleteThread)
		threads.POST("/:id/reactions", forumHandler.AddThreadReaction)
		threads.DELETE("/:id/reactions", forumHandler.RemoveThreadReaction)
		threads.POST("/:id/report", forumHandler.ReportThread)
		// POST /:id/comments и GET/POST/PATCH/DELETE /:id/... должны использовать
		// одно и то же имя параметра - gin паникует, если для одного HTTP-метода
		// один и тот же узел дерева маршрутов регистрируется с разными именами
		// wildcard'ов (тут было бы :id vs :thread_id на одной позиции для POST).
		threads.GET("/:id/comments", forumHandler.ListComments)
		threads.POST("/:id/comments", forumHandler.CreateComment)
	}

	// --- COMMENTS (protected) ---
	comments := api.Group("/comments")
	comments.Use(middleware.AuthRequired(cfg))
	{
		comments.PATCH("/:id", forumHandler.UpdateComment)
		comments.DELETE("/:id", forumHandler.DeleteComment)
		comments.POST("/:id/reactions", forumHandler.AddCommentReaction)
		comments.DELETE("/:id/reactions", forumHandler.RemoveCommentReaction)
		comments.POST("/:id/report", forumHandler.ReportComment)
	}

	// --- SUBSCRIPTIONS (protected) ---
	// subscriptions := api.Group("/subscriptions")
	// subscriptions.Use(middleware.AuthRequired(cfg))
	// {
	//     subscriptions.POST("", subscriptionHandler.Create)
	//     subscriptions.DELETE("/:id", subscriptionHandler.Delete)
	//     subscriptions.GET("/feed", subscriptionHandler.Feed)
	// }

	// --- NOTIFICATIONS (protected) ---
	// notifications := api.Group("/notifications")
	// notifications.Use(middleware.AuthRequired(cfg))
	// {
	//     notifications.GET("", notificationHandler.List)
	//     notifications.PATCH("/read", notificationHandler.MarkRead)
	// }

	// --- PUSH (protected) ---
	push := api.Group("/push")
	push.Use(middleware.AuthRequired(cfg))
	{
		push.POST("/subscribe", pushHandler.Subscribe)
		push.DELETE("/unsubscribe", pushHandler.Unsubscribe)
		push.PATCH("/preferences", pushHandler.UpdatePreferences)
	}

	// --- ADMIN LIBRARY (admin only) ---
	adminLibrary := api.Group("/admin/library")
	adminLibrary.Use(middleware.AuthRequired(cfg), middleware.RequireRole(models.RoleAdmin))
	{
		adminLibrary.GET("/textbooks", libraryHandler.AdminListTextbooks)
		adminLibrary.POST("/textbooks", libraryHandler.AdminCreateTextbook)
		adminLibrary.PATCH("/textbooks/:id", libraryHandler.AdminUpdateTextbook)
		adminLibrary.DELETE("/textbooks/:id", libraryHandler.AdminDeleteTextbook)
	}

	// --- ADMIN USERS (admin only) ---
	adminUsers := api.Group("/admin/users")
	adminUsers.Use(middleware.AuthRequired(cfg), middleware.RequireRole(models.RoleAdmin))
	{
		adminUsers.GET("", userHandler.AdminListUsers)
		adminUsers.PATCH("/:id/role", userHandler.AdminChangeRole)
		adminUsers.POST("/:id/ban", userHandler.AdminBanUser)
		adminUsers.DELETE("/:id/ban", userHandler.AdminUnbanUser)
	}

	// --- ADMIN REPORTS (moderator+) ---
	adminReports := api.Group("/admin/reports")
	adminReports.Use(middleware.AuthRequired(cfg), middleware.RequireRole(models.RoleModerator, models.RoleAdmin))
	{
		adminReports.GET("", adminHandler.AdminListReports)
		adminReports.PATCH("/:id", adminHandler.AdminReviewReport)
	}

	// --- ADMIN MODERATION ---
	// hide - moderator+; полное удаление - admin only, поэтому роль задаётся
	// поточечно на каждый route, а не на всю группу (см. openapi.yaml:
	// "Скрыть тред (moderator+)" vs "Полное удаление треда (admin)").
	adminThreads := api.Group("/admin/threads")
	adminThreads.Use(middleware.AuthRequired(cfg))
	{
		adminThreads.POST("/:id/hide", middleware.RequireRole(models.RoleModerator, models.RoleAdmin), forumHandler.AdminHideThread)
		adminThreads.DELETE("/:id", middleware.RequireRole(models.RoleAdmin), forumHandler.AdminDeleteThread)
	}
	adminComments := api.Group("/admin/comments")
	adminComments.Use(middleware.AuthRequired(cfg))
	{
		adminComments.POST("/:id/hide", middleware.RequireRole(models.RoleModerator, models.RoleAdmin), forumHandler.AdminHideComment)
		adminComments.DELETE("/:id", middleware.RequireRole(models.RoleAdmin), forumHandler.AdminDeleteComment)
	}

	// --- ADMIN SYSTEM (admin only) ---
	adminSystem := api.Group("/admin")
	adminSystem.Use(middleware.AuthRequired(cfg), middleware.RequireRole(models.RoleAdmin))
	{
		adminSystem.GET("/stats", adminHandler.AdminStats)
		adminSystem.GET("/audit-logs", adminHandler.AdminAuditLogs)
	}

	return router
}
