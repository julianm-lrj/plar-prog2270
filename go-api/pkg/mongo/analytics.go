package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"julianmorley.ca/con-plar/prog2270/pkg/global"
)

// SalesData represents daily sales summary
type SalesData struct {
	Date            string  `json:"date" bson:"_id"`
	TotalOrders     int     `json:"total_orders" bson:"total_orders"`
	TotalRevenue    float64 `json:"total_revenue" bson:"total_revenue"`
	AvgOrderValue   float64 `json:"avg_order_value" bson:"avg_order_value"`
	UniqueCustomers int     `json:"unique_customers" bson:"unique_customers"`
}

type CustomerSegment struct {
	Segment             string  `json:"segment" bson:"_id"`
	MinSpent            float64 `json:"min_spent" bson:"min_spent"`
	MaxSpent            float64 `json:"max_spent" bson:"max_spent"`
	CustomerCount       int     `json:"customer_count" bson:"count"`
	AvgOrders           float64 `json:"avg_orders" bson:"avg_orders"`
	TotalSpent          float64 `json:"total_spent" bson:"total_spent"`
	AvgSpentPerCustomer float64 `json:"avg_spent_per_customer" bson:"avg_spent_per_customer"`
}

type CustomerSegmentsResult struct {
	Segments       []CustomerSegment `json:"segments"`
	TotalCustomers int               `json:"total_customers"`
}

func GetCustomerSpendingSegments(ctx context.Context) (*CustomerSegmentsResult, error) {
	collection := GetCollection("customers")

	pipeline := bson.A{
		bson.D{
			{Key: "$bucket", Value: bson.D{
				{Key: "groupBy", Value: "$total_spent"},
				{Key: "boundaries", Value: bson.A{0, 500, 2000, 5000, 10000, 50000}},
				{Key: "default", Value: "50000+"},
				{Key: "output", Value: bson.D{
					{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
					{Key: "avg_orders", Value: bson.D{{Key: "$avg", Value: "$total_orders"}}},
					{Key: "total_spent", Value: bson.D{{Key: "$sum", Value: "$total_spent"}}},
					{Key: "avg_spent_per_customer", Value: bson.D{{Key: "$avg", Value: "$total_spent"}}},
					{Key: "min_spent", Value: bson.D{{Key: "$min", Value: "$total_spent"}}},
					{Key: "max_spent", Value: bson.D{{Key: "$max", Value: "$total_spent"}}},
				}},
			}},
		},
		bson.D{
			{Key: "$addFields", Value: bson.D{
				{Key: "segment", Value: bson.D{
					{Key: "$switch", Value: bson.D{
						{Key: "branches", Value: bson.A{
							bson.D{
								{Key: "case", Value: bson.D{{Key: "$eq", Value: bson.A{"$_id", 0}}}},
								{Key: "then", Value: "New (0-500)"},
							},
							bson.D{
								{Key: "case", Value: bson.D{{Key: "$eq", Value: bson.A{"$_id", 500}}}},
								{Key: "then", Value: "Regular (500-2000)"},
							},
							bson.D{
								{Key: "case", Value: bson.D{{Key: "$eq", Value: bson.A{"$_id", 2000}}}},
								{Key: "then", Value: "Loyal (2000-5000)"},
							},
							bson.D{
								{Key: "case", Value: bson.D{{Key: "$eq", Value: bson.A{"$_id", 5000}}}},
								{Key: "then", Value: "VIP (5000-10000)"},
							},
							bson.D{
								{Key: "case", Value: bson.D{{Key: "$eq", Value: bson.A{"$_id", 10000}}}},
								{Key: "then", Value: "Premium (10000-50000)"},
							},
						}},
						{Key: "default", Value: "Premium Plus (50000+)"},
					}},
				}},
			}},
		},
		bson.D{
			{Key: "$project", Value: bson.D{
				{Key: "_id", Value: "$segment"},
				{Key: "min_spent", Value: 1},
				{Key: "max_spent", Value: 1},
				{Key: "count", Value: 1},
				{Key: "avg_orders", Value: bson.D{{Key: "$round", Value: bson.A{"$avg_orders", 2}}}},
				{Key: "total_spent", Value: bson.D{{Key: "$round", Value: bson.A{"$total_spent", 2}}}},
				{Key: "avg_spent_per_customer", Value: bson.D{{Key: "$round", Value: bson.A{"$avg_spent_per_customer", 2}}}},
			}},
		},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var segments []CustomerSegment
	if err := cursor.All(ctx, &segments); err != nil {
		return nil, err
	}

	totalCustomers := 0
	for _, segment := range segments {
		totalCustomers += segment.CustomerCount
	}

	result := &CustomerSegmentsResult{
		Segments:       segments,
		TotalCustomers: totalCustomers,
	}

	return result, nil
}

// TopProduct represents product performance data
type TopProduct struct {
	ProductID    string  `json:"product_id" bson:"_id"`
	ProductName  string  `json:"product_name" bson:"product_name"`
	SKU          string  `json:"sku" bson:"sku"`
	TotalRevenue float64 `json:"total_revenue" bson:"total_revenue"`
	TotalSold    int     `json:"total_sold" bson:"total_sold"`
	AvgPrice     float64 `json:"avg_price" bson:"avg_price"`
	OrderCount   int     `json:"order_count" bson:"order_count"`
}

// InventoryStatus represents real-time inventory data
type InventoryStatus struct {
	ProductID    string    `json:"product_id" bson:"_id"`
	ProductName  string    `json:"product_name" bson:"product_name"`
	SKU          string    `json:"sku" bson:"sku"`
	Category     string    `json:"category" bson:"category"`
	CurrentStock int       `json:"current_stock" bson:"current_stock"`
	ReorderLevel int       `json:"reorder_level" bson:"reorder_level"`
	StockStatus  string    `json:"stock_status" bson:"stock_status"`
	LastUpdated  time.Time `json:"last_updated" bson:"last_updated"`
}

// GetTopProductsByRevenue returns top N products by revenue or quantity
func GetTopProductsByRevenue(limit int, sortBy string, startDate, endDate string) ([]TopProduct, error) {
	ctx, cancel := global.GetDefaultTimer()
	defer cancel()

	collection := GetCollection("orders")

	// Build match stage for completed orders
	matchStage := bson.M{
		"status": bson.M{"$in": []string{"shipped", "delivered", "completed"}},
	}

	// Add date range filtering if provided
	if startDate != "" || endDate != "" {
		dateFilter := bson.M{}
		if startDate != "" {
			startTime, err := time.Parse("2006-01-02", startDate)
			if err == nil {
				dateFilter["$gte"] = startTime
			}
		}
		if endDate != "" {
			endTime, err := time.Parse("2006-01-02", endDate)
			if err == nil {
				dateFilter["$lt"] = endTime.Add(24 * time.Hour)
			}
		}
		if len(dateFilter) > 0 {
			matchStage["created_at"] = dateFilter
		}
	}

	// Determine sort field
	sortField := "total_revenue"
	if sortBy == "quantity" {
		sortField = "total_sold"
	}

	// Build aggregation pipeline
	pipeline := []bson.M{
		{"$match": matchStage},
		{"$unwind": "$items"},
		{"$group": bson.M{
			"_id":           "$items.product_id",
			"product_name":  bson.M{"$first": "$items.product_name"},
			"sku":           bson.M{"$first": "$items.sku"},
			"total_revenue": bson.M{"$sum": bson.M{"$multiply": []interface{}{"$items.quantity", "$items.price"}}},
			"total_sold":    bson.M{"$sum": "$items.quantity"},
			"avg_price":     bson.M{"$avg": "$items.price"},
			"order_count":   bson.M{"$sum": 1},
		}},
		{"$project": bson.M{
			"_id":           1,
			"product_name":  1,
			"sku":           1,
			"total_revenue": bson.M{"$round": []interface{}{"$total_revenue", 2}},
			"total_sold":    1,
			"avg_price":     bson.M{"$round": []interface{}{"$avg_price", 2}},
			"order_count":   1,
		}},
		{"$sort": bson.M{sortField: -1}},
		{"$limit": limit},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var topProducts []TopProduct
	if err := cursor.All(ctx, &topProducts); err != nil {
		return nil, err
	}

	return topProducts, nil
}

// GetInventoryStatus returns real-time inventory status with alerts
func GetInventoryStatus(alertsOnly bool) ([]InventoryStatus, error) {
	ctx, cancel := global.GetDefaultTimer()
	defer cancel()

	collection := GetCollection("products")

	// Build match stage
	matchStage := bson.M{
		"status": "active",
	}

	// If alertsOnly is true, only show products below reorder level
	if alertsOnly {
		matchStage["$expr"] = bson.M{
			"$lt": []interface{}{"$stock.total", "$stock.reorder_level"},
		}
	}

	// Build aggregation pipeline
	pipeline := []bson.M{
		{"$match": matchStage},
		{"$addFields": bson.M{
			"stock_status": bson.M{
				"$switch": bson.M{
					"branches": []bson.M{
						{
							"case": bson.M{"$eq": []interface{}{"$stock.total", 0}},
							"then": "out_of_stock",
						},
						{
							"case": bson.M{"$lt": []interface{}{"$stock.total", "$stock.reorder_level"}},
							"then": "low_stock",
						},
						{
							"case": bson.M{"$lt": []interface{}{"$stock.total", bson.M{"$multiply": []interface{}{"$stock.reorder_level", 2}}}},
							"then": "medium_stock",
						},
					},
					"default": "in_stock",
				},
			},
		}},
		{"$project": bson.M{
			"_id":           1,
			"product_name":  "$name",
			"sku":           1,
			"category":      1,
			"current_stock": "$stock.total",
			"reorder_level": "$stock.reorder_level",
			"stock_status":  1,
			"last_updated":  "$updated_at",
		}},
		{"$sort": bson.M{"current_stock": 1}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var inventory []InventoryStatus
	if err := cursor.All(ctx, &inventory); err != nil {
		return nil, err
	}

	return inventory, nil
}

// GetSalesAnalytics retrieves sales data with grouping by day, week, or month
func GetSalesAnalytics(startDate, endDate, groupBy string) ([]SalesData, error) {
	ctx, cancel := global.GetDefaultTimer()
	defer cancel()

	collection := GetCollection("orders")

	// Build match stage for date filtering
	matchStage := bson.M{
		"status": bson.M{"$in": []string{"shipped", "delivered", "completed"}},
	}

	// Add date range filtering if provided
	if startDate != "" || endDate != "" {
		dateFilter := bson.M{}
		if startDate != "" {
			startTime, err := time.Parse("2006-01-02", startDate)
			if err == nil {
				dateFilter["$gte"] = startTime
			}
		}
		if endDate != "" {
			endTime, err := time.Parse("2006-01-02", endDate)
			if err == nil {
				// Add 24 hours to include the entire end date
				dateFilter["$lt"] = endTime.Add(24 * time.Hour)
			}
		}
		if len(dateFilter) > 0 {
			matchStage["created_at"] = dateFilter
		}
	}

	// Build group stage based on groupBy parameter
	var groupStage bson.M
	switch groupBy {
	case "week":
		groupStage = bson.M{
			"$group": bson.M{
				"_id": bson.M{
					"year": bson.M{"$year": "$created_at"},
					"week": bson.M{"$week": "$created_at"},
				},
				"total_orders":     bson.M{"$sum": 1},
				"total_revenue":    bson.M{"$sum": "$totals.grand_total"},
				"unique_customers": bson.M{"$addToSet": "$customer_id"},
			},
		}
	case "month":
		groupStage = bson.M{
			"$group": bson.M{
				"_id": bson.M{
					"year":  bson.M{"$year": "$created_at"},
					"month": bson.M{"$month": "$created_at"},
				},
				"total_orders":     bson.M{"$sum": 1},
				"total_revenue":    bson.M{"$sum": "$totals.grand_total"},
				"unique_customers": bson.M{"$addToSet": "$customer_id"},
			},
		}
	default: // day
		groupStage = bson.M{
			"$group": bson.M{
				"_id": bson.M{
					"year":  bson.M{"$year": "$created_at"},
					"month": bson.M{"$month": "$created_at"},
					"day":   bson.M{"$dayOfMonth": "$created_at"},
				},
				"total_orders":     bson.M{"$sum": 1},
				"total_revenue":    bson.M{"$sum": "$totals.grand_total"},
				"unique_customers": bson.M{"$addToSet": "$customer_id"},
			},
		}
	}

	// Add projection stage to calculate average order value and format date
	projectionStage := bson.M{
		"$project": bson.M{
			"_id":              formatDateProjection(groupBy),
			"total_orders":     1,
			"total_revenue":    bson.M{"$round": []interface{}{"$total_revenue", 2}},
			"avg_order_value":  bson.M{"$round": []interface{}{bson.M{"$divide": []interface{}{"$total_revenue", "$total_orders"}}, 2}},
			"unique_customers": bson.M{"$size": "$unique_customers"},
		},
	}

	// Sort stage
	sortStage := bson.M{
		"$sort": bson.M{"_id": 1},
	}

	// Build aggregation pipeline
	pipeline := []bson.M{
		{"$match": matchStage},
		groupStage,
		projectionStage,
		sortStage,
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var salesData []SalesData
	if err := cursor.All(ctx, &salesData); err != nil {
		return nil, err
	}

	return salesData, nil
}

// formatDateProjection returns the appropriate date formatting based on groupBy
func formatDateProjection(groupBy string) bson.M {
	switch groupBy {
	case "week":
		return bson.M{
			"$dateToString": bson.M{
				"format": "Week %V, %Y",
				"date":   bson.M{"$dateFromParts": bson.M{"isoWeekYear": "$_id.year", "isoWeek": "$_id.week"}},
			},
		}
	case "month":
		return bson.M{
			"$dateToString": bson.M{
				"format": "%B %Y",
				"date":   bson.M{"$dateFromParts": bson.M{"year": "$_id.year", "month": "$_id.month"}},
			},
		}
	default: // day
		return bson.M{
			"$dateToString": bson.M{
				"format": "%Y-%m-%d",
				"date":   bson.M{"$dateFromParts": bson.M{"year": "$_id.year", "month": "$_id.month", "day": "$_id.day"}},
			},
		}
	}
}
