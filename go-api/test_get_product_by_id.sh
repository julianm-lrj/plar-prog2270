#!/bin/bash

echo "Testing GetProductBySKU with Redis Caching..."
echo "============================================"
echo ""

# First, let's get a product SKU from the created product
echo "Step 1: Creating a test product to get its SKU..."

# Create a test product first
echo "Creating a test product..."
response=$(curl -s -X POST http://localhost:8000/api/products \
  -H "Content-Type: application/json" \
  -d '[{
    "name": "Test Redis Product",
    "description": "A test product for Redis caching",
    "category": "Electronics", 
    "subcategory": "Testing",
    "brand": "TestBrand",
    "price": 99.99,
    "currency": "CAD",
    "images": ["https://example.com/test.jpg"],
    "attributes": {"color": "red", "size": "medium"},
    "tags": ["test", "redis", "cache"]
  }]')

echo "Product creation response:"
echo "$response" | jq '.'
echo ""

# Extract product SKU
product_sku=$(echo "$response" | jq -r '.data.products[0].sku // empty')

if [ -z "$product_sku" ] || [ "$product_sku" = "null" ]; then
    echo "❌ Failed to create test product or extract SKU"
    exit 1
fi

echo "✅ Test product created with SKU: $product_sku"
echo ""

# Test 1: First request (should be MISS from cache)
echo "Test 1: First request (should be cache MISS)..."
response1=$(curl -s -i http://localhost:8000/api/products/$product_sku)
echo "Response headers:"
echo "$response1" | head -20 | grep -E "(HTTP|X-Cache|Content-Type)"
echo ""
echo "Response body:"
echo "$response1" | tail -n +$(echo "$response1" | grep -n "^$" | head -1 | cut -d: -f1) | jq '.' 2>/dev/null || echo "$response1" | tail -n +$(echo "$response1" | grep -n "^$" | head -1 | cut -d: -f1)
echo ""

# Test 2: Second request (should be HIT from cache)
echo "Test 2: Second request (should be cache HIT)..."
response2=$(curl -s -i http://localhost:8000/api/products/$product_sku)
echo "Response headers:"
echo "$response2" | head -20 | grep -E "(HTTP|X-Cache|Content-Type)"
echo ""

# Test 3: Invalid SKU format
echo "Test 3: Invalid SKU format (should return 400)..."
response3=$(curl -s -i http://localhost:8000/api/products/x)
echo "Response status:"
echo "$response3" | head -1
echo "Response body:"
echo "$response3" | tail -n +$(echo "$response3" | grep -n "^$" | head -1 | cut -d: -f1) | jq '.' 2>/dev/null || echo "No valid JSON response"
echo ""

# Test 4: Non-existent but valid SKU format
echo "Test 4: Valid SKU format but non-existent product (should return 404)..."
fake_sku="NON-EXI-123456789abcdef"
response4=$(curl -s -i http://localhost:8000/api/products/$fake_sku)
echo "Response status:"
echo "$response4" | head -1
echo "Response body:"
echo "$response4" | tail -n +$(echo "$response4" | grep -n "^$" | head -1 | cut -d: -f1) | jq '.' 2>/dev/null || echo "No valid JSON response"

echo ""
echo "🏁 Test completed! Check the X-Cache headers to verify Redis caching behavior."
echo "   - First request should show X-Cache: MISS"
echo "   - Second request should show X-Cache: HIT"