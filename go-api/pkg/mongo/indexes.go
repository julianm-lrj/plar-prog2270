package mongo

import (
	"log"

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

		// Create the index
		indexName, err := collection.Indexes().CreateOne(ctx, idxConfig.IndexModel)
		if err != nil {
			log.Printf("Error creating index on collection %s: %v",
				idxConfig.CollectionName, err)
			return err
		}

		log.Printf("✓ Created index '%s' on collection '%s'", indexName, idxConfig.CollectionName)
	}

	log.Println("All indexes created successfully!")
	return nil
}

func EnsureIndexesOnStartup() {
	if err := EnsureIndexes(); err != nil {
		log.Fatalf("Failed to ensure indexes: %v", err)
	}
}
