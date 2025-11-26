package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Customer struct {
	ID            bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Email         string        `bson:"email" json:"email" validate:"required,email"`
	Password      string        `bson:"password" json:"-" validate:"required,min=6"` // Never expose in JSON
	FirstName     string        `bson:"first_name" json:"first_name" validate:"required,min=2,max=50"`
	LastName      string        `bson:"last_name" json:"last_name" validate:"required,min=2,max=50"`
	Phone         string        `bson:"phone" json:"phone" validate:"required,min=10,max=20"`
	Addresses     []Address     `bson:"addresses" json:"addresses" validate:"dive"`
	Preferences   Preferences   `bson:"preferences" json:"preferences"`
	LoyaltyPoints int           `bson:"loyalty_points" json:"loyalty_points" validate:"gte=0"`
	AccountStatus string        `bson:"account_status" json:"account_status" validate:"required,oneof=active inactive suspended deleted"`
	EmailVerified bool          `bson:"email_verified" json:"email_verified"`
	PhoneVerified bool          `bson:"phone_verified" json:"phone_verified"`
	TotalOrders   int           `bson:"total_orders" json:"total_orders" validate:"gte=0"`
	TotalSpent    float64       `bson:"total_spent" json:"total_spent" validate:"gte=0"`
	LastOrderDate time.Time     `bson:"last_order_date,omitempty" json:"last_order_date,omitempty"`
	CreatedAt     time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time     `bson:"updated_at" json:"updated_at"`
}

type CreateCustomerRequest struct {
	Email     string  `json:"email" validate:"required,email"`
	Password  string  `json:"password" validate:"required,min=8"`
	FirstName string  `json:"first_name" validate:"required,min=2,max=50"`
	LastName  string  `json:"last_name" validate:"required,min=2,max=50"`
	Phone     string  `json:"phone" validate:"required,min=10,max=20"`
	Address   Address `json:"address" validate:"required"`
}

type UpdateCustomerRequest struct {
	FirstName     *string      `json:"first_name,omitempty" validate:"omitempty,min=2,max=50"`
	LastName      *string      `json:"last_name,omitempty" validate:"omitempty,min=2,max=50"`
	Phone         *string      `json:"phone,omitempty" validate:"omitempty,min=10,max=20"`
	Addresses     []Address    `json:"addresses,omitempty" validate:"omitempty,dive"`
	Preferences   *Preferences `json:"preferences,omitempty"`
	AccountStatus *string      `json:"account_status,omitempty" validate:"omitempty,oneof=active inactive suspended deleted"`
}

type Preferences struct {
	Newsletter         bool     `bson:"newsletter" json:"newsletter"`
	SMSNotifications   bool     `bson:"sms_notifications" json:"sms_notifications"`
	EmailNotifications bool     `bson:"email_notifications" json:"email_notifications"`
	Language           string   `bson:"language" json:"language" validate:"oneof=en fr es"`
	Currency           string   `bson:"currency" json:"currency" validate:"oneof=CAD USD EUR"`
	FavoriteCategories []string `bson:"favorite_categories,omitempty" json:"favorite_categories,omitempty"`
}

func (c *Customer) SetTimestamps() {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now
}

func (c *Customer) GetFullName() string {
	return c.FirstName + " " + c.LastName
}

func (c *Customer) AddLoyaltyPoints(points int) {
	if points > 0 {
		c.LoyaltyPoints += points
		c.UpdatedAt = time.Now()
	}
}

func (c *Customer) RedeemLoyaltyPoints(points int) bool {
	if points > 0 && c.LoyaltyPoints >= points {
		c.LoyaltyPoints -= points
		c.UpdatedAt = time.Now()
		return true
	}
	return false
}

func (c *Customer) GetDefaultAddress() *Address {
	for i := range c.Addresses {
		if c.Addresses[i].IsDefault {
			return &c.Addresses[i]
		}
	}
	if len(c.Addresses) > 0 {
		return &c.Addresses[0]
	}
	return nil
}

func (c *Customer) AddAddress(address Address) {
	if len(c.Addresses) == 0 {
		address.IsDefault = true
	}
	c.Addresses = append(c.Addresses, address)
	c.UpdatedAt = time.Now()
}

func (c *Customer) SetDefaultAddress(addressIndex int) bool {
	if addressIndex < 0 || addressIndex >= len(c.Addresses) {
		return false
	}

	for i := range c.Addresses {
		c.Addresses[i].IsDefault = (i == addressIndex)
	}
	c.UpdatedAt = time.Now()
	return true
}

func (c *Customer) IsActive() bool {
	return c.AccountStatus == "active"
}

func (c *Customer) IsSuspended() bool {
	return c.AccountStatus == "suspended"
}

func (c *Customer) IsVerified() bool {
	return c.EmailVerified && c.PhoneVerified
}

func (c *Customer) UpdateOrderStats(orderAmount float64) {
	c.TotalOrders++
	c.TotalSpent += orderAmount
	c.LastOrderDate = time.Now()
	c.UpdatedAt = time.Now()
}

func (c *Customer) CalculateLoyaltyTier() string {
	switch {
	case c.LoyaltyPoints >= 10000:
		return "Platinum"
	case c.LoyaltyPoints >= 5000:
		return "Gold"
	case c.LoyaltyPoints >= 1000:
		return "Silver"
	default:
		return "Bronze"
	}
}

func (c *Customer) GetAverageOrderValue() float64 {
	if c.TotalOrders == 0 {
		return 0.0
	}
	return c.TotalSpent / float64(c.TotalOrders)
}
