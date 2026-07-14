package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/medflow/backend/internal/config"
	"github.com/medflow/backend/internal/handler"
	"github.com/medflow/backend/internal/middleware"
)

func SetupRouter(
	cfg *config.Config,
	authHandler *handler.AuthHandler,
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

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	api := router.Group("/api/v1")

	auth := api.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", authHandler.Logout)
	}

	// --- USERS (protected routes) ---
	// TODO: Раскомментировать после реализации Users модуля
	// users := api.Group("/users")
	// users.Use(middleware.AuthRequired(cfg))
	// {
	//     users.GET("/me", userHandler.Me)
	//     users.PATCH("/me", userHandler.Update)
	//     users.DELETE("/me", userHandler.Delete)
	//     users.GET("/:id", userHandler.PublicProfile)
	// }

	// --- LIBRARY (public catalog) ---
	// TODO: Раскомментировать после реализации Library модуля
	// library := api.Group("/library")
	// {
	//     library.GET("/textbooks", libraryHandler.List)
	//     library.GET("/textbooks/:id", libraryHandler.Get)
	//     library.GET("/textbooks/:id/download", libraryHandler.Download)
	//     library.GET("/textbooks/:id/source", libraryHandler.Source)
	// }

	// --- CARDS (protected) ---
	// TODO: Раскомментировать после реализации Cards модуля
	// cards := api.Group("/cards")
	// cards.Use(middleware.AuthRequired(cfg))
	// {
	//     cards.POST("/tasks", cardHandler.CreateTask)
	//     cards.GET("/tasks", cardHandler.ListTasks)
	//     cards.GET("/tasks/:id", cardHandler.GetTask)
	//     cards.GET("/tasks/:id/cards", cardHandler.GetTaskCards)
	//     cards.GET("/review", cardHandler.Review)
	//     cards.POST("/:id/rate", cardHandler.Rate)
	//     cards.POST("/:id/report", cardHandler.Report)
	//     cards.GET("/stats", cardHandler.Stats)
	// }

	// --- UPLOAD (protected) ---
	// TODO: Раскомментировать после реализации Upload модуля
	// upload := api.Group("/upload")
	// upload.Use(middleware.AuthRequired(cfg))
	// {
	//     upload.POST("", uploadHandler.Upload)
	// }

	// --- MAP (public) ---
	// TODO: Раскомментировать после реализации Map модуля
	// mapGroup := api.Group("/map")
	// {
	//     mapGroup.GET("/poi", mapHandler.ListPOI)
	// }

	// --- THREADS (protected) ---
	// TODO: Раскомментировать после реализации Threads модуля
	// threads := api.Group("/threads")
	// threads.Use(middleware.AuthRequired(cfg))
	// {
	//     threads.GET("", threadHandler.List)
	//     threads.POST("", threadHandler.Create)
	//     threads.GET("/:id", threadHandler.Get)
	//     threads.PATCH("/:id", threadHandler.Update)
	//     threads.DELETE("/:id", threadHandler.Delete)
	//     threads.POST("/:id/reactions", threadHandler.AddReaction)
	//     threads.DELETE("/:id/reactions", threadHandler.RemoveReaction)
	//     threads.POST("/:id/report", threadHandler.Report)
	//     threads.GET("/:id/comments", commentHandler.List)
	// }

	// --- COMMENTS (protected) ---
	// TODO: Раскомментировать после реализации Comments модуля
	// comments := api.Group("/comments")
	// comments.Use(middleware.AuthRequired(cfg))
	// {
	//     comments.POST("", commentHandler.Create)
	//     comments.PATCH("/:id", commentHandler.Update)
	//     comments.DELETE("/:id", commentHandler.Delete)
	//     comments.POST("/:id/reactions", commentHandler.AddReaction)
	//     comments.DELETE("/:id/reactions", commentHandler.RemoveReaction)
	//     comments.POST("/:id/report", commentHandler.Report)
	// }

	// --- SUBSCRIPTIONS (protected) ---
	// TODO: Раскомментировать после реализации Subscriptions модуля
	// subscriptions := api.Group("/subscriptions")
	// subscriptions.Use(middleware.AuthRequired(cfg))
	// {
	//     subscriptions.POST("", subscriptionHandler.Create)
	//     subscriptions.DELETE("/:id", subscriptionHandler.Delete)
	//     subscriptions.GET("/feed", subscriptionHandler.Feed)
	// }

	// --- NOTIFICATIONS (protected) ---
	// TODO: Раскомментировать после реализации Notifications модуля
	// notifications := api.Group("/notifications")
	// notifications.Use(middleware.AuthRequired(cfg))
	// {
	//     notifications.GET("", notificationHandler.List)
	//     notifications.PATCH("/read", notificationHandler.MarkRead)
	// }

	// --- PUSH (protected) ---
	// TODO: Раскомментировать после реализации Push модуля
	// push := api.Group("/push")
	// push.Use(middleware.AuthRequired(cfg))
	// {
	//     push.POST("/subscribe", pushHandler.Subscribe)
	//     push.DELETE("/unsubscribe", pushHandler.Unsubscribe)
	//     push.PATCH("/preferences", pushHandler.UpdatePreferences)
	// }

	// --- ADMIN (protected, role-based) ---
	// TODO: Раскомментировать после реализации Admin модуля
	// admin := api.Group("/admin")
	// admin.Use(middleware.AuthRequired(cfg))
	// {
	//     // Admin Users
	//     adminUsers := admin.Group("/users")
	//     adminUsers.Use(middleware.RequireRole(models.RoleAdmin))
	//     {
	//         adminUsers.GET("", adminUserHandler.List)
	//         adminUsers.PATCH("/:id/role", adminUserHandler.ChangeRole)
	//         adminUsers.POST("/:id/ban", adminUserHandler.Ban)
	//         adminUsers.DELETE("/:id/ban", adminUserHandler.Unban)
	//     }
	//
	//     // Admin Reports (moderator+)
	//     adminReports := admin.Group("/reports")
	//     adminReports.Use(middleware.RequireRole(models.RoleModerator, models.RoleAdmin))
	//     {
	//         adminReports.GET("", adminReportHandler.List)
	//         adminReports.PATCH("/:id", adminReportHandler.Review)
	//     }
	//
	//     // Admin Moderation (moderator+)
	//     adminMod := admin.Group("")
	//     adminMod.Use(middleware.RequireRole(models.RoleModerator, models.RoleAdmin))
	//     {
	//         adminMod.POST("/threads/:id/hide", adminModHandler.HideThread)
	//         adminMod.POST("/comments/:id/hide", adminModHandler.HideComment)
	//         adminMod.DELETE("/threads/:id", adminModHandler.DeleteThread)
	//         adminMod.DELETE("/comments/:id", adminModHandler.DeleteComment)
	//     }
	//
	//     // Admin Library (admin only)
	//     adminLibrary := admin.Group("/library")
	//     adminLibrary.Use(middleware.RequireRole(models.RoleAdmin))
	//     {
	//         adminLibrary.GET("/textbooks", adminLibraryHandler.List)
	//         adminLibrary.POST("/textbooks", adminLibraryHandler.Create)
	//         adminLibrary.PATCH("/textbooks/:id", adminLibraryHandler.Update)
	//         adminLibrary.DELETE("/textbooks/:id", adminLibraryHandler.Delete)
	//     }
	//
	//     // Admin POI (admin only)
	//     adminPOI := admin.Group("/poi")
	//     adminPOI.Use(middleware.RequireRole(models.RoleAdmin))
	//     {
	//         adminPOI.GET("", adminPOIHandler.List)
	//         adminPOI.POST("", adminPOIHandler.Create)
	//         adminPOI.PATCH("/:id", adminPOIHandler.Update)
	//         adminPOI.DELETE("/:id", adminPOIHandler.Delete)
	//     }
	//
	//     // Admin System (admin only)
	//     adminSystem := admin.Group("")
	//     adminSystem.Use(middleware.RequireRole(models.RoleAdmin))
	//     {
	//         adminSystem.GET("/stats", adminSystemHandler.Stats)
	//         adminSystem.GET("/audit-logs", adminSystemHandler.AuditLogs)
	//     }
	// }

	return router
}
