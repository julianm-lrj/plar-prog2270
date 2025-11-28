package models

import (
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Stock represents inventory levels across different warehouses
type Stock struct {
	WarehouseMain int `json:"warehouse_main" bson:"warehouse_main" validate:"gte=0"`
	WarehouseEast int `json:"warehouse_east" bson:"warehouse_east" validate:"gte=0"`
	WarehouseWest int `json:"warehouse_west" bson:"warehouse_west" validate:"gte=0"`
	Total         int `json:"total" bson:"total" validate:"gte=0"`
}

// Ratings represents product review statistics
type Ratings struct {
	Average float64 `json:"average" bson:"average" validate:"gte=0,lte=5"`
	Count   int     `json:"count" bson:"count" validate:"gte=0"`
}

// Product represents an e-commerce product in the catalog
type Product struct {
	ID          bson.ObjectID     `json:"id" bson:"_id,omitempty"`
	SKU         string            `json:"sku" bson:"sku" validate:"required,min=3,max=50"`
	Name        string            `json:"name" bson:"name" validate:"required,min=2,max=200"`
	Description string            `json:"description" bson:"description" validate:"max=2000"`
	Category    string            `json:"category" bson:"category" validate:"required,min=2,max=100"`
	Subcategory string            `json:"subcategory" bson:"subcategory" validate:"max=100"`
	Brand       string            `json:"brand" bson:"brand" validate:"required,min=2,max=100"`
	Price       float64           `json:"price" bson:"price" validate:"required,gt=0"`
	Currency    string            `json:"currency" bson:"currency" validate:"required,len=3"` // CAD, USD, etc.
	Stock       Stock             `json:"stock" bson:"stock"`
	Attributes  map[string]string `json:"attributes" bson:"attributes"` // Flexible key-value pairs
	Images      []string          `json:"images" bson:"images" validate:"dive,url"`
	Ratings     Ratings           `json:"ratings" bson:"ratings"`
	Tags        []string          `json:"tags" bson:"tags" validate:"dive,min=2,max=50"`
	Status      string            `json:"status" bson:"status" validate:"required,oneof=active inactive deleted"`
	CreatedAt   time.Time         `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" bson:"updated_at"`
}

func (p *Product) CalculateTotalStock() {
	p.Stock.Total = p.Stock.WarehouseMain + p.Stock.WarehouseEast + p.Stock.WarehouseWest
}

type CreateProductRequest struct {
	Name        string            `json:"name" validate:"required,min=2,max=200"`
	Description string            `json:"description" validate:"max=2000"`
	Category    string            `json:"category" validate:"required,min=2,max=100"`
	Subcategory string            `json:"subcategory" validate:"max=100"`
	Brand       string            `json:"brand" validate:"required,min=2,max=100"`
	Price       float64           `json:"price" validate:"required,gt=0"`
	Currency    string            `json:"currency" validate:"required,len=3"`
	Images      []string          `json:"images" validate:"dive,url"`
	Attributes  map[string]string `json:"attributes"`
	Tags        []string          `json:"tags" validate:"dive,min=2,max=50"`
}

func (req *CreateProductRequest) GenerateSKU() string {
	brandPrefix := strings.ToUpper(req.Brand[:min(3, len(req.Brand))])
	categoryPrefix := strings.ToUpper(req.Category[:min(3, len(req.Category))])
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s-%s-%d", brandPrefix, categoryPrefix, timestamp)
}

func (req *CreateProductRequest) ToProduct() *Product {
	now := time.Now()
	product := &Product{
		ID:          bson.NewObjectID(),
		SKU:         req.GenerateSKU(),
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Subcategory: req.Subcategory,
		Brand:       req.Brand,
		Price:       req.Price,
		Currency:    req.Currency,
		Stock:       Stock{WarehouseMain: 0, WarehouseEast: 0, WarehouseWest: 0, Total: 0},
		Attributes:  req.Attributes,
		Images:      req.Images,
		Ratings:     Ratings{Average: 0.0, Count: 0},
		Tags:        req.Tags,
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if product.Attributes == nil {
		product.Attributes = make(map[string]string)
	}
	return product
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (p *Product) IsInStock() bool {
	return p.Stock.Total > 0 && p.Status == "active"
}

func (p *Product) IsLowStock(threshold int) bool {
	return p.Stock.Total <= threshold && p.Stock.Total > 0
}

func (p *Product) SetTimestamps() {
	now := time.Now()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
}
