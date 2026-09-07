package handler

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"
	"pack_mate/internal/dto/response"
	"pack_mate/internal/service"

	"github.com/gin-gonic/gin"
)

const defaultItemCoverObjectKey = "items/default/cover.jpg"

// ItemHandler handles item HTTP requests.
type ItemHandler struct {
	svc        service.ItemService
	categories service.CategoryService
	urlSigner  service.ObjectURLSigner
}

// NewItemHandler creates an item handler.
func NewItemHandler(svc service.ItemService, categories service.CategoryService, urlSigner service.ObjectURLSigner) *ItemHandler {
	return &ItemHandler{svc: svc, categories: categories, urlSigner: urlSigner}
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
		CategoryID:  strings.TrimSpace(c.PostForm("category_id")),
		Name:        name,
		Description: description,
	}
	if file, header, err := c.Request.FormFile("image"); err == nil {
		defer file.Close()
		input.Photos = []request.ItemPhotoUploadInput{{
			File:     file,
			FileName: header.Filename,
		}}
	}
	if photos, closePhotos, err := itemPhotoUploads(c); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	} else if len(photos) > 0 {
		defer closePhotos()
		input.Photos = photos
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

// CreateItemsBatch handles batch item creation requests.
func (h *ItemHandler) CreateItemsBatch(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization is required"})
		return
	}

	var input request.BatchCreateItemsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	if len(input.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "items are required"})
		return
	}
	input.UserID = userID

	items, err := h.svc.CreateItemsBatch(c.Request.Context(), input)
	if err != nil {
		respondItemError(c, err)
		return
	}

	itemResponses := make([]response.ItemResponse, 0, len(items))
	for _, item := range items {
		itemResponse, err := h.buildItemResponse(c.Request.Context(), &item)
		if err != nil {
			respondItemError(c, err)
			return
		}
		itemResponses = append(itemResponses, itemResponse)
	}

	c.JSON(http.StatusOK, response.BatchCreateItemsResponse{Items: itemResponses})
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
	categoryID, hasCategoryID := c.GetQuery("category_id")
	categoryID = strings.TrimSpace(categoryID)
	if hasCategoryID && categoryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category_id is required"})
		return
	}

	items, err := h.svc.ListItems(c.Request.Context(), request.ListItemsInput{
		UserID:        userID,
		Q:             q,
		HasQ:          hasQ,
		CategoryID:    categoryID,
		HasCategoryID: hasCategoryID,
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

	categoryID, hasCategoryID := c.GetPostForm("category_id")
	input := request.UpdateItemInput{
		CategoryID:    strings.TrimSpace(categoryID),
		HasCategoryID: hasCategoryID,
		Name:          name,
		Description:   description,
	}
	if file, header, err := c.Request.FormFile("image"); err == nil {
		defer file.Close()
		input.Photos = []request.ItemPhotoUploadInput{{
			File:     file,
			FileName: header.Filename,
		}}
		input.HasPhotos = true
	}
	if photos, closePhotos, err := itemPhotoUploads(c); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	} else if len(photos) > 0 {
		defer closePhotos()
		input.Photos = photos
		input.HasPhotos = true
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
		strings.Contains(message, "invalid category"),
		strings.Contains(message, "category_id is required"),
		strings.Contains(message, "item name is required"),
		strings.Contains(message, "item photos are too many"),
		strings.Contains(message, "items are required"),
		strings.Contains(message, "items are too many"),
		strings.Contains(message, "item search keyword is required"),
		strings.Contains(message, "item search keyword is too long"):
		status = http.StatusBadRequest
	case strings.Contains(message, "item not found"):
		status = http.StatusNotFound
	case strings.Contains(message, "upload item image failed"):
		status = http.StatusBadGateway
	case strings.Contains(message, "sign item image url failed"):
		status = http.StatusInternalServerError
	case strings.Contains(message, "default category not found"),
		strings.Contains(message, "get default category failed"),
		strings.Contains(message, "get category failed"):
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
	photos, err := h.buildItemPhotoResponses(ctx, item.Photos)
	if err != nil {
		return response.ItemResponse{}, err
	}
	category, err := h.categoryForItem(ctx, item)
	if err != nil {
		return response.ItemResponse{}, err
	}

	itemResponse := response.ItemResponse{
		ID:          item.ID.Hex(),
		UserID:      item.UserID.Hex(),
		Name:        item.Name,
		Description: item.Description,
		Photos:      photos,
		Status:      string(item.Status),
	}
	if len(photos) > 0 {
		itemResponse.CoverImageURL = photos[0].ImageURL
	} else {
		coverImageURL, err := h.signObjectURL(ctx, defaultItemCoverObjectKey, "default cover image")
		if err != nil {
			return response.ItemResponse{}, err
		}
		itemResponse.CoverImageURL = coverImageURL
	}
	if category != nil {
		itemResponse.CategoryID = category.ID.Hex()
		itemResponse.CategoryKey = category.Key
		itemResponse.CategoryName = category.Name
	}

	return itemResponse, nil
}

func (h *ItemHandler) buildItemPhotoResponses(ctx context.Context, photos []domain.ItemPhoto) ([]response.ItemPhotoResponse, error) {
	responses := make([]response.ItemPhotoResponse, 0, len(photos))
	for _, photo := range photos {
		sourceImageURL, err := h.signObjectURL(ctx, photo.SourceObjectKey, "source image")
		if err != nil {
			return nil, err
		}
		imageURL, err := h.signObjectURL(ctx, photo.DisplayObjectKey, "display image")
		if err != nil {
			return nil, err
		}
		responses = append(responses, response.ItemPhotoResponse{
			ID:             photo.ID.Hex(),
			SourceImageURL: sourceImageURL,
			ImageURL:       imageURL,
		})
	}

	return responses, nil
}

func (h *ItemHandler) categoryForItem(ctx context.Context, item *domain.Item) (*domain.Category, error) {
	if h.categories == nil {
		return nil, nil
	}
	return h.categories.GetCategoryByID(ctx, item.CategoryID)
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

func itemPhotoUploads(c *gin.Context) ([]request.ItemPhotoUploadInput, func(), error) {
	form := c.Request.MultipartForm
	if form == nil || len(form.File["photos"]) == 0 {
		return nil, func() {}, nil
	}

	uploads := make([]request.ItemPhotoUploadInput, 0, len(form.File["photos"]))
	files := make([]multipart.File, 0, len(form.File["photos"]))
	for _, header := range form.File["photos"] {
		file, err := header.Open()
		if err != nil {
			closeMultipartFiles(files)
			return nil, func() {}, err
		}
		files = append(files, file)
		readSeeker, ok := file.(io.ReadSeeker)
		if !ok {
			closeMultipartFiles(files)
			return nil, func() {}, fmt.Errorf("invalid input")
		}
		uploads = append(uploads, request.ItemPhotoUploadInput{
			File:     readSeeker,
			FileName: header.Filename,
		})
	}

	return uploads, func() { closeMultipartFiles(files) }, nil
}

func closeMultipartFiles(files []multipart.File) {
	for _, file := range files {
		_ = file.Close()
	}
}
