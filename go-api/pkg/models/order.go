package models

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type CreateOrderRequest struct {
	CustomerID      bson.ObjectID `json:"customer_id" bson:"customer_id" validate:"required"`
	CustomerEmail   string        `json:"customer_email" bson:"customer_email" validate:"required,email"`
	Items           []OrderItem   `json:"items" bson:"items" validate:"required,min=1,dive"`
	ShippingAddress Address       `json:"shipping_address" bson:"shipping_address" validate:"required"`
	BillingAddress  *Address      `json:"billing_address" bson:"billing_address,omitempty"`
	Payment         Payment       `json:"payment" bson:"payment" validate:"required"`
	Notes           string        `json:"notes" bson:"notes,omitempty"`
}

// OrderItem represents a single item in an order
type OrderItem struct {
	ProductID bson.ObjectID `json:"product_id" bson:"product_id" validate:"required"`
	SKU       string        `json:"sku" bson:"sku" validate:"required"`
	Name      string        `json:"name" bson:"name" validate:"required"`
	Quantity  int           `json:"quantity" bson:"quantity" validate:"required,gte=1"`
	UnitPrice float64       `json:"unit_price" bson:"unit_price" validate:"required,gt=0"`
	Subtotal  float64       `json:"subtotal" bson:"subtotal" validate:"required,gte=0"`
}

// Address represents shipping or billing address
type Address struct {
	Street     string `json:"street" bson:"street" validate:"required"`
	City       string `json:"city" bson:"city" validate:"required"`
	Province   string `json:"province" bson:"province" validate:"required,len=2"` // ON, BC, etc.
	PostalCode string `json:"postal_code" bson:"postal_code" validate:"required"`
	Country    string `json:"country" bson:"country" validate:"required"`
	IsDefault  bool   `json:"is_default" bson:"is_default"`
}

// OrderTotals represents the financial breakdown of an order
type OrderTotals struct {
	Subtotal   float64 `json:"subtotal" bson:"subtotal" validate:"gte=0"`
	Tax        float64 `json:"tax" bson:"tax" validate:"gte=0"`
	Shipping   float64 `json:"shipping" bson:"shipping" validate:"gte=0"`
	Discount   float64 `json:"discount" bson:"discount" validate:"gte=0"`
	GrandTotal float64 `json:"grand_total" bson:"grand_total" validate:"gt=0"`
}

// Payment represents payment information for an order
type Payment struct {
	Method        string `json:"method" bson:"method" validate:"required,oneof=credit_card debit_card paypal cash"`
	Status        string `json:"status" bson:"status" validate:"required,oneof=pending completed failed refunded"`
	TransactionID string `json:"transaction_id" bson:"transaction_id"`
}

// Timeline tracks the lifecycle of an order
type Timeline struct {
	OrderedAt         time.Time  `json:"ordered_at" bson:"ordered_at"`
	PaidAt            *time.Time `json:"paid_at" bson:"paid_at,omitempty"`
	ShippedAt         *time.Time `json:"shipped_at" bson:"shipped_at,omitempty"`
	DeliveredAt       *time.Time `json:"delivered_at" bson:"delivered_at,omitempty"`
	CancelledAt       *time.Time `json:"cancelled_at" bson:"cancelled_at,omitempty"`
	EstimatedDelivery *time.Time `json:"estimated_delivery" bson:"estimated_delivery,omitempty"`
}

// Order represents a customer order in the e-commerce system
type Order struct {
	ID              bson.ObjectID `json:"id" bson:"_id,omitempty"`
	OrderNumber     string        `json:"order_number" bson:"order_number" validate:"required"`
	CustomerID      bson.ObjectID `json:"customer_id" bson:"customer_id" validate:"required"`
	CustomerEmail   string        `json:"customer_email" bson:"customer_email" validate:"required,email"`
	Status          string        `json:"status" bson:"status" validate:"required,oneof=pending processing shipped delivered cancelled"`
	Items           []OrderItem   `json:"items" bson:"items" validate:"required,min=1,dive"`
	Totals          OrderTotals   `json:"totals" bson:"totals"`
	ShippingAddress Address       `json:"shipping_address" bson:"shipping_address"`
	BillingAddress  *Address      `json:"billing_address" bson:"billing_address,omitempty"`
	Payment         Payment       `json:"payment" bson:"payment"`
	Timeline        Timeline      `json:"timeline" bson:"timeline"`
	Notes           string        `json:"notes" bson:"notes,omitempty"`
	CreatedAt       time.Time     `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at" bson:"updated_at"`
}

// CalculateItemSubtotal calculates subtotal for a single order item
func (oi *OrderItem) CalculateItemSubtotal() {
	oi.Subtotal = oi.UnitPrice * float64(oi.Quantity)
}

// CalculateTotals calculates all order totals (subtotal, tax, shipping, grand total)
// Tax rate is 13% (Ontario HST), shipping is flat $15
func (o *Order) CalculateTotals() {
	// Calculate subtotal from items
	var subtotal float64
	for _, item := range o.Items {
		subtotal += item.Subtotal
	}
	o.Totals.Subtotal = subtotal

	// Calculate tax (13% HST for Ontario)
	o.Totals.Tax = subtotal * 0.13

	// Set shipping (flat rate or free over $100)
	if subtotal >= 100 {
		o.Totals.Shipping = 0.00
	} else {
		o.Totals.Shipping = 15.00
	}

	// Calculate grand total
	o.Totals.GrandTotal = o.Totals.Subtotal + o.Totals.Tax + o.Totals.Shipping - o.Totals.Discount
}

// CalculateAllTotals recalculates item subtotals and order totals
func (o *Order) CalculateAllTotals() {
	// First calculate each item's subtotal
	for i := range o.Items {
		o.Items[i].CalculateItemSubtotal()
	}
	// Then calculate order totals
	o.CalculateTotals()
}

// SetTimestamps sets created_at and updated_at timestamps
func (o *Order) SetTimestamps() {
	now := time.Now()
	if o.CreatedAt.IsZero() {
		o.CreatedAt = now
		o.Timeline.OrderedAt = now
	}
	o.UpdatedAt = now
}

// UpdateStatus updates the order status and timeline accordingly
func (o *Order) UpdateStatus(newStatus string) {
	o.Status = newStatus
	now := time.Now()

	switch newStatus {
	case "processing":
		if o.Timeline.PaidAt == nil {
			o.Timeline.PaidAt = &now
		}
	case "shipped":
		if o.Timeline.ShippedAt == nil {
			o.Timeline.ShippedAt = &now
		}
		// Set estimated delivery to 5 days from now
		estimatedDelivery := now.AddDate(0, 0, 5)
		o.Timeline.EstimatedDelivery = &estimatedDelivery
	case "delivered":
		if o.Timeline.DeliveredAt == nil {
			o.Timeline.DeliveredAt = &now
		}
	case "cancelled":
		if o.Timeline.CancelledAt == nil {
			o.Timeline.CancelledAt = &now
		}
	}

	o.UpdatedAt = now
}

// GetItemCount returns the total number of items in the order
func (o *Order) GetItemCount() int {
	var count int
	for _, item := range o.Items {
		count += item.Quantity
	}
	return count
}

// HasBeenPaid checks if payment has been completed
func (o *Order) HasBeenPaid() bool {
	return o.Payment.Status == "completed"
}

// CanBeCancelled checks if the order can still be cancelled
func (o *Order) CanBeCancelled() bool {
	return o.Status == "pending" || o.Status == "processing"
}

func GenerateOrderNumber() string {
	now := time.Now()
	// Format: ORD-YYYYMMDD-HHMMSS-RAND
	return fmt.Sprintf("ORD-%s-%03d",
		now.Format("20060102-150405"),
		now.Nanosecond()%1000,
	)
}
