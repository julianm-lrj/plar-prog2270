package mongo

import (
	"fmt"
	"log"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"julianmorley.ca/con-plar/prog2270/pkg/global"
)

type IndexConfig struct {
	CollectionName string
	IndexModel     mongo.IndexModel
}

var requiredIndexes = []IndexConfig{
	// Customers Collection Indexes
	{
		CollectionName: "customers",
		IndexModel: mongo.IndexModel{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_customer_email_unique"),
		},
	},

	// Products Collection Indexes
	// Index 1: Single-field index on category for filtering
	{
		CollectionName: "products",
		IndexModel: mongo.IndexModel{
			Keys:    bson.D{{Key: "category", Value: 1}},
			Options: options.Index().SetName("idx_category"),
		},
	},
	// Index 2: Compound index on status and price for sorted product listings
	{
		CollectionName: "products",
		IndexModel: mongo.IndexModel{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "price", Value: -1},
			},
			Options: options.Index().SetName("idx_status_price"),
		},
	},
	// Index 3: Text index for full-text search on products
	{
		CollectionName: "products",
		IndexModel: mongo.IndexModel{
			Keys: bson.D{
				{Key: "name", Value: "text"},
				{Key: "description", Value: "text"},
				{Key: "tags", Value: "text"},
			},
			Options: options.Index().
				SetName("idx_product_text_search").
				SetWeights(bson.D{
					{Key: "name", Value: 10},
					{Key: "tags", Value: 5},
					{Key: "description", Value: 1},
				}),
		},
	},
	// Index 4: Compound index for low-stock alerts
	{
		CollectionName: "products",
		IndexModel: mongo.IndexModel{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "stock.total", Value: 1},
			},
			Options: options.Index().SetName("idx_stock_alert"),
		},
	},
	// Index 5: SKU unique index
	{
		CollectionName: "products",
		IndexModel: mongo.IndexModel{
			Keys:    bson.D{{Key: "sku", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_sku_unique"),
		},
	},

	// Orders Collection Indexes
	// Index 6: Compound index for customer order history
	{
		CollectionName: "orders",
		IndexModel: mongo.IndexModel{
			Keys: bson.D{
				{Key: "customer_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("idx_customer_orders"),
		},
	},
	// Index 7: Compound index for analytics queries
	{
		CollectionName: "orders",
		IndexModel: mongo.IndexModel{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "timeline.ordered_at", Value: -1},
			},
			Options: options.Index().SetName("idx_analytics"),
		},
	},
	// Index 8: Unique index on order_number
	{
		CollectionName: "orders",
		IndexModel: mongo.IndexModel{
			Keys:    bson.D{{Key: "order_number", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_order_number_unique"),
		},
	},

	// Reviews Collection Indexes
	// Index 9: Product reviews lookup
	{
		CollectionName: "reviews",
		IndexModel: mongo.IndexModel{
			Keys:    bson.D{{Key: "product_id", Value: 1}},
			Options: options.Index().SetName("idx_product_reviews"),
		},
	},
	// Index 10: Customer reviews lookup
	{
		CollectionName: "reviews",
		IndexModel: mongo.IndexModel{
			Keys:    bson.D{{Key: "customer_id", Value: 1}},
			Options: options.Index().SetName("idx_customer_reviews"),
		},
	},

	// Inventory Logs Collection Indexes
	// Index 11: Time-series index for recent inventory changes
	{
		CollectionName: "inventory_logs",
		IndexModel: mongo.IndexModel{
			Keys: bson.D{
				{Key: "timestamp", Value: -1},
				{Key: "sku", Value: 1},
			},
			Options: options.Index().SetName("idx_inventory_time"),
		},
	},
	// Index 12: SKU history lookup
	{
		CollectionName: "inventory_logs",
		IndexModel: mongo.IndexModel{
			Keys: bson.D{
				{Key: "sku", Value: 1},
				{Key: "timestamp", Value: -1},
			},
			Options: options.Index().SetName("idx_sku_history"),
		},
	},
}

func EnsureIndexes() error {
	log.Println("Starting index creation...")

	for _, idxConfig := range requiredIndexes {
		collection := GetCollection(idxConfig.CollectionName)
		ctx, cancel := global.GetDefaultTimer()
		defer cancel()

		// Try to extract index name from the options
		var indexName string
		if idxConfig.IndexModel.Options != nil {
			// Build the options to extract the name
			opts := idxConfig.IndexModel.Options
			if opts != nil {
				indexName = "custom_index" // We'll use the defined names from our config
			}
		}

		// For our specific indexes, we know the names
		switch {
		case strings.Contains(fmt.Sprintf("%v", idxConfig.IndexModel.Keys), "category"):
			indexName = "idx_category"
		case strings.Contains(fmt.Sprintf("%v", idxConfig.IndexModel.Keys), "sku"):
			indexName = "idx_sku_unique"
		case strings.Contains(fmt.Sprintf("%v", idxConfig.IndexModel.Keys), "name"):
			indexName = "idx_product_text_search"
		case strings.Contains(fmt.Sprintf("%v", idxConfig.IndexModel.Keys), "status") &&
			strings.Contains(fmt.Sprintf("%v", idxConfig.IndexModel.Keys), "stock"):
			indexName = "idx_stock_alert"
		case strings.Contains(fmt.Sprintf("%v", idxConfig.IndexModel.Keys), "status") &&
			strings.Contains(fmt.Sprintf("%v", idxConfig.IndexModel.Keys), "price"):
			indexName = "idx_status_price"
		case strings.Contains(fmt.Sprintf("%v", idxConfig.IndexModel.Keys), "customer_id"):
			indexName = "idx_customer_orders"
		case strings.Contains(fmt.Sprintf("%v", idxConfig.IndexModel.Keys), "order_number"):
			indexName = "idx_order_number_unique"
		case strings.Contains(fmt.Sprintf("%v", idxConfig.IndexModel.Keys), "email"):
			indexName = "idx_customer_email_unique"
		default:
			indexName = "unknown_index"
		}

		// Check if index already exists
		cursor, err := collection.Indexes().List(ctx)
		if err != nil {
			log.Printf("Error listing indexes on collection %s: %v", idxConfig.CollectionName, err)
			continue
		}

		var existingIndexes []bson.M
		if err = cursor.All(ctx, &existingIndexes); err != nil {
			log.Printf("Error reading indexes on collection %s: %v", idxConfig.CollectionName, err)
			continue
		}

		indexExists := false
		for _, index := range existingIndexes {
			if name, ok := index["name"].(string); ok && name == indexName {
				indexExists = true
				break
			}
		}

		if indexExists {
			log.Printf("✓ Index '%s' already exists on collection '%s'", indexName, idxConfig.CollectionName)
			continue
		}

		// Create the index
		createdIndexName, err := collection.Indexes().CreateOne(ctx, idxConfig.IndexModel)
		if err != nil {
			// Handle duplicate key errors gracefully for unique indexes
			if strings.Contains(err.Error(), "DuplicateKey") || strings.Contains(err.Error(), "E11000") {
				log.Printf("⚠ Skipping index '%s' on collection '%s' due to duplicate keys in existing data.",
					indexName, idxConfig.CollectionName)
				log.Printf("💡 Consider running cleanup: CleanupDuplicateSKUs()")
				continue
			}
			log.Printf("Error creating index '%s' on collection %s: %v", indexName, idxConfig.CollectionName, err)
			return err
		}

		log.Printf("✓ Created index '%s' on collection '%s'", createdIndexName, idxConfig.CollectionName)
	}

	log.Println("All indexes processed successfully!")
	return nil
}

func EnsureIndexesOnStartup() {
	if err := EnsureIndexes(); err != nil {
		log.Fatalf("Failed to ensure indexes: %v", err)
	}
}

// CleanupDuplicateSKUs removes products with duplicate SKUs, keeping the most recent one
func CleanupDuplicateSKUs() error {
	log.Println("Checking for duplicate SKUs...")

	collection := GetCollection("products")
	ctx, cancel := global.GetDefaultTimer()
	defer cancel()

	// Aggregation pipeline to find duplicate SKUs
	pipeline := bson.A{
		bson.D{{"$group", bson.D{
			{"_id", "$sku"},
			{"ids", bson.D{{"$push", "$_id"}}},
			{"count", bson.D{{"$sum", 1}}},
		}}},
		bson.D{{"$match", bson.D{
			{"count", bson.D{{"$gt", 1}}},
		}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var duplicates []bson.M
	if err = cursor.All(ctx, &duplicates); err != nil {
		return err
	}

	if len(duplicates) == 0 {
		log.Println("✓ No duplicate SKUs found")
		return nil
	}

	log.Printf("Found %d SKUs with duplicates, cleaning up...", len(duplicates))

	for _, dup := range duplicates {
		sku := dup["_id"].(string)
		ids := dup["ids"].(bson.A)

		// Keep only the first ID (most recent), delete the rest
		for i := 1; i < len(ids); i++ {
			_, err := collection.DeleteOne(ctx, bson.D{{"_id", ids[i]}})
			if err != nil {
				log.Printf("Error deleting duplicate product with ID %v: %v", ids[i], err)
				continue
			}
			log.Printf("✓ Deleted duplicate product with SKU %s", sku)
		}
	}

	log.Println("✓ Duplicate SKU cleanup completed")
	return nil
}
