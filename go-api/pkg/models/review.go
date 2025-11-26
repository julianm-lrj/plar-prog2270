package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Review represents a customer review for a product
type Review struct {
	ID               bson.ObjectID `json:"id" bson:"_id,omitempty"`
	ProductID        bson.ObjectID `json:"product_id" bson:"product_id" validate:"required"`
	CustomerID       bson.ObjectID `json:"customer_id" bson:"customer_id" validate:"required"`
	OrderID          bson.ObjectID `json:"order_id" bson:"order_id,omitempty"`
	Rating           int           `json:"rating" bson:"rating" validate:"required,gte=1,lte=5"`
	Title            string        `json:"title" bson:"title" validate:"required,min=2,max=200"`
	Comment          string        `json:"comment" bson:"comment" validate:"max=2000"`
	VerifiedPurchase bool          `json:"verified_purchase" bson:"verified_purchase"`
	HelpfulCount     int           `json:"helpful_count" bson:"helpful_count" validate:"gte=0"`
	CreatedAt        time.Time     `json:"created_at" bson:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at" bson:"updated_at"`
}

// SetTimestamps sets created_at and updated_at timestamps
func (r *Review) SetTimestamps() {
	now := time.Now()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	r.UpdatedAt = now
}

// IncrementHelpfulCount increments the helpful count by 1
func (r *Review) IncrementHelpfulCount() {
	r.HelpfulCount++
	r.UpdatedAt = time.Now()
}

// DecrementHelpfulCount decrements the helpful count by 1 (minimum 0)
func (r *Review) DecrementHelpfulCount() {
	if r.HelpfulCount > 0 {
		r.HelpfulCount--
		r.UpdatedAt = time.Now()
	}
}

// IsPositive checks if the review is positive (4-5 stars)
func (r *Review) IsPositive() bool {
	return r.Rating >= 4
}

// IsNegative checks if the review is negative (1-2 stars)
func (r *Review) IsNegative() bool {
	return r.Rating <= 2
}

// IsVerified checks if this is a verified purchase
func (r *Review) IsVerified() bool {
	return r.VerifiedPurchase
}
