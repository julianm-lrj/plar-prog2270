package mongo

import (
	"context"
	"errors"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"julianmorley.ca/con-plar/prog2270/pkg/global"
	"julianmorley.ca/con-plar/prog2270/pkg/models"
)

func GetAllProducts() ([]bson.M, error) {
	ctx, cancel := global.GetDefaultTimer()
	defer cancel()
	collection := GetCollection("products")

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

func CreateProducts(ctx context.Context, products []*models.Product) ([]*models.Product, error) {
	collection := GetCollection("products")

	// Convert to interface slice for InsertMany
	docs := make([]interface{}, len(products))
	for i, product := range products {
		docs[i] = product
	}

	result, err := collection.InsertMany(ctx, docs)
	if err != nil {
		return nil, err
	}

	// Update the products with their inserted IDs
	for i, insertedID := range result.InsertedIDs {
		if objectID, ok := insertedID.(bson.ObjectID); ok {
			products[i].ID = objectID
		}
	}

	return products, nil
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

// GetProductBySKU retrieves a single product by its SKU
func GetProductBySKU(ctx context.Context, sku string) (*models.Product, error) {
	collection := GetCollection("products")

	var product models.Product
	err := collection.FindOne(ctx, bson.D{{"sku", sku}}).Decode(&product)
	if err != nil {
		return nil, err
	}

	return &product, nil
}

// UpdateProductBySKU updates specific fields of a product by SKU and returns the updated product
func UpdateProductBySKU(ctx context.Context, sku string, updates map[string]interface{}) (*models.Product, error) {
	collection := GetCollection("products")

	// Add updated_at timestamp to the updates
	updates["updated_at"] = time.Now()

	// Create update document
	updateDoc := bson.D{{"$set", updates}}

	// Update the document
	_, err := collection.UpdateOne(ctx, bson.D{{"sku", sku}}, updateDoc)
	if err != nil {
		return nil, err
	}

	// Fetch and return the updated product
	return GetProductBySKU(ctx, sku)
}

// DeleteProductBySKU deletes a product by SKU and returns the deleted product info
func DeleteProductBySKU(ctx context.Context, sku string) (*models.Product, error) {
	collection := GetCollection("products")

	// First get the product to return it and for cache cleanup
	product, err := GetProductBySKU(ctx, sku)
	if err != nil {
		return nil, err
	}

	// Delete the document
	result, err := collection.DeleteOne(ctx, bson.D{{"sku", sku}})
	if err != nil {
		return nil, err
	}

	// Check if document was actually deleted
	if result.DeletedCount == 0 {
		return nil, errors.New("mongo: no documents in result")
	}

	return product, nil
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

// UpdateCustomer updates a customer profile with partial updates
func UpdateCustomer(ctx context.Context, customerID bson.ObjectID, req *models.UpdateCustomerRequest) (*models.Customer, error) {
	collection := GetCollection("customers")

	// Build update document with only provided fields
	updateDoc := bson.D{}

	if req.FirstName != nil {
		updateDoc = append(updateDoc, bson.E{Key: "first_name", Value: *req.FirstName})
	}
	if req.LastName != nil {
		updateDoc = append(updateDoc, bson.E{Key: "last_name", Value: *req.LastName})
	}
	if req.Phone != nil {
		updateDoc = append(updateDoc, bson.E{Key: "phone", Value: *req.Phone})
	}
	if req.Addresses != nil {
		updateDoc = append(updateDoc, bson.E{Key: "addresses", Value: req.Addresses})
	}
	if req.Preferences != nil {
		updateDoc = append(updateDoc, bson.E{Key: "preferences", Value: *req.Preferences})
	}
	if req.AccountStatus != nil {
		updateDoc = append(updateDoc, bson.E{Key: "account_status", Value: *req.AccountStatus})
	}

	// Always update the updated_at timestamp
	updateDoc = append(updateDoc, bson.E{Key: "updated_at", Value: time.Now()})

	// Email cannot be updated (not included in update document)

	if len(updateDoc) == 1 { // Only updated_at was added
		return nil, errors.New("no fields to update")
	}

	update := bson.D{{Key: "$set", Value: updateDoc}}

	// Find and update, returning the updated document
	findOptions := options.FindOneAndUpdate().SetReturnDocument(options.After)
	// Exclude password from response
	findOptions.SetProjection(bson.D{{Key: "password", Value: 0}})

	var updatedCustomer models.Customer
	err := collection.FindOneAndUpdate(
		ctx,
		bson.D{{Key: "_id", Value: customerID}},
		update,
		findOptions,
	).Decode(&updatedCustomer)

	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, errors.New("customer not found")
		}
		return nil, err
	}

	return &updatedCustomer, nil
}

func AddCustomerAddress(ctx context.Context, customerID bson.ObjectID, address models.Address) (*models.Customer, error) {
	collection := GetCollection("customers")

	if address.IsDefault {
		unsetUpdate := bson.D{{Key: "$set", Value: bson.D{{Key: "addresses.$[].is_default", Value: false}}}}
		_, _ = collection.UpdateOne(ctx, bson.D{{Key: "_id", Value: customerID}}, unsetUpdate)
	}

	update := bson.D{
		{Key: "$push", Value: bson.D{{Key: "addresses", Value: address}}},
		{Key: "$set", Value: bson.D{{Key: "updated_at", Value: time.Now()}}},
	}

	findOptions := options.FindOneAndUpdate().SetReturnDocument(options.After)
	findOptions.SetProjection(bson.D{{Key: "password", Value: 0}})

	var updatedCustomer models.Customer
	err := collection.FindOneAndUpdate(
		ctx,
		bson.D{{Key: "_id", Value: customerID}},
		update,
		findOptions,
	).Decode(&updatedCustomer)

	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, errors.New("customer not found")
		}
		return nil, err
	}

	return &updatedCustomer, nil
}

func UpdateCustomerAddress(ctx context.Context, customerID bson.ObjectID, addressIndex int, address models.Address) (*models.Customer, error) {
	collection := GetCollection("customers")

	var customer models.Customer
	err := collection.FindOne(ctx, bson.D{{Key: "_id", Value: customerID}}).Decode(&customer)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, errors.New("customer not found")
		}
		return nil, err
	}

	if addressIndex < 0 || addressIndex >= len(customer.Addresses) {
		return nil, errors.New("address not found")
	}

	// If setting as default, unset all other defaults first
	if address.IsDefault {
		unsetUpdate := bson.D{{Key: "$set", Value: bson.D{{Key: "addresses.$[].is_default", Value: false}}}}
		_, _ = collection.UpdateOne(ctx, bson.D{{Key: "_id", Value: customerID}}, unsetUpdate)
	}

	// Update the specific address using array index
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "addresses." + strconv.Itoa(addressIndex), Value: address},
			{Key: "updated_at", Value: time.Now()},
		}},
	}

	findOptions := options.FindOneAndUpdate().SetReturnDocument(options.After)
	findOptions.SetProjection(bson.D{{Key: "password", Value: 0}})

	var updatedCustomer models.Customer
	err = collection.FindOneAndUpdate(
		ctx,
		bson.D{{Key: "_id", Value: customerID}},
		update,
		findOptions,
	).Decode(&updatedCustomer)

	if err != nil {
		return nil, err
	}

	return &updatedCustomer, nil
}

func DeleteCustomerAddress(ctx context.Context, customerID bson.ObjectID, addressIndex int) (*models.Customer, error) {
	collection := GetCollection("customers")

	var customer models.Customer
	err := collection.FindOne(ctx, bson.D{{Key: "_id", Value: customerID}}).Decode(&customer)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, errors.New("customer not found")
		}
		return nil, err
	}

	if addressIndex < 0 || addressIndex >= len(customer.Addresses) {
		return nil, errors.New("address not found")
	}

	if len(customer.Addresses) == 1 {
		return nil, errors.New("cannot delete last address")
	}

	wasDefault := customer.Addresses[addressIndex].IsDefault

	customer.Addresses = append(customer.Addresses[:addressIndex], customer.Addresses[addressIndex+1:]...)

	if wasDefault && len(customer.Addresses) > 0 {
		customer.Addresses[0].IsDefault = true
	}

	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "addresses", Value: customer.Addresses},
			{Key: "updated_at", Value: time.Now()},
		}},
	}

	findOptions := options.FindOneAndUpdate().SetReturnDocument(options.After)
	findOptions.SetProjection(bson.D{{Key: "password", Value: 0}})

	var updatedCustomer models.Customer
	err = collection.FindOneAndUpdate(
		ctx,
		bson.D{{Key: "_id", Value: customerID}},
		update,
		findOptions,
	).Decode(&updatedCustomer)

	if err != nil {
		return nil, err
	}

	return &updatedCustomer, nil
}
