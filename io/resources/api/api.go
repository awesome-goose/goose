package api

import (
	"strconv"

	"github.com/awesome-goose/goose/io/output"
	"github.com/awesome-goose/goose/types"
)

// Resource is a generic controller for working with typed entities
// It provides standard CRUD operations for RESTful API endpoints
type Resource[T any] struct {
	entity types.Entity[T]

	log types.Log `inject:""`
}

// Hydrate initializes the Resource with configuration and entity
func (r *Resource[T]) Hydrate(
	entity types.Entity[T],
) {
	r.entity = entity
}

// Create handles POST /resources - creates a new entity
func (r *Resource[T]) Create(dto *T) types.Output {
	if err := r.entity.Insert(dto); err != nil {
		r.log.Error("Failed to create entity", "error", err)
		return output.InternalServerError("Failed to create resource")
	}

	return output.Created(dto)
}

// List handles GET /resources - returns a paginated list of entities
// Query params: page (default: 1), per_page (default: 10), search, sort
func (r *Resource[T]) List(dto *ListDto) types.Output {
	// Parse pagination params
	page, _ := strconv.Atoi(dto.Page)
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(dto.PerPage)
	if perPage < 1 {
		perPage = 10
	}

	// Build query from search params
	queries := map[string]string{
		"page":     dto.Page,
		"per_page": dto.PerPage,
		"from":     dto.From,
		"to":       dto.To,
		"query":    dto.Query,
	}
	query, args := r.entity.BuildQuery(queries)

	// Get paginated results
	entities, err := r.entity.Page(page, perPage, query, args...)
	if err != nil {
		r.log.Error("Failed to fetch entities", "error", err)
		return output.InternalServerError("Failed to fetch resources")
	}

	// Get total count for pagination meta
	total, err := r.entity.Count(query, args...)
	if err != nil {
		r.log.Error("Failed to count entities", "error", err)
		return output.InternalServerError("Failed to count resources")
	}

	return output.Paginated(entities, page, perPage, total)
}

// Get handles GET /resources/{id} - returns a single entity by ID
func (r *Resource[T]) Get(dto *GetDto) types.Output {
	if dto.ID == "" {
		return output.BadRequest("Resource ID is required")
	}

	entity, err := r.entity.First(dto.ID)
	if err != nil {
		r.log.Error("Failed to fetch entity", "id", dto.ID, "error", err)
		return output.NotFound("Resource not found")
	}

	return output.OK(entity)
}

// Update handles PUT/PATCH /resources/{id} - updates an existing entity
func (r *Resource[T]) Update(dto *UpdateDto[T]) types.Output {
	if dto.ID == "" {
		return output.BadRequest("Resource ID is required")
	}

	// Check if entity exists
	exists, err := r.entity.Exists(dto.ID)
	if err != nil {
		r.log.Error("Failed to check entity existence", "id", dto.ID, "error", err)
		return output.InternalServerError("Failed to verify resource")
	}
	if !exists {
		return output.NotFound("Resource not found")
	}

	rowsAffected, err := r.entity.Update(&dto.Body, dto.ID)
	if err != nil {
		r.log.Error("Failed to update entity", "id", dto.ID, "error", err)
		return output.InternalServerError("Failed to update resource")
	}

	// Fetch updated entity
	updatedEntity, err := r.entity.First(dto.ID)
	if err != nil {
		r.log.Error("Failed to fetch updated entity", "id", dto.ID, "error", err)
		return output.InternalServerError("Failed to retrieve updated resource")
	}

	return output.SuccessWithMeta("Resource updated", updatedEntity, map[string]any{
		"rows_affected": rowsAffected,
	})
}

// Delete handles DELETE /resources/{id} - deletes an entity
func (r *Resource[T]) Delete(dto *DeleteDto) types.Output {
	if dto.ID == "" {
		return output.BadRequest("Resource ID is required")
	}

	rowsAffected, err := r.entity.Delete(dto.ID)
	if err != nil {
		r.log.Error("Failed to delete entity", "id", dto.ID, "error", err)
		return output.InternalServerError("Failed to delete resource")
	}

	if rowsAffected == 0 {
		return output.NotFound("Resource not found")
	}

	return output.Success("Resource deleted", map[string]any{
		"deleted": true,
		"id":      dto.ID,
	})
}
