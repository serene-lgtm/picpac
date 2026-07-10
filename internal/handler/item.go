package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"
	"pack_mate/internal/dto/response"
	"pack_mate/internal/service"

	"github.com/gin-gonic/gin"
)

// ItemHandler handles item HTTP requests.
type ItemHandler struct {
	svc       service.ItemService
	urlSigner service.ObjectURLSigner
}

// NewItemHandler creates an item handler.
func NewItemHandler(svc service.ItemService, urlSigner service.ObjectURLSigner) *ItemHandler {
	return &ItemHandler{svc: svc, urlSigner: urlSigner}
}

// CreateItem handles item creation requests.
func (h *ItemHandler) CreateItem(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization is required"})
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))
	description := c.PostForm("description")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	input := request.CreateItemInput{
		UserID:      userID,
		Name:        name,
		Description: description,
	}
	if file, header, err := c.Request.FormFile("image"); err == nil {
		defer file.Close()
		input.File = file
		input.FileName = header.Filename
	}

	item, err := h.svc.CreateItem(c.Request.Context(), input)
	if err != nil {
		respondItemError(c, err)
		return
	}

	itemResponse, err := h.buildItemResponse(c.Request.Context(), item)
	if err != nil {
		respondItemError(c, err)
		return
	}

	c.JSON(http.StatusOK, itemResponse)
}

// ListItems handles item list requests.
func (h *ItemHandler) ListItems(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization is required"})
		return
	}

	q, hasQ := c.GetQuery("q")
	q = strings.TrimSpace(q)
	if hasQ && q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "q is required"})
		return
	}

	items, err := h.svc.ListItems(c.Request.Context(), request.ListItemsInput{
		UserID: userID,
		Q:      q,
		HasQ:   hasQ,
	})
	if err != nil {
		respondItemError(c, err)
		return
	}

	responses := make([]response.ItemResponse, 0, len(items))
	for _, item := range items {
		itemResponse, err := h.buildItemResponse(c.Request.Context(), &item)
		if err != nil {
			respondItemError(c, err)
			return
		}
		responses = append(responses, itemResponse)
	}

	c.JSON(http.StatusOK, response.ListItemsResponse{Items: responses})
}

// GetItem handles item detail requests.
func (h *ItemHandler) GetItem(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization is required"})
		return
	}

	itemID := strings.TrimSpace(c.Param("item_id"))
	if itemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "item_id is required"})
		return
	}

	item, err := h.svc.GetItem(c.Request.Context(), itemID, userID)
	if err != nil {
		respondItemError(c, err)
		return
	}

	itemResponse, err := h.buildItemResponse(c.Request.Context(), item)
	if err != nil {
		respondItemError(c, err)
		return
	}

	c.JSON(http.StatusOK, itemResponse)
}

// UpdateItem handles item update requests.
func (h *ItemHandler) UpdateItem(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization is required"})
		return
	}

	itemID := strings.TrimSpace(c.Param("item_id"))
	if itemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "item_id is required"})
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))
	description := c.PostForm("description")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	input := request.UpdateItemInput{
		Name:        name,
		Description: description,
	}
	if file, header, err := c.Request.FormFile("image"); err == nil {
		defer file.Close()
		input.File = file
		input.FileName = header.Filename
	}

	item, err := h.svc.UpdateItem(c.Request.Context(), itemID, userID, input)
	if err != nil {
		respondItemError(c, err)
		return
	}

	itemResponse, err := h.buildItemResponse(c.Request.Context(), item)
	if err != nil {
		respondItemError(c, err)
		return
	}

	c.JSON(http.StatusOK, itemResponse)
}

// DeleteItem handles item deletion requests.
func (h *ItemHandler) DeleteItem(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization is required"})
		return
	}

	itemID := strings.TrimSpace(c.Param("item_id"))
	if itemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "item_id is required"})
		return
	}

	if err := h.svc.DeleteItem(c.Request.Context(), itemID, userID); err != nil {
		respondItemError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.DeleteItemResponse{Deleted: true})
}

func respondItemError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := err.Error()
	switch {
	case strings.Contains(message, "invalid input"),
		strings.Contains(message, "item name is required"),
		strings.Contains(message, "item search keyword is required"),
		strings.Contains(message, "item search keyword is too long"):
		status = http.StatusBadRequest
	case strings.Contains(message, "item not found"):
		status = http.StatusNotFound
	case strings.Contains(message, "upload item image failed"):
		status = http.StatusBadGateway
	case strings.Contains(message, "sign item image url failed"):
		status = http.StatusInternalServerError
	case strings.Contains(message, "create item failed"),
		strings.Contains(message, "list items failed"),
		strings.Contains(message, "get item failed"),
		strings.Contains(message, "update item failed"),
		strings.Contains(message, "delete item failed"):
		status = http.StatusInternalServerError
	}
	c.JSON(status, gin.H{"error": message})
}

func (h *ItemHandler) buildItemResponse(ctx context.Context, item *domain.Item) (response.ItemResponse, error) {
	sourceImageURL, err := h.signObjectURL(ctx, item.SourceImageObjectKey, "source image")
	if err != nil {
		return response.ItemResponse{}, err
	}
	imageThumbnailURL, err := h.signObjectURL(ctx, item.ImageThumbnailObjectKey, "image thumbnail")
	if err != nil {
		return response.ItemResponse{}, err
	}
	aiRenderedImageURL, err := h.signObjectURL(ctx, item.AIRenderedImageObjectKey, "ai rendered image")
	if err != nil {
		return response.ItemResponse{}, err
	}

	return response.ItemResponse{
		ID:                 item.ID.Hex(),
		UserID:             item.UserID.Hex(),
		Name:               item.Name,
		Description:        item.Description,
		SourceImageURL:     sourceImageURL,
		ImageThumbnailURL:  imageThumbnailURL,
		AIRenderedImageURL: aiRenderedImageURL,
		Status:             string(item.Status),
	}, nil
}

func (h *ItemHandler) signObjectURL(ctx context.Context, objectKey string, label string) (string, error) {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return "", nil
	}
	if h.urlSigner == nil {
		return "", fmt.Errorf("sign item image url failed: url signer is not configured")
	}

	signedURL, err := h.urlSigner.SignGetURL(ctx, objectKey)
	if err != nil {
		return "", fmt.Errorf("sign item image url failed: %s: %w", label, err)
	}

	return signedURL, nil
}
