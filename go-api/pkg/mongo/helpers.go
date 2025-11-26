package mongo

import (
	"go.mongodb.org/mongo-driver/v2/bson"
	"julianmorley.ca/con-plar/prog2270/pkg/global"
)

// var databaseName = global.GetEnvOrDefault("MONGODB_DATABASE", "ecommerce")
// var productsConn = Client.Database(databaseName).Collection("products")
// var ordersConn = Client.Database(databaseName).Collection("orders")
// var customersConn = Client.Database(databaseName).Collection("customers")
// var reviewsConn = Client.Database(databaseName).Collection("reviews")
// var cartItemsConn = Client.Database(databaseName).Collection("cart_items")
// var inventoryConn = Client.Database(databaseName).Collection("inventory")

// func EnsureIndexesOnStartup() {
// 	ctx, cancel := global.GetDefaultTimer()
// 	defer cancel()

// 	if err := EnsureIndexes(ctx, db); err != nil {
// 		log.Fatalf("Failed to ensure indexes: %v", err)
// 	}
// }

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
