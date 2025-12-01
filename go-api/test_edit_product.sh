#!/bin/bash

echo "Testing EditProductBySKU functionality..."
echo "========================================"
echo ""

# Step 1: Create a test product
echo "Step 1: Creating a test product..."
create_response=$(curl -s -X POST http://localhost:8000/api/products \
  -H "Content-Type: application/json" \
  -d '[{
    "name": "Original Product Name",
    "description": "Original description",
    "category": "Electronics", 
    "subcategory": "Testing",
    "brand": "TestBrand",
    "price": 99.99,
    "currency": "CAD",
    "images": ["https://example.com/original.jpg"],
    "attributes": {"color": "blue", "size": "large"},
    "tags": ["original", "test"]
  }]')

echo "Product creation response:"
echo "$create_response" | jq '.'
echo ""

# Extract SKU
product_sku=$(echo "$create_response" | jq -r '.data.products[0].sku // empty')

if [ -z "$product_sku" ] || [ "$product_sku" = "null" ]; then
    echo "❌ Failed to create test product or extract SKU"
    exit 1
fi

echo "✅ Test product created with SKU: $product_sku"
echo ""

# Step 2: Test partial update
echo "Step 2: Testing partial product update..."
update_response=$(curl -s -i -X PUT http://localhost:8000/api/products/$product_sku \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Product Name",
    "price": 149.99,
    "attributes": {"color": "red", "size": "medium", "material": "premium"},
    "tags": ["updated", "test", "premium"]
  }')

echo "Update response headers:"
echo "$update_response" | head -20 | grep -E "(HTTP|X-Cache|Content-Type)"
echo ""
echo "Update response body:"
echo "$update_response" | tail -n +$(echo "$update_response" | grep -n "^$" | head -1 | cut -d: -f1) | jq '.' 2>/dev/null || echo "No valid JSON response"
echo ""

# Step 3: Verify the update by getting the product (should be cache HIT)
echo "Step 3: Verifying update (should be cache HIT)..."
get_response=$(curl -s -i http://localhost:8000/api/products/$product_sku)
echo "Get response headers:"
echo "$get_response" | head -20 | grep -E "(HTTP|X-Cache|Content-Type)"
echo ""
echo "Product after update:"
echo "$get_response" | tail -n +$(echo "$get_response" | grep -n "^$" | head -1 | cut -d: -f1) | jq '.data | {name, price, attributes, tags}' 2>/dev/null || echo "No valid JSON response"
echo ""

# Step 4: Test error cases
echo "Step 4: Testing error cases..."
echo ""

# Test 4a: Empty update body
echo "Test 4a: Empty update body (should return 400)..."
empty_response=$(curl -s -i -X PUT http://localhost:8000/api/products/$product_sku \
  -H "Content-Type: application/json" \
  -d '{}')
echo "Response status:"
echo "$empty_response" | head -1
echo "Response body:"
echo "$empty_response" | tail -n +$(echo "$empty_response" | grep -n "^$" | head -1 | cut -d: -f1) | jq '.message // "No message"' 2>/dev/null
echo ""

# Test 4b: Try to update immutable field
echo "Test 4b: Try to update immutable field (should return 400)..."
immutable_response=$(curl -s -i -X PUT http://localhost:8000/api/products/$product_sku \
  -H "Content-Type: application/json" \
  -d '{
    "sku": "NEW-SKU-123",
    "name": "Should not work"
  }')
echo "Response status:"
echo "$immutable_response" | head -1
echo "Response body:"
echo "$immutable_response" | tail -n +$(echo "$immutable_response" | grep -n "^$" | head -1 | cut -d: -f1) | jq '.message // "No message"' 2>/dev/null
echo ""

# Test 4c: Non-existent SKU
echo "Test 4c: Non-existent SKU (should return 404)..."
notfound_response=$(curl -s -i -X PUT http://localhost:8000/api/products/NON-EXI-123456789 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "This should not work"
  }')
echo "Response status:"
echo "$notfound_response" | head -1
echo "Response body:"
echo "$notfound_response" | tail -n +$(echo "$notfound_response" | grep -n "^$" | head -1 | cut -d: -f1) | jq '.message // "No message"' 2>/dev/null

echo ""
echo "🏁 EditProductBySKU test completed!"
echo "   - Check that the product was updated correctly"
echo "   - Verify X-Cache: REFRESHED header on successful updates"
echo "   - Confirm error handling for invalid cases"