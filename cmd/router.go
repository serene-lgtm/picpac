package main

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"pack_mate/internal/config"
	"pack_mate/internal/handler"
	mongodb "pack_mate/internal/repository/mongodb"
	"pack_mate/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/tencentyun/cos-go-sdk-v5"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func newRouter(cfg *config.Configuration, client *cos.Client, bucketURL *url.URL, db *mongo.Database) *gin.Engine {
	router := gin.Default()
	router.Use(corsMiddleware(cfg.CORS.AllowedOrigins))

	itemRepo := mongodb.NewItemRepository(db)
	packRepo := mongodb.NewPackRepository(db)
	checklistRepo := mongodb.NewChecklistRepository(db)
	userRepo := mongodb.NewUserRepository(db)
	authIdentityRepo := mongodb.NewAuthIdentityRepository(db)
	phoneCodeRepo := mongodb.NewPhoneVerificationCodeRepository(db)
	refreshTokenRepo := mongodb.NewRefreshTokenRepository(db)
	uploadService := service.NewCOSUploadService(client, bucketURL)
	tokenService := service.NewTokenService(cfg.Auth.AccessTokenSecret, time.Duration(cfg.Auth.AccessTokenTTLSeconds)*time.Second)
	itemService := service.NewItemService(itemRepo, uploadService)
	packService := service.NewPackService(packRepo, itemRepo)
	checklistService := service.NewChecklistService(checklistRepo, itemRepo)
	authService := service.NewAuthService(userRepo, authIdentityRepo, phoneCodeRepo, refreshTokenRepo, service.NewFakeSMSService(), tokenService, cfg.Auth)

	registerAPIRoutes(router, itemService, packService, checklistService, authService, tokenService)

	return router
}

func registerAPIRoutes(router *gin.Engine, itemService service.ItemService, packService service.PackService, checklistService service.ChecklistService, authService service.AuthService, tokenService service.TokenService) {
	itemHandler := handler.NewItemHandler(itemService)
	packHandler := handler.NewPackHandler(packService)
	checklistHandler := handler.NewChecklistHandler(checklistService)
	authHandler := handler.NewAuthHandler(authService)
	authMiddleware := handler.NewAuthMiddleware(tokenService, authService)

	authRoutes := router.Group("/api/v1/auth")
	authRoutes.POST("/phone/code", authHandler.SendPhoneCode)
	authRoutes.POST("/phone/login", authHandler.LoginWithPhone)
	authRoutes.POST("/refresh", authHandler.Refresh)
	authRoutes.POST("/logout", authHandler.Logout)
	router.GET("/api/v1/me", authMiddleware.RequireAuth(), authHandler.Me)

	itemRoutes := router.Group("/api/v1/item")
	itemRoutes.Use(authMiddleware.RequireAuth())
	itemRoutes.POST("", itemHandler.CreateItem)
	itemRoutes.GET("", itemHandler.ListItems)
	itemRoutes.GET("/:item_id", itemHandler.GetItem)
	itemRoutes.PUT("/:item_id", itemHandler.UpdateItem)
	itemRoutes.DELETE("/:item_id", itemHandler.DeleteItem)

	packRoutes := router.Group("/api/v1/pack")
	packRoutes.Use(authMiddleware.RequireAuth())
	packRoutes.POST("", packHandler.CreatePack)
	packRoutes.GET("", packHandler.ListPacks)
	packRoutes.GET("/:pack_id", packHandler.GetPack)
	packRoutes.PUT("/:pack_id", packHandler.UpdatePack)
	packRoutes.DELETE("/:pack_id", packHandler.DeletePack)

	checklistRoutes := router.Group("/api/v1/checklist")
	checklistRoutes.Use(authMiddleware.RequireAuth())
	checklistRoutes.POST("", checklistHandler.CreateChecklist)
	checklistRoutes.GET("", checklistHandler.ListChecklists)
	checklistRoutes.GET("/:checklist_id", checklistHandler.GetChecklist)
	checklistRoutes.PUT("/:checklist_id", checklistHandler.UpdateChecklist)
	checklistRoutes.POST("/:checklist_id/items", checklistHandler.AddChecklistLineItems)
	checklistRoutes.DELETE("/:checklist_id/items", checklistHandler.RemoveChecklistLineItems)
	checklistRoutes.PATCH("/:checklist_id/items/:line_item_id/status", checklistHandler.UpdateChecklistLineItemStatus)
	checklistRoutes.DELETE("/:checklist_id", checklistHandler.DeleteChecklist)
}

func corsMiddleware(origins []string) gin.HandlerFunc {
	allowedOrigins := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		allowedOrigins[origin] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := allowedOrigins[origin]; ok {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
