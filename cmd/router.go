package main

import (
	"net/http"
	"net/url"
	"strings"

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
	uploadService := service.NewCOSUploadService(client, bucketURL)
	itemService := service.NewItemService(itemRepo, uploadService)
	packService := service.NewPackService(packRepo)

	registerAPIRoutes(router, itemService, packService)

	return router
}

func registerAPIRoutes(router *gin.Engine, itemService service.ItemService, packService service.PackService) {
	itemHandler := handler.NewItemHandler(itemService)
	packHandler := handler.NewPackHandler(packService)

	itemRoutes := router.Group("/api/v1/item")
	itemRoutes.POST("", itemHandler.CreateItem)
	itemRoutes.GET("", itemHandler.ListItems)
	itemRoutes.GET("/:item_id", itemHandler.GetItem)
	itemRoutes.PUT("/:item_id", itemHandler.UpdateItem)
	itemRoutes.DELETE("/:item_id", itemHandler.DeleteItem)

	packRoutes := router.Group("/api/v1/pack")
	packRoutes.POST("", packHandler.CreatePack)
	packRoutes.GET("", packHandler.ListPacks)
	packRoutes.GET("/:pack_id", packHandler.GetPack)
	packRoutes.PUT("/:pack_id", packHandler.UpdatePack)
	packRoutes.DELETE("/:pack_id", packHandler.DeletePack)
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
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
