#!/bin/bash

# Test AI Analytics Endpoints
# Make sure to set your Azure OpenAI credentials in .env file first

BASE_URL="http://localhost:8000/api/analytics/ai"

echo "Testing AI Analytics Endpoints..."
echo "=================================="
echo

echo "1. Testing AI Sales Report:"
curl -s "$BASE_URL/sales-report?startDate=2025-11-01&endDate=2025-11-30" | jq '.'
echo
echo

echo "2. Testing AI Customer Insights:"
curl -s "$BASE_URL/customer-insights" | jq '.'
echo
echo

echo "3. Testing AI Inventory Report:"
curl -s "$BASE_URL/inventory-report?alertsOnly=true" | jq '.'
echo
echo

echo "4. Testing AI Product Analysis:"
curl -s "$BASE_URL/product-analysis?limit=5&sortBy=revenue" | jq '.'
echo
echo

echo "AI Analytics testing complete!"