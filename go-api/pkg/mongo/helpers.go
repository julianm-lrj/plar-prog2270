package mongo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"julianmorley.ca/con-plar/prog2270/pkg/global"
	"julianmorley.ca/con-plar/prog2270/pkg/models"
)

func GetAllProducts() ([]bson.M, error) {
	ctx, cancel := global.GetDefaultTimer()
	defer cancel()
	collection := GetCollection("inventory")

	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []bson.M
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}

	return items, nil
}

func GetAllOrders() ([]bson.M, error) {
	ctx, cancel := global.GetDefaultTimer()
	defer cancel()
	collection := GetCollection("orders")

	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []bson.M
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}

	return items, nil
}

func GetAllCustomers() ([]bson.M, error) {
	ctx, cancel := global.GetDefaultTimer()
	defer cancel()
	collection := GetCollection("customers")

	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []bson.M
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}

	return items, nil
}

func GetAllReviews() ([]bson.M, error) {
	ctx, cancel := global.GetDefaultTimer()
	defer cancel()
	collection := GetCollection("reviews")

	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []bson.M
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}

	return items, nil
}

func GetAllCartItems() ([]bson.M, error) {
	ctx, cancel := global.GetDefaultTimer()
	defer cancel()
	collection := GetCollection("cart_items")

	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []bson.M
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}

	return items, nil
}

func GetInventoryPagenated() ([]bson.M, error) {
	ctx, cancel := global.GetDefaultTimer()
	defer cancel()
	collection := GetCollection("inventory")

	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []bson.M
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}

	return items, nil
}

type CustomerOrdersResult struct {
	Orders     []bson.M           `json:"orders"`
	Summary    CustomerOrderStats `json:"summary"`
	Pagination PaginationInfo     `json:"pagination"`
}

type CustomerOrderStats struct {
	TotalOrders int     `json:"total_orders"`
	TotalSpent  float64 `json:"total_spent"`
}

type PaginationInfo struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"total_pages"`
	TotalItems int `json:"total_items"`
}

func GetCustomerOrdersWithStats(customerID bson.ObjectID, page int, limit int) (*CustomerOrdersResult, error) {
	ctx, cancel := global.GetDefaultTimer()
	defer cancel()
	collection := GetCollection("orders")

	filter := bson.D{{Key: "customer_id", Value: customerID}}

	totalCount, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	skip := (page - 1) * limit
	totalPages := int(totalCount) / limit
	if int(totalCount)%limit > 0 {
		totalPages++
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var orders []bson.M
	if err := cursor.All(ctx, &orders); err != nil {
		return nil, err
	}

	var totalSpent float64
	for _, order := range orders {
		if totals, ok := order["totals"].(bson.M); ok {
			if grandTotal, ok := totals["grand_total"].(float64); ok {
				totalSpent += grandTotal
			}
		}
	}

	result := &CustomerOrdersResult{
		Orders: orders,
		Summary: CustomerOrderStats{
			TotalOrders: int(totalCount),
			TotalSpent:  totalSpent,
		},
		Pagination: PaginationInfo{
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
			TotalItems: int(totalCount),
		},
	}

	return result, nil
}

// CreateCustomer creates a new customer document in the database
func CreateCustomer(ctx context.Context, customer *models.Customer) (*models.Customer, error) {
	collection := GetCollection("customers")

	// Check if email already exists
	var existingCustomer bson.M
	err := collection.FindOne(ctx, bson.D{{Key: "email", Value: customer.Email}}).Decode(&existingCustomer)
	if err == nil {
		// Email already exists
		return nil, errors.New("email already exists")
	}

	// Insert the customer
	result, err := collection.InsertOne(ctx, customer)
	if err != nil {
		return nil, err
	}

	// Set the generated ID
	customer.ID = result.InsertedID.(bson.ObjectID)

	return customer, nil
}

func GetCustomerByID(ctx context.Context, customerID bson.ObjectID) (*models.Customer, error) {
	collection := GetCollection("customers")

	projection := bson.D{
		{Key: "password", Value: 0},
	}
	findOptions := options.FindOne().SetProjection(projection)

	var customer models.Customer
	err := collection.FindOne(ctx, bson.D{{Key: "_id", Value: customerID}}, findOptions).Decode(&customer)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, errors.New("customer not found")
		}
		return nil, err
	}

	return &customer, nil
}
