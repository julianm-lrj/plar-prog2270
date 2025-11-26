package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
)

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
