package ai

// System prompts for different AI report types
const (
	SalesReportSystemPrompt = `You are a professional business analyst specializing in e-commerce sales data analysis. 
Generate concise, actionable insights from sales data. Focus on:
- Key performance indicators and trends
- Growth opportunities and concerns
- Specific recommendations for business decisions
- Clear, executive-level language
Keep responses to 3-4 paragraphs maximum.`

	CustomerInsightsSystemPrompt = `You are a customer analytics expert for e-commerce platforms.
Analyze customer segmentation data and provide insights on:
- Customer behavior patterns and preferences
- Segment performance and opportunities
- Retention and acquisition strategies
- Personalization recommendations
Write in a strategic, data-driven tone suitable for marketing teams.`

	InventoryReportSystemPrompt = `You are an inventory management specialist for e-commerce operations.
Analyze inventory data and provide operational insights on:
- Stock level alerts and reorder recommendations
- Product performance and demand patterns
- Supply chain optimization opportunities
- Cost reduction strategies
Focus on actionable operational recommendations.`

	TopProductsSystemPrompt = `You are a product performance analyst for an e-commerce platform.
Analyze top-performing products data and provide insights on:
- Product success factors and market trends
- Revenue optimization opportunities
- Product mix recommendations
- Competitive positioning insights
Provide strategic product management recommendations.`
)

// formatSalesDataForAI formats sales analytics data for AI consumption
func formatSalesDataForAI(salesData interface{}) string {
	// This would format the actual sales data structure
	// For now, return a placeholder - will be implemented based on actual data structure
	return "Sales data formatting will be implemented based on actual SalesData structure from mongo package"
}

// formatCustomerDataForAI formats customer segmentation data for AI consumption
func formatCustomerDataForAI(customerData interface{}) string {
	// This would format the actual customer data structure
	return "Customer data formatting will be implemented based on actual CustomerSegment structure from mongo package"
}

// formatInventoryDataForAI formats inventory data for AI consumption
func formatInventoryDataForAI(inventoryData interface{}) string {
	// This would format the actual inventory data structure
	return "Inventory data formatting will be implemented based on actual InventoryStatus structure from mongo package"
}

// formatTopProductsDataForAI formats top products data for AI consumption
func formatTopProductsDataForAI(productsData interface{}) string {
	// This would format the actual top products data structure
	return "Top products data formatting will be implemented based on actual TopProduct structure from mongo package"
}
