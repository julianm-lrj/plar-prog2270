package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// InventoryLog represents a record of inventory changes for audit trail
type InventoryLog struct {
	ID              bson.ObjectID `bson:"_id,omitempty" json:"id"`
	ProductID       bson.ObjectID `bson:"product_id" json:"product_id" validate:"required"`
	SKU             string        `bson:"sku" json:"sku" validate:"required,min=3,max=50"`
	Warehouse       string        `bson:"warehouse" json:"warehouse" validate:"required,min=2,max=100"`
	ChangeType      string        `bson:"change_type" json:"change_type" validate:"required,oneof=adjustment purchase sale return damage lost recount transfer"`
	QuantityBefore  int           `bson:"quantity_before" json:"quantity_before" validate:"gte=0"`
	QuantityAfter   int           `bson:"quantity_after" json:"quantity_after" validate:"gte=0"`
	QuantityChanged int           `bson:"quantity_changed" json:"quantity_changed"` // Can be positive or negative
	Reason          string        `bson:"reason" json:"reason" validate:"required,min=5,max=500"`
	PerformedBy     string        `bson:"performed_by" json:"performed_by" validate:"required,min=2,max=100"` // User ID or system name
	Notes           string        `bson:"notes,omitempty" json:"notes,omitempty" validate:"max=1000"`
	CreatedAt       time.Time     `bson:"created_at" json:"created_at"`
}

// SetTimestamp sets the creation timestamp
func (il *InventoryLog) SetTimestamp() {
	if il.CreatedAt.IsZero() {
		il.CreatedAt = time.Now()
	}
}

// CalculateQuantityChanged calculates the difference between before and after
func (il *InventoryLog) CalculateQuantityChanged() {
	il.QuantityChanged = il.QuantityAfter - il.QuantityBefore
}

// IsIncrease returns true if inventory increased
func (il *InventoryLog) IsIncrease() bool {
	return il.QuantityChanged > 0
}

// IsDecrease returns true if inventory decreased
func (il *InventoryLog) IsDecrease() bool {
	return il.QuantityChanged < 0
}

// GetAbsoluteChange returns the absolute value of quantity changed
func (il *InventoryLog) GetAbsoluteChange() int {
	if il.QuantityChanged < 0 {
		return -il.QuantityChanged
	}
	return il.QuantityChanged
}

// IsSystemGenerated checks if the log was created by an automated system
func (il *InventoryLog) IsSystemGenerated() bool {
	return il.PerformedBy == "system" || il.PerformedBy == "auto"
}

// GetChangeDescription returns a human-readable description of the change
func (il *InventoryLog) GetChangeDescription() string {
	direction := "unchanged"
	if il.IsIncrease() {
		direction = "increased"
	} else if il.IsDecrease() {
		direction = "decreased"
	}

	return direction + " by " + string(rune(il.GetAbsoluteChange())) + " units"
}
