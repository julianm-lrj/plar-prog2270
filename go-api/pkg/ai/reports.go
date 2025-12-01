package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"julianmorley.ca/con-plar/prog2270/pkg/mongo"
)

// AIReportResponse represents the structure of AI-generated reports
type AIReportResponse struct {
	Status      string     `json:"status"`
	Data        ReportData `json:"data"`
	GeneratedAt time.Time  `json:"generated_at"`
	AIEnabled   bool       `json:"ai_enabled"`
}

type ReportData struct {
	RawData    interface{} `json:"raw_data"`
	AIInsights string      `json:"ai_insights,omitempty"`
	Summary    string      `json:"summary"`
	Error      string      `json:"error,omitempty"`
}

// GenerateSalesReport generates AI-powered insights from sales analytics data
func GenerateSalesReport(ctx context.Context, startDate, endDate string) (*AIReportResponse, error) {
	// Fetch sales data using existing mongo functions
	// Default to daily grouping if no specific grouping is needed
	groupBy := "day"
	salesData, err := mongo.GetSalesAnalytics(startDate, endDate, groupBy)
	if err != nil {
		return &AIReportResponse{
			Status:      "error",
			Data:        ReportData{Error: "Failed to fetch sales data: " + err.Error()},
			GeneratedAt: time.Now(),
			AIEnabled:   IsEnabled(),
		}, err
	}

	response := &AIReportResponse{
		Status:      "success",
		GeneratedAt: time.Now(),
		AIEnabled:   IsEnabled(),
		Data: ReportData{
			RawData: salesData,
			Summary: "Sales data retrieved successfully",
		},
	}

	// Generate AI insights if service is enabled
	if IsEnabled() {
		userPrompt := formatSalesDataPrompt(salesData)
		aiInsights, err := generateCompletion(ctx, SalesReportSystemPrompt, userPrompt)
		if err != nil {
			response.Data.Error = "AI analysis failed: " + err.Error()
		} else {
			response.Data.AIInsights = aiInsights
			response.Data.Summary = "AI-generated sales insights and recommendations"
		}
	} else {
		response.Data.Summary = "Raw sales data (AI insights unavailable)"
	}

	return response, nil
}

// GenerateCustomerInsights generates AI-powered customer segmentation analysis
func GenerateCustomerInsights(ctx context.Context) (*AIReportResponse, error) {
	// Fetch customer segments using existing mongo functions
	customerData, err := mongo.GetCustomerSpendingSegments(ctx)
	if err != nil {
		return &AIReportResponse{
			Status:      "error",
			Data:        ReportData{Error: "Failed to fetch customer data: " + err.Error()},
			GeneratedAt: time.Now(),
			AIEnabled:   IsEnabled(),
		}, err
	}

	response := &AIReportResponse{
		Status:      "success",
		GeneratedAt: time.Now(),
		AIEnabled:   IsEnabled(),
		Data: ReportData{
			RawData: customerData,
			Summary: "Customer segmentation data retrieved successfully",
		},
	}

	if IsEnabled() {
		userPrompt := formatCustomerDataPrompt(customerData)
		aiInsights, err := generateCompletion(ctx, CustomerInsightsSystemPrompt, userPrompt)
		if err != nil {
			response.Data.Error = "AI analysis failed: " + err.Error()
		} else {
			response.Data.AIInsights = aiInsights
			response.Data.Summary = "AI-generated customer insights and recommendations"
		}
	} else {
		response.Data.Summary = "Raw customer data (AI insights unavailable)"
	}

	return response, nil
}

// GenerateInventoryReport generates AI-powered inventory analysis
func GenerateInventoryReport(ctx context.Context, alertsOnly bool) (*AIReportResponse, error) {
	// Fetch inventory data using existing mongo functions
	inventoryData, err := mongo.GetInventoryStatus(alertsOnly)
	if err != nil {
		return &AIReportResponse{
			Status:      "error",
			Data:        ReportData{Error: "Failed to fetch inventory data: " + err.Error()},
			GeneratedAt: time.Now(),
			AIEnabled:   IsEnabled(),
		}, err
	}

	response := &AIReportResponse{
		Status:      "success",
		GeneratedAt: time.Now(),
		AIEnabled:   IsEnabled(),
		Data: ReportData{
			RawData: inventoryData,
			Summary: "Inventory status data retrieved successfully",
		},
	}

	if IsEnabled() {
		userPrompt := formatInventoryDataPrompt(inventoryData, alertsOnly)
		aiInsights, err := generateCompletion(ctx, InventoryReportSystemPrompt, userPrompt)
		if err != nil {
			response.Data.Error = "AI analysis failed: " + err.Error()
		} else {
			response.Data.AIInsights = aiInsights
			response.Data.Summary = "AI-generated inventory insights and recommendations"
		}
	} else {
		response.Data.Summary = "Raw inventory data (AI insights unavailable)"
	}

	return response, nil
}

// GenerateTopProductsAnalysis generates AI-powered top products analysis
func GenerateTopProductsAnalysis(ctx context.Context, limit int, sortBy, startDate, endDate string) (*AIReportResponse, error) {
	// Fetch top products data using existing mongo functions
	topProducts, err := mongo.GetTopProductsByRevenue(limit, sortBy, startDate, endDate)
	if err != nil {
		return &AIReportResponse{
			Status:      "error",
			Data:        ReportData{Error: "Failed to fetch top products data: " + err.Error()},
			GeneratedAt: time.Now(),
			AIEnabled:   IsEnabled(),
		}, err
	}

	response := &AIReportResponse{
		Status:      "success",
		GeneratedAt: time.Now(),
		AIEnabled:   IsEnabled(),
		Data: ReportData{
			RawData: topProducts,
			Summary: "Top products data retrieved successfully",
		},
	}

	if IsEnabled() {
		userPrompt := formatTopProductsDataPrompt(topProducts, sortBy, limit)
		aiInsights, err := generateCompletion(ctx, TopProductsSystemPrompt, userPrompt)
		if err != nil {
			response.Data.Error = "AI analysis failed: " + err.Error()
		} else {
			response.Data.AIInsights = aiInsights
			response.Data.Summary = "AI-generated top products insights and recommendations"
		}
	} else {
		response.Data.Summary = "Raw top products data (AI insights unavailable)"
	}

	return response, nil
}

// Helper functions to format data for AI prompts

func formatSalesDataPrompt(salesData interface{}) string {
	jsonData, _ := json.MarshalIndent(salesData, "", "  ")
	return fmt.Sprintf(`Analyze the following sales analytics data and provide business insights:

%s

Please provide:
1. Key performance highlights and trends
2. Areas of concern or opportunity
3. Specific recommendations for business growth
4. Actionable next steps for the management team`, string(jsonData))
}

func formatCustomerDataPrompt(customerData interface{}) string {
	jsonData, _ := json.MarshalIndent(customerData, "", "  ")
	return fmt.Sprintf(`Analyze the following customer segmentation data and provide insights:

%s

Please provide:
1. Customer behavior patterns and trends
2. High-value segment opportunities
3. Retention and acquisition strategies
4. Personalization recommendations for each segment`, string(jsonData))
}

func formatInventoryDataPrompt(inventoryData interface{}, alertsOnly bool) string {
	jsonData, _ := json.MarshalIndent(inventoryData, "", "  ")
	alertsContext := ""
	if alertsOnly {
		alertsContext = " (This data shows only products requiring immediate attention)"
	}

	return fmt.Sprintf(`Analyze the following inventory status data%s and provide operational insights:

%s

Please provide:
1. Immediate actions required for stock management
2. Demand patterns and forecasting insights
3. Supply chain optimization opportunities
4. Cost reduction recommendations`, alertsContext, string(jsonData))
}

func formatTopProductsDataPrompt(productsData interface{}, sortBy string, limit int) string {
	jsonData, _ := json.MarshalIndent(productsData, "", "  ")
	return fmt.Sprintf(`Analyze the following top %d products data (sorted by %s) and provide strategic insights:

%s

Please provide:
1. Success factors driving top product performance
2. Market trends and opportunities identified
3. Product mix optimization recommendations
4. Competitive positioning strategies`, limit, sortBy, string(jsonData))
}
