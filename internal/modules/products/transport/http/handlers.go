package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Lovealone1/nex21-api/internal/modules/products/repo"
	"github.com/Lovealone1/nex21-api/internal/modules/products/service"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	"github.com/Lovealone1/nex21-api/shared/pagination"
)

// ProductListResponse represents a paginated list of products
type ProductListResponse struct {
	Items      []repo.Product `json:"items"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	Limit      int            `json:"limit"`
	TotalPages int64          `json:"total_pages"`
}

type ProductHandler struct {
	svc service.ProductService
}

func NewProductHandler(svc service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

// RegisterRoutes sets up the REST endpoints for products.
// This is typically mounted onto the router UNDER `/api/admin/v1/tenants/{tenantId}/products`
func (h *ProductHandler) RegisterRoutes(r chi.Router) {
	r.Post("/", h.CreateProduct)
	r.Get("/", h.ListProducts)
	r.Get("/{id}", h.GetProductByID)
	r.Patch("/{id}", h.UpdateProduct)
	r.Delete("/{id}", h.DeleteProduct)
	r.Patch("/{id}/status", h.ToggleStatus)
	r.Post("/{id}/stock", h.SetStock)
}

// CreateProduct
// @Summary      Create a new Product
// @Description  Creates a new product in the specified tenant workspace.
// @Tags         Products
// @Accept       json
// @Produce      json
// @Param        tenantId path      string                     true  "Tenant UUID"
// @Param        request  body      service.CreateProductInput true  "Product data"
// @Success      201      {object}  repo.Product
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/products [post]
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	var req service.CreateProductInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "ProductHandler.Create", "Invalid JSON format"))
		return
	}

	p, err := h.svc.CreateProduct(r.Context(), tenantID, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(p)
}

// GetProductByID
// @Summary      Get a Product by ID
// @Description  Returns the product details if found inside the tenant.
// @Tags         Products
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        id       path      string  true  "Product UUID"
// @Success      200      {object}  repo.Product
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/products/{id} [get]
func (h *ProductHandler) GetProductByID(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	p, err := h.svc.GetProductByID(r.Context(), tenantID, id)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(p)
}

// UpdateProduct
// @Summary      Update a Product
// @Description  Applies a partial update to a product. Quantity cannot be updated here.
// @Tags         Products
// @Accept       json
// @Produce      json
// @Param        tenantId path      string                     true  "Tenant UUID"
// @Param        id       path      string                     true  "Product UUID"
// @Param        request  body      service.UpdateProductInput true  "Fields to update"
// @Success      200      {object}  repo.Product
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/products/{id} [patch]
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	var req service.UpdateProductInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "ProductHandler.Update", "Invalid JSON format"))
		return
	}

	p, err := h.svc.UpdateProduct(r.Context(), tenantID, id, req)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(p)
}

// DeleteProduct
// @Summary      Delete a Product
// @Description  Permanently deletes a product from the tenant.
// @Tags         Products
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        id       path      string  true  "Product UUID"
// @Success      200      {object}  map[string]string
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/products/{id} [delete]
func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	if err := h.svc.DeleteProduct(r.Context(), tenantID, id); err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Product deleted successfully",
	})
}

// ListProducts
// @Summary      List all Products for a Tenant
// @Description  Returns a paginated list of products.
// @Tags         Products
// @Produce      json
// @Param        tenantId     path      string  true   "Tenant UUID"
// @Param        page         query     int     false  "Page number (1-based, default: 1)"
// @Param        limit        query     int     false  "Records per page (default: 20, max: 100)"
// @Param        sort_by      query     string  false  "Sort column: created_at | name | price | quantity | is_active | sku"
// @Param        sort_dir     query     string  false  "Sort direction: ASC | DESC (default: DESC)"
// @Success      200          {object}  ProductListResponse
// @Failure      400          {object}  errors.HTTPErrorResponse
// @Failure      401          {object}  errors.HTTPErrorResponse
// @Failure      500          {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/products [get]
func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	page := pagination.ParseRequest(r,
		"created_at", "updated_at", "name", "price", "is_active", "sku", "quantity",
	)

	result, err := h.svc.ListProducts(r.Context(), tenantID, page)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	resp := pagination.NewResponse(result, page)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// ToggleStatus
// @Summary      Toggle Product Status
// @Description  Automatically flips a product's is_active status boolean.
// @Tags         Products
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        id       path      string  true  "Product UUID"
// @Success      200      {object}  repo.Product
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/products/{id}/status [patch]
func (h *ProductHandler) ToggleStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	p, err := h.svc.ToggleStatus(r.Context(), tenantID, id)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(p)
}

// SetStock
// @Summary      Set Product Stock
// @Description  Manually overrides the stock quantity of a product.
// @Tags         Products
// @Accept       json
// @Produce      json
// @Param        tenantId path      string  true  "Tenant UUID"
// @Param        id       path      string  true  "Product UUID"
// @Param        request  body      service.UpdateStockInput true "Stock quantity"
// @Success      200      {object}  repo.Product
// @Failure      400      {object}  errors.HTTPErrorResponse
// @Failure      401      {object}  errors.HTTPErrorResponse
// @Failure      404      {object}  errors.HTTPErrorResponse
// @Failure      500      {object}  errors.HTTPErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/v1/tenants/{tenantId}/products/{id}/stock [post]
func (h *ProductHandler) SetStock(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	id := chi.URLParam(r, "id")

	var req service.UpdateStockInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteHTTPError(w, errors.New(errors.InvalidArgument, "ProductHandler.SetStock", "Invalid JSON format"))
		return
	}

	p, err := h.svc.SetStock(r.Context(), tenantID, id, req.Quantity)
	if err != nil {
		errors.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(p)
}
