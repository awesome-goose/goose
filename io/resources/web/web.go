package web

import (
	"fmt"
	"strconv"

	"github.com/awesome-goose/goose/io/output"
	"github.com/awesome-goose/goose/types"
)

// Resource is a generic controller for rendering HTML templates for typed entities
// It provides standard CRUD operations with HTML template rendering
// Routes follow RESTful conventions:
//   - GET    /resources          -> Index   (resources.index)
//   - GET    /resources/create   -> Create  (resources.create)
//   - POST   /resources          -> Store   (resources.store)
//   - GET    /resources/{id}     -> Show    (resources.show)
//   - GET    /resources/{id}/edit-> Edit    (resources.edit)
//   - PUT    /resources/{id}     -> Update  (resources.update)
//   - DELETE /resources/{id}     -> Destroy (resources.destroy)
type Resource[T any] struct {
	entity       types.Entity[T]
	resourceName string // e.g., "photos" - used for template paths

	log types.Log `inject:""`
}

// Hydrate initializes the Resource with configuration and entity
func (r *Resource[T]) Hydrate(
	entity types.Entity[T],
	resourceName string,
) {
	r.entity = entity
	r.resourceName = resourceName
}

// templatePath returns the path to a template for this resource
// Templates are expected at: /templates/pages/[resource name]/[template].html
func (r *Resource[T]) templatePath(template string) string {
	return fmt.Sprintf("pages/%s/%s.html", r.resourceName, template)
}

// redirectPath returns the redirect path for this resource
func (r *Resource[T]) redirectPath(suffix ...string) string {
	path := fmt.Sprintf("/%s", r.resourceName)
	if len(suffix) > 0 {
		path = fmt.Sprintf("%s/%s", path, suffix[0])
	}
	return path
}

// Index handles GET /resources - displays a paginated list of entities
// Renders: /templates/pages/[resource]/index.html
func (r *Resource[T]) Index(dto *IndexDto) types.Output {
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
		return r.errorView("Failed to fetch resources", err)
	}

	// Get total count for pagination meta
	total, err := r.entity.Count(query, args...)
	if err != nil {
		r.log.Error("Failed to count entities", "error", err)
		return r.errorView("Failed to count resources", err)
	}

	// Calculate pagination meta
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))

	return output.View(r.templatePath("index"), map[string]any{
		"items":       entities,
		"page":        page,
		"perPage":     perPage,
		"total":       total,
		"totalPages":  totalPages,
		"hasMore":     page < totalPages,
		"hasPrevious": page > 1,
		"query":       dto.Query,
	})
}

// Create handles GET /resources/create - displays a form to create a new entity
// Renders: /templates/pages/[resource]/create.html
func (r *Resource[T]) Create(dto *CreateDto) types.Output {
	return output.View(r.templatePath("create"), map[string]any{
		"item": new(T), // Empty item for form binding
	})
}

// Store handles POST /resources - creates a new entity and redirects
// Redirects to: /resources/{id} or /resources on success
func (r *Resource[T]) Store(dto *StoreDto[T]) types.Output {
	if err := r.entity.Insert(&dto.Body); err != nil {
		r.log.Error("Failed to create entity", "error", err)
		return output.View(r.templatePath("create"), map[string]any{
			"item":  &dto.Body,
			"error": "Failed to create resource: " + err.Error(),
		}, output.WithHTMLCode(422))
	}

	return output.Redirect(
		r.redirectPath(),
		output.WithFlash("success", "Resource created successfully"),
	)
}

// Show handles GET /resources/{id} - displays a single entity
// Renders: /templates/pages/[resource]/show.html
func (r *Resource[T]) Show(dto *ShowDto) types.Output {
	if dto.ID == "" {
		return r.errorView("Resource ID is required", nil)
	}

	entity, err := r.entity.First(dto.ID)
	if err != nil {
		r.log.Error("Failed to fetch entity", "id", dto.ID, "error", err)
		return r.notFoundView()
	}

	return output.View(r.templatePath("show"), map[string]any{
		"item": entity,
	})
}

// Edit handles GET /resources/{id}/edit - displays a form to edit an entity
// Renders: /templates/pages/[resource]/edit.html
func (r *Resource[T]) Edit(dto *EditDto) types.Output {
	if dto.ID == "" {
		return r.errorView("Resource ID is required", nil)
	}

	entity, err := r.entity.First(dto.ID)
	if err != nil {
		r.log.Error("Failed to fetch entity", "id", dto.ID, "error", err)
		return r.notFoundView()
	}

	return output.View(r.templatePath("edit"), map[string]any{
		"item": entity,
	})
}

// Update handles PUT/PATCH /resources/{id} - updates an entity and redirects
// Redirects to: /resources/{id} on success
func (r *Resource[T]) Update(dto *UpdateDto[T]) types.Output {
	if dto.ID == "" {
		return r.errorView("Resource ID is required", nil)
	}

	// Check if entity exists
	exists, err := r.entity.Exists(dto.ID)
	if err != nil {
		r.log.Error("Failed to check entity existence", "id", dto.ID, "error", err)
		return r.errorView("Failed to verify resource", err)
	}
	if !exists {
		return r.notFoundView()
	}

	_, err = r.entity.Update(&dto.Body, dto.ID)
	if err != nil {
		r.log.Error("Failed to update entity", "id", dto.ID, "error", err)
		// Re-render edit form with error
		return output.View(r.templatePath("edit"), map[string]any{
			"item":  &dto.Body,
			"error": "Failed to update resource: " + err.Error(),
		}, output.WithHTMLCode(422))
	}

	return output.Redirect(
		r.redirectPath(dto.ID),
		output.WithFlash("success", "Resource updated successfully"),
	)
}

// Destroy handles DELETE /resources/{id} - deletes an entity and redirects
// Redirects to: /resources on success
func (r *Resource[T]) Destroy(dto *DestroyDto) types.Output {
	if dto.ID == "" {
		return r.errorView("Resource ID is required", nil)
	}

	rowsAffected, err := r.entity.Delete(dto.ID)
	if err != nil {
		r.log.Error("Failed to delete entity", "id", dto.ID, "error", err)
		return output.Redirect(
			r.redirectPath(),
			output.WithFlash("error", "Failed to delete resource"),
		)
	}

	if rowsAffected == 0 {
		return output.Redirect(
			r.redirectPath(),
			output.WithFlash("error", "Resource not found"),
		)
	}

	return output.Redirect(
		r.redirectPath(),
		output.WithFlash("success", "Resource deleted successfully"),
	)
}

// errorView renders an error page
func (r *Resource[T]) errorView(message string, err error) types.Output {
	errMsg := message
	if err != nil {
		errMsg = fmt.Sprintf("%s: %v", message, err)
	}
	return output.View("errors/error.html", map[string]any{
		"message": errMsg,
	}, output.WithHTMLCode(500))
}

// notFoundView renders a 404 page
func (r *Resource[T]) notFoundView() types.Output {
	return output.View("errors/404.html", map[string]any{
		"message": "Resource not found",
	}, output.WithHTMLCode(404))
}
