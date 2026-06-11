package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"
	"pack_mate/internal/dto/response"
	"pack_mate/internal/service"

	"github.com/gin-gonic/gin"
)

const checklistResponseDateLayout = "2006-01-02"

// ChecklistHandler handles checklist HTTP requests.
type ChecklistHandler struct {
	svc service.ChecklistService
}

// NewChecklistHandler creates a checklist handler.
func NewChecklistHandler(svc service.ChecklistService) *ChecklistHandler {
	return &ChecklistHandler{svc: svc}
}

// CreateChecklist handles checklist creation requests.
func (h *ChecklistHandler) CreateChecklist(c *gin.Context) {
	var input request.CreateChecklistInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	input.UserID = strings.TrimSpace(input.UserID)
	// TODO: Read current user from auth context after user accounts are implemented.
	if !validateOptionalObjectID(input.UserID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is invalid"})
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if strings.TrimSpace(input.TargetDate) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_date is required"})
		return
	}

	checklist, err := h.svc.CreateChecklist(c.Request.Context(), input)
	if err != nil {
		respondChecklistError(c, err)
		return
	}

	c.JSON(http.StatusOK, buildChecklistResponse(checklist))
}

// ListChecklists handles checklist list requests.
func (h *ChecklistHandler) ListChecklists(c *gin.Context) {
	userID := strings.TrimSpace(c.Query("user_id"))
	// TODO: Read current user from auth context after user accounts are implemented.
	if !validateOptionalObjectID(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is invalid"})
		return
	}

	q, hasQ := c.GetQuery("q")
	q = strings.TrimSpace(q)
	if hasQ && q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "q is required"})
		return
	}

	checklists, err := h.svc.ListChecklists(c.Request.Context(), request.ListChecklistsInput{
		UserID: userID,
		Q:      q,
		HasQ:   hasQ,
	})
	if err != nil {
		respondChecklistError(c, err)
		return
	}

	responses := make([]response.ChecklistResponse, 0, len(checklists))
	for _, checklist := range checklists {
		responses = append(responses, buildChecklistResponse(&checklist))
	}

	c.JSON(http.StatusOK, response.ListChecklistsResponse{Checklists: responses})
}

// GetChecklist handles checklist detail requests.
func (h *ChecklistHandler) GetChecklist(c *gin.Context) {
	checklistID := strings.TrimSpace(c.Param("checklist_id"))
	if checklistID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "checklist_id is required"})
		return
	}
	if !validateRequiredObjectID(checklistID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "checklist_id is invalid"})
		return
	}

	checklist, err := h.svc.GetChecklist(c.Request.Context(), checklistID)
	if err != nil {
		respondChecklistError(c, err)
		return
	}

	c.JSON(http.StatusOK, buildChecklistResponse(checklist))
}

// UpdateChecklist handles checklist update requests.
func (h *ChecklistHandler) UpdateChecklist(c *gin.Context) {
	checklistID := strings.TrimSpace(c.Param("checklist_id"))
	if checklistID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "checklist_id is required"})
		return
	}
	if !validateRequiredObjectID(checklistID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "checklist_id is invalid"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	if _, ok := raw["items"]; ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "items cannot be updated here"})
		return
	}
	var input request.UpdateChecklistInput
	if err := json.Unmarshal(body, &input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if strings.TrimSpace(input.TargetDate) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_date is required"})
		return
	}

	checklist, err := h.svc.UpdateChecklist(c.Request.Context(), checklistID, input)
	if err != nil {
		respondChecklistError(c, err)
		return
	}

	c.JSON(http.StatusOK, buildChecklistResponse(checklist))
}

// AddChecklistLineItems handles checklist line item add requests.
func (h *ChecklistHandler) AddChecklistLineItems(c *gin.Context) {
	checklistID := strings.TrimSpace(c.Param("checklist_id"))
	if checklistID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "checklist_id is required"})
		return
	}
	if !validateRequiredObjectID(checklistID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "checklist_id is invalid"})
		return
	}

	var input request.AddChecklistLineItemsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	if len(input.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "items are required"})
		return
	}

	checklist, err := h.svc.AddChecklistLineItems(c.Request.Context(), checklistID, input)
	if err != nil {
		respondChecklistError(c, err)
		return
	}

	c.JSON(http.StatusOK, buildChecklistResponse(checklist))
}

// RemoveChecklistLineItems handles checklist line item remove requests.
func (h *ChecklistHandler) RemoveChecklistLineItems(c *gin.Context) {
	checklistID := strings.TrimSpace(c.Param("checklist_id"))
	if checklistID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "checklist_id is required"})
		return
	}
	if !validateRequiredObjectID(checklistID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "checklist_id is invalid"})
		return
	}

	var input request.RemoveChecklistLineItemsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	if len(input.LineItemIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "line_item_ids are required"})
		return
	}

	checklist, err := h.svc.RemoveChecklistLineItems(c.Request.Context(), checklistID, input)
	if err != nil {
		respondChecklistError(c, err)
		return
	}

	c.JSON(http.StatusOK, buildChecklistResponse(checklist))
}

// UpdateChecklistLineItemStatus handles checklist line item status update requests.
func (h *ChecklistHandler) UpdateChecklistLineItemStatus(c *gin.Context) {
	checklistID := strings.TrimSpace(c.Param("checklist_id"))
	if checklistID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "checklist_id is required"})
		return
	}
	if !validateRequiredObjectID(checklistID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "checklist_id is invalid"})
		return
	}

	lineItemID := strings.TrimSpace(c.Param("line_item_id"))
	if lineItemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "line_item_id is required"})
		return
	}
	if !validateRequiredObjectID(lineItemID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "line_item_id is invalid"})
		return
	}

	var input request.UpdateChecklistLineItemStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	if strings.TrimSpace(input.Status) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
		return
	}

	checklist, err := h.svc.UpdateChecklistLineItemStatus(c.Request.Context(), checklistID, lineItemID, input)
	if err != nil {
		respondChecklistError(c, err)
		return
	}

	c.JSON(http.StatusOK, buildChecklistResponse(checklist))
}

// DeleteChecklist handles checklist deletion requests.
func (h *ChecklistHandler) DeleteChecklist(c *gin.Context) {
	checklistID := strings.TrimSpace(c.Param("checklist_id"))
	if checklistID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "checklist_id is required"})
		return
	}
	if !validateRequiredObjectID(checklistID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "checklist_id is invalid"})
		return
	}

	if err := h.svc.DeleteChecklist(c.Request.Context(), checklistID); err != nil {
		respondChecklistError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.DeleteChecklistResponse{Deleted: true})
}

func respondChecklistError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := err.Error()
	switch {
	case strings.Contains(message, "invalid input"),
		strings.Contains(message, "items cannot be updated here"),
		strings.Contains(message, "checklist name is required"),
		strings.Contains(message, "checklist target_date is required"),
		strings.Contains(message, "checklist line items are required"),
		strings.Contains(message, "checklist line item ids are required"),
		strings.Contains(message, "checklist line item status is required"),
		strings.Contains(message, "checklist line item status is invalid"),
		strings.Contains(message, "checklist line item reference item not found"),
		strings.Contains(message, "checklist search keyword is required"),
		strings.Contains(message, "checklist search keyword is too long"):
		status = http.StatusBadRequest
	case strings.Contains(message, "checklist line item not found"),
		strings.Contains(message, "checklist not found"):
		status = http.StatusNotFound
	case strings.Contains(message, "create checklist failed"),
		strings.Contains(message, "list checklists failed"),
		strings.Contains(message, "get checklist failed"),
		strings.Contains(message, "get checklist line item reference item failed"),
		strings.Contains(message, "update checklist failed"),
		strings.Contains(message, "update checklist line item status failed"),
		strings.Contains(message, "delete checklist failed"):
		status = http.StatusInternalServerError
	}
	c.JSON(status, gin.H{"error": message})
}

func buildChecklistResponse(checklist *domain.Checklist) response.ChecklistResponse {
	items := make([]response.ChecklistLineItemResponse, 0, len(checklist.Items))
	for _, item := range checklist.Items {
		referenceID := ""
		if !item.ReferenceID.IsZero() {
			referenceID = item.ReferenceID.Hex()
		}
		var snapshot *response.ItemSnapshotResponse
		if item.Snapshot != nil {
			snapshot = &response.ItemSnapshotResponse{Name: item.Snapshot.Name}
		}
		items = append(items, response.ChecklistLineItemResponse{
			ID:            item.ID.Hex(),
			ReferenceID:   referenceID,
			ReferenceType: string(item.ReferenceType),
			Snapshot:      snapshot,
			Status:        string(item.Status),
		})
	}

	return response.ChecklistResponse{
		ID:          checklist.ID.Hex(),
		UserID:      checklist.UserID.Hex(),
		Name:        checklist.Name,
		Description: checklist.Description,
		TargetDate:  checklist.TargetDate.Format(checklistResponseDateLayout),
		Items:       items,
		Status:      string(checklist.Status),
	}
}
