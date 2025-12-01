package mongo

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
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

// GetOrderByNumber retrieves an order by its order number
func GetOrderByNumber(ctx context.Context, orderNumber string) (*models.Order, error) {
	collection := GetCollection("orders")

	var order models.Order
	err := collection.FindOne(ctx, bson.M{"order_number": orderNumber}).Decode(&order)
	if err != nil {
		return nil, err
	}

	return &order, nil
}

// UpdateOrderByNumber updates an order by its order number with partial updates
func UpdateOrderByNumber(ctx context.Context, orderNumber string, updates map[string]interface{}) (*models.Order, error) {
	collection := GetCollection("orders")

	// Add updated_at timestamp
	updates["updated_at"] = time.Now()

	// Perform the update
	filter := bson.M{"order_number": orderNumber}
	update := bson.M{"$set": updates}

	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}

	// Return the updated order
	return GetOrderByNumber(ctx, orderNumber)
}

// DeleteOrderByNumber deletes an order by its order number
func DeleteOrderByNumber(ctx context.Context, orderNumber string) (*models.Order, error) {
	collection := GetCollection("orders")

	// First, get the order to return it
	order, err := GetOrderByNumber(ctx, orderNumber)
	if err != nil {
		return nil, err
	}

	// Then delete it
	filter := bson.M{"order_number": orderNumber}
	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return nil, err
	}

	if result.DeletedCount == 0 {
		return nil, errors.New("order not found")
	}

	return order, nil
}

// CreateNewOrder creates a new order in the database
func CreateNewOrder(ctx context.Context, orderRequest *models.CreateOrderRequest) (*models.Order, error) {
	collection := GetCollection("orders")

	// Create the order from the request
	order := &models.Order{
		OrderNumber:     models.GenerateOrderNumber(),
		CustomerID:      orderRequest.CustomerID,
		CustomerEmail:   orderRequest.CustomerEmail,
		Status:          "pending",
		Items:           orderRequest.Items,
		ShippingAddress: orderRequest.ShippingAddress,
		BillingAddress:  orderRequest.BillingAddress,
		Payment:         orderRequest.Payment,
		Notes:           orderRequest.Notes,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Calculate item subtotals
	for i := range order.Items {
		order.Items[i].CalculateItemSubtotal()
	}

	// Calculate order totals
	order.CalculateTotals()

	// Set timeline
	order.Timeline.OrderedAt = time.Now()

	// Insert into database
	result, err := collection.InsertOne(ctx, order)
	if err != nil {
		return nil, err
	}

	// Set the generated ID
	order.ID = result.InsertedID.(bson.ObjectID)

	return order, nil
}

// CreateNewOrders creates multiple orders in a single operation
func CreateNewOrders(ctx context.Context, orderRequests []models.CreateOrderRequest) ([]models.Order, []error) {
	collection := GetCollection("orders")
	customersCollection := GetCollection("customers")

	var orders []models.Order
	var errorsList []error
	var ordersToInsert []interface{}

	// Process each order request
	for _, orderRequest := range orderRequests {
		// Validate customer exists by email
		var customer models.Customer
		err := customersCollection.FindOne(ctx, bson.D{{Key: "email", Value: orderRequest.CustomerEmail}}).Decode(&customer)
		if err != nil {
			if err.Error() == "mongo: no documents in result" {
				errorsList = append(errorsList, errors.New("customer with email '"+orderRequest.CustomerEmail+"' not found"))
			} else {
				errorsList = append(errorsList, err)
			}
			// Add a placeholder order to maintain index alignment
			orders = append(orders, models.Order{})
			continue
		}

		// Verify CustomerID matches the found customer
		if !customer.ID.IsZero() && customer.ID != orderRequest.CustomerID {
			errorsList = append(errorsList, errors.New("customer ID does not match email '"+orderRequest.CustomerEmail+"'"))
			// Add a placeholder order to maintain index alignment
			orders = append(orders, models.Order{})
			continue
		}

		// Create the order
		order := models.Order{
			OrderNumber:     models.GenerateOrderNumber(),
			CustomerID:      customer.ID,    // Use the verified customer ID
			CustomerEmail:   customer.Email, // Use the verified customer email
			Status:          "pending",
			Items:           orderRequest.Items,
			ShippingAddress: orderRequest.ShippingAddress,
			BillingAddress:  orderRequest.BillingAddress,
			Payment:         orderRequest.Payment,
			Notes:           orderRequest.Notes,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		// Calculate item subtotals
		for j := range order.Items {
			order.Items[j].CalculateItemSubtotal()
		}

		// Calculate order totals
		order.CalculateTotals()

		// Set timeline
		order.Timeline.OrderedAt = time.Now()

		orders = append(orders, order)
		ordersToInsert = append(ordersToInsert, order)
		errorsList = append(errorsList, nil) // No error for this order
	}

	// Insert valid orders only
	if len(ordersToInsert) > 0 {
		result, err := collection.InsertMany(ctx, ordersToInsert)
		if err != nil {
			// If bulk insert fails, mark all valid orders as failed
			for i := 0; i < len(orders); i++ {
				if errorsList[i] == nil { // This was a valid order that should have been inserted
					errorsList[i] = err
				}
			}
			return orders, errorsList
		}

		// Set the generated IDs for successfully inserted orders
		insertIndex := 0
		for i := 0; i < len(orders); i++ {
			if errorsList[i] == nil { // This order was successfully processed
				if insertIndex < len(result.InsertedIDs) {
					orders[i].ID = result.InsertedIDs[insertIndex].(bson.ObjectID)
					insertIndex++
				}
			}
		}
	}

	return orders, errorsList
}

// GetAllCategories retrieves distinct category values from the products collection
func GetAllCategories() ([]string, error) {
	ctx, cancel := global.GetDefaultTimer()
	defer cancel()
	collection := GetCollection("products")

	// Use MongoDB aggregation to get distinct categories (case insensitive)
	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id": bson.M{
					"$toLower": "$category",
				},
				"originalCase": bson.M{
					"$first": "$category",
				},
				"count": bson.M{
					"$sum": 1,
				},
			},
		},
		{
			"$sort": bson.M{
				"originalCase": 1,
			},
		},
		{
			"$project": bson.M{
				"_id":      0,
				"category": "$originalCase",
				"count":    1,
			},
		},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	// Extract category names from results
	var categories []string
	for _, result := range results {
		if category, ok := result["category"].(string); ok {
			categories = append(categories, category)
		}
	}

	return categories, nil
}

func GetAllReviewsForItem(entity string, entityId string) ([]models.Review, error) {
	var reviews []models.Review

	ctx, cancel := global.GetDefaultTimer()
	defer cancel()

	collection := GetCollection("reviews")

	// Convert entityId to ObjectID
	objId, err := bson.ObjectIDFromHex(entityId)
	if err != nil {
		return nil, err
	}

	// Build filter based on entity type
	var filter bson.M
	switch entity {
	case "product":
		filter = bson.M{"product_id": objId}
	case "customer":
		filter = bson.M{"customer_id": objId}
	case "order":
		filter = bson.M{"order_id": objId}
	default:
		return nil, errors.New("invalid entity type: " + entity)
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &reviews); err != nil {
		return nil, err
	}

	return reviews, nil
}

// CreateReviewForItem creates a new review in the database
func CreateReviewForItem(reviewRequest *models.CreateReviewRequest) (*models.Review, error) {
	ctx, cancel := global.GetDefaultTimer()
	defer cancel()

	collection := GetCollection("reviews")

	// Validate that the product exists
	productCollection := GetCollection("products")
	var product models.Product
	err := productCollection.FindOne(ctx, bson.M{"_id": reviewRequest.ProductID}).Decode(&product)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, errors.New("product not found")
		}
		return nil, err
	}

	// Validate that the customer exists
	customersCollection := GetCollection("customers")
	var customer models.Customer
	err = customersCollection.FindOne(ctx, bson.M{"_id": reviewRequest.CustomerID}).Decode(&customer)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, errors.New("customer not found")
		}
		return nil, err
	}

	// If order ID is provided, validate it exists and belongs to the customer
	if !reviewRequest.OrderID.IsZero() {
		ordersCollection := GetCollection("orders")
		var order models.Order
		err = ordersCollection.FindOne(ctx, bson.M{
			"_id":         reviewRequest.OrderID,
			"customer_id": reviewRequest.CustomerID,
		}).Decode(&order)
		if err != nil {
			if err.Error() == "mongo: no documents in result" {
				return nil, errors.New("order not found or does not belong to customer")
			}
			return nil, err
		}

		// Verify the product is in the order
		productInOrder := false
		for _, item := range order.Items {
			if item.ProductID == reviewRequest.ProductID {
				productInOrder = true
				break
			}
		}
		if !productInOrder {
			return nil, errors.New("product not found in the specified order")
		}
	}

	// Check if customer has already reviewed this product
	existingReview := collection.FindOne(ctx, bson.M{
		"product_id":  reviewRequest.ProductID,
		"customer_id": reviewRequest.CustomerID,
	})
	if existingReview.Err() == nil {
		return nil, errors.New("customer has already reviewed this product")
	}

	// Create the review
	review := &models.Review{
		ProductID:        reviewRequest.ProductID,
		CustomerID:       reviewRequest.CustomerID,
		OrderID:          reviewRequest.OrderID,
		Rating:           reviewRequest.Rating,
		Title:            reviewRequest.Title,
		Comment:          reviewRequest.Comment,
		VerifiedPurchase: reviewRequest.VerifiedPurchase,
		HelpfulCount:     0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Insert into database
	result, err := collection.InsertOne(ctx, review)
	if err != nil {
		return nil, err
	}

	// Set the generated ID
	review.ID = result.InsertedID.(bson.ObjectID)

	return review, nil
}

// UpdateReviewForItem updates an existing review with partial updates
func UpdateReviewForItem(reviewID string, productID string, updateRequest *models.UpdateReviewRequest) (*models.Review, error) {
	ctx, cancel := global.GetDefaultTimer()
	defer cancel()

	collection := GetCollection("reviews")

	// Convert IDs to ObjectIDs
	reviewObjID, err := bson.ObjectIDFromHex(reviewID)
	if err != nil {
		return nil, errors.New("invalid review ID format")
	}

	productObjID, err := bson.ObjectIDFromHex(productID)
	if err != nil {
		return nil, errors.New("invalid product ID format")
	}

	// Build update document
	updates := bson.M{
		"updated_at": time.Now(),
	}

	// Add fields that are being updated
	if updateRequest.Rating != nil {
		updates["rating"] = *updateRequest.Rating
	}
	if updateRequest.Title != nil {
		updates["title"] = *updateRequest.Title
	}
	if updateRequest.Comment != nil {
		updates["comment"] = *updateRequest.Comment
	}

	// Perform the update - only update if review belongs to the specified product
	filter := bson.M{
		"_id":        reviewObjID,
		"product_id": productObjID,
	}

	updateDoc := bson.M{"$set": updates}
	result, err := collection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return nil, err
	}

	if result.MatchedCount == 0 {
		return nil, errors.New("review not found for this product")
	}

	// Return the updated review
	var updatedReview models.Review
	err = collection.FindOne(ctx, bson.M{"_id": reviewObjID}).Decode(&updatedReview)
	if err != nil {
		return nil, err
	}

	return &updatedReview, nil
}

// DeleteReviewForItem deletes a review by ID for a specific product
func DeleteReviewForItem(reviewID string, productID string) (string, error) {
	ctx, cancel := global.GetDefaultTimer()
	defer cancel()

	collection := GetCollection("reviews")

	// Convert IDs to ObjectIDs
	reviewObjID, err := bson.ObjectIDFromHex(reviewID)
	if err != nil {
		return "", errors.New("invalid review ID format")
	}

	productObjID, err := bson.ObjectIDFromHex(productID)
	if err != nil {
		return "", errors.New("invalid product ID format")
	}

	// Delete review - only delete if review belongs to the specified product
	filter := bson.M{
		"_id":        reviewObjID,
		"product_id": productObjID,
	}

	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return "", err
	}

	if result.DeletedCount == 0 {
		return "", errors.New("review not found for this product")
	}

	return reviewID, nil
}

// SearchResult represents a search result item with metadata
type SearchResult struct {
	ID      interface{} `json:"id"`
	Type    string      `json:"type"`
	Title   string      `json:"title"`
	Snippet string      `json:"snippet"`
	Score   float64     `json:"score,omitempty"`
	Data    interface{} `json:"data"`
}

// SearchResults represents grouped search results by collection type
type SearchResults struct {
	Products  []SearchResult `json:"products"`
	Customers []SearchResult `json:"customers"`
	Orders    []SearchResult `json:"orders"`
	Reviews   []SearchResult `json:"reviews"`
	Total     int            `json:"total"`
}

// SearchDatabase performs full-text search across all collections
func SearchDatabase(query string, limit int) (*SearchResults, error) {
	ctx, cancel := global.GetDefaultTimer()
	defer cancel()

	results := &SearchResults{
		Products:  []SearchResult{},
		Customers: []SearchResult{},
		Orders:    []SearchResult{},
		Reviews:   []SearchResult{},
	}

	// Search Products
	productsCollection := GetCollection("products")
	productResults, err := searchProducts(ctx, productsCollection, query, limit)
	if err == nil {
		results.Products = productResults
	}

	// Search Customers
	customersCollection := GetCollection("customers")
	customerResults, err := searchCustomers(ctx, customersCollection, query, limit)
	if err == nil {
		results.Customers = customerResults
	}

	// Search Orders
	ordersCollection := GetCollection("orders")
	orderResults, err := searchOrders(ctx, ordersCollection, query, limit)
	if err == nil {
		results.Orders = orderResults
	}

	// Search Reviews
	reviewsCollection := GetCollection("reviews")
	reviewResults, err := searchReviews(ctx, reviewsCollection, query, limit)
	if err == nil {
		results.Reviews = reviewResults
	}

	results.Total = len(results.Products) + len(results.Customers) + len(results.Orders) + len(results.Reviews)

	return results, nil
}

func searchProducts(ctx context.Context, collection *mongo.Collection, query string, limit int) ([]SearchResult, error) {
	var products []models.Product
	var results []SearchResult

	// Use regex search for name, description, category, tags
	filter := bson.M{
		"$or": []bson.M{
			{"name": bson.M{"$regex": query, "$options": "i"}},
			{"description": bson.M{"$regex": query, "$options": "i"}},
			{"category": bson.M{"$regex": query, "$options": "i"}},
			{"tags": bson.M{"$regex": query, "$options": "i"}},
			{"sku": bson.M{"$regex": query, "$options": "i"}},
		},
	}

	cursor, err := collection.Find(ctx, filter, options.Find().SetLimit(int64(limit)))
	if err != nil {
		return results, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &products); err != nil {
		return results, err
	}

	for _, product := range products {
		snippet := product.Description
		if len(snippet) > 150 {
			snippet = snippet[:150] + "..."
		}

		results = append(results, SearchResult{
			ID:      product.ID,
			Type:    "product",
			Title:   product.Name,
			Snippet: snippet,
			Data:    product,
		})
	}

	return results, nil
}

func searchCustomers(ctx context.Context, collection *mongo.Collection, query string, limit int) ([]SearchResult, error) {
	var customers []models.Customer
	var results []SearchResult

	// Search customer name and email
	filter := bson.M{
		"$or": []bson.M{
			{"first_name": bson.M{"$regex": query, "$options": "i"}},
			{"last_name": bson.M{"$regex": query, "$options": "i"}},
			{"email": bson.M{"$regex": query, "$options": "i"}},
			{"phone": bson.M{"$regex": query, "$options": "i"}},
		},
	}

	cursor, err := collection.Find(ctx, filter, options.Find().SetLimit(int64(limit)))
	if err != nil {
		return results, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &customers); err != nil {
		return results, err
	}

	for _, customer := range customers {
		name := customer.FirstName + " " + customer.LastName
		snippet := fmt.Sprintf("Email: %s | Phone: %s", customer.Email, customer.Phone)

		results = append(results, SearchResult{
			ID:      customer.ID,
			Type:    "customer",
			Title:   name,
			Snippet: snippet,
			Data:    customer,
		})
	}

	return results, nil
}

func searchOrders(ctx context.Context, collection *mongo.Collection, query string, limit int) ([]SearchResult, error) {
	var orders []models.Order
	var results []SearchResult

	// Search order number, customer email, status
	filter := bson.M{
		"$or": []bson.M{
			{"order_number": bson.M{"$regex": query, "$options": "i"}},
			{"customer_email": bson.M{"$regex": query, "$options": "i"}},
			{"status": bson.M{"$regex": query, "$options": "i"}},
			{"notes": bson.M{"$regex": query, "$options": "i"}},
		},
	}

	cursor, err := collection.Find(ctx, filter, options.Find().SetLimit(int64(limit)))
	if err != nil {
		return results, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &orders); err != nil {
		return results, err
	}

	for _, order := range orders {
		snippet := fmt.Sprintf("Status: %s | Total: $%.2f | Items: %d", order.Status, order.Totals.GrandTotal, len(order.Items))

		results = append(results, SearchResult{
			ID:      order.ID,
			Type:    "order",
			Title:   order.OrderNumber,
			Snippet: snippet,
			Data:    order,
		})
	}

	return results, nil
}

func searchReviews(ctx context.Context, collection *mongo.Collection, query string, limit int) ([]SearchResult, error) {
	var reviews []models.Review
	var results []SearchResult

	// Search review title and comment
	filter := bson.M{
		"$or": []bson.M{
			{"title": bson.M{"$regex": query, "$options": "i"}},
			{"comment": bson.M{"$regex": query, "$options": "i"}},
		},
	}

	cursor, err := collection.Find(ctx, filter, options.Find().SetLimit(int64(limit)))
	if err != nil {
		return results, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &reviews); err != nil {
		return results, err
	}

	for _, review := range reviews {
		snippet := review.Comment
		if len(snippet) > 150 {
			snippet = snippet[:150] + "..."
		}

		results = append(results, SearchResult{
			ID:      review.ID,
			Type:    "review",
			Title:   review.Title,
			Snippet: snippet,
			Data:    review,
		})
	}

	return results, nil
}

// DeleteCustomer removes a customer by ID
func DeleteCustomer(ctx context.Context, customerID string) error {
	collection := GetCollection("customers")

	// Parse customer ID
	objectID, err := bson.ObjectIDFromHex(customerID)
	if err != nil {
		return err
	}

	// Delete the customer
	result, err := collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}

	// Check if customer was found and deleted
	if result.DeletedCount == 0 {
		return errors.New("customer not found")
	}

	return nil
}
