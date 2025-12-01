#!/bin/bash

echo "Testing BulkEditProducts functionality..."
echo "========================================"
echo ""

# Step 1: Create multiple test products
echo "Step 1: Creating multiple test products..."
create_response=$(curl -s -X POST http://localhost:8000/api/products \
  -H "Content-Type: application/json" \
  -d '[
    {
      "name": "Bulk Test Product 1",
      "description": "First product for bulk testing",
      "category": "Electronics", 
      "subcategory": "Testing",
      "brand": "BulkBrand",
      "price": 99.99,
      "currency": "CAD",
      "images": ["https://example.com/bulk1.jpg"],
      "attributes": {"color": "red", "size": "small"},
      "tags": ["bulk", "test", "first"]
    },
    {
      "name": "Bulk Test Product 2", 
      "description": "Second product for bulk testing",
      "category": "Clothing",
      "subcategory": "Testing", 
      "brand": "BulkBrand",
      "price": 149.99,
      "currency": "CAD",
      "images": ["https://example.com/bulk2.jpg"],
      "attributes": {"color": "blue", "size": "medium"},
      "tags": ["bulk", "test", "second"]
    },
    {
      "name": "Bulk Test Product 3",
      "description": "Third product for bulk testing", 
      "category": "Books",
      "subcategory": "Testing",
      "brand": "BulkBrand", 
      "price": 29.99,
      "currency": "CAD",
      "images": ["https://example.com/bulk3.jpg"],
      "attributes": {"color": "green", "format": "paperback"},
      "tags": ["bulk", "test", "third"]
    }
  ]')

echo "Products creation response:"
echo "$create_response" | jq '.'
echo ""

# Extract SKUs
sku1=$(echo "$create_response" | jq -r '.data.products[0].sku')
sku2=$(echo "$create_response" | jq -r '.data.products[1].sku')
sku3=$(echo "$create_response" | jq -r '.data.products[2].sku')

if [ -z "$sku1" ] || [ "$sku1" = "null" ]; then
    echo "❌ Failed to create test products or extract SKUs"
    exit 1
fi

echo "✅ Test products created with SKUs: $sku1, $sku2, $sku3"
echo ""

# Step 2: Test bulk update
echo "Step 2: Testing bulk product updates..."
bulk_update_response=$(curl -s -i -X PUT http://localhost:8000/api/products \
  -H "Content-Type: application/json" \
  -d "[
    {
      \"sku\": \"$sku1\",
      \"name\": \"Updated Bulk Product 1\",
      \"price\": 199.99,
      \"attributes\": {\"color\": \"purple\", \"size\": \"large\", \"updated\": \"true\"}
    },
    {
      \"sku\": \"$sku2\", 
      \"description\": \"Updated description for product 2\",
      \"tags\": [\"bulk\", \"updated\", \"awesome\"]
    },
    {
      \"sku\": \"$sku3\",
      \"price\": 39.99,
      \"attributes\": {\"color\": \"yellow\", \"format\": \"hardcover\", \"edition\": \"deluxe\"}
    }
  ]")

echo "Bulk update response headers:"
echo "$bulk_update_response" | head -20 | grep -E "(HTTP|X-Cache|Content-Type)"
echo ""
echo "Bulk update response body:"
echo "$bulk_update_response" | tail -n +$(echo "$bulk_update_response" | grep -n "^$" | head -1 | cut -d: -f1) | jq '.' 2>/dev/null || echo "No valid JSON response"
echo ""

# Step 3: Verify updates by getting individual products (should be cache HIT)
echo "Step 3: Verifying individual product updates (cache HITs)..."
for sku in $sku1 $sku2 $sku3; do
    echo "Getting product $sku:"
    get_response=$(curl -s -i http://localhost:8000/api/products/$sku)
    echo "Headers: $(echo "$get_response" | grep "X-Cache")"
    echo "Product: $(echo "$get_response" | tail -n +$(echo "$get_response" | grep -n "^$" | head -1 | cut -d: -f1) | jq '.data | {name, price, attributes}' 2>/dev/null)"
    echo ""
done

# Step 4: Test error cases
echo "Step 4: Testing bulk update error cases..."
echo ""

# Test 4a: Missing SKU in one update
echo "Test 4a: Missing SKU (should show partial success)..."
partial_error_response=$(curl -s -i -X PUT http://localhost:8000/api/products \
  -H "Content-Type: application/json" \
  -d "[
    {
      \"sku\": \"$sku1\",
      \"name\": \"Valid update\"
    },
    {
      \"name\": \"Missing SKU - should fail\",
      \"price\": 999.99
    }
  ]")
echo "Response status:"
echo "$partial_error_response" | head -1
echo "Response body:"
echo "$partial_error_response" | tail -n +$(echo "$partial_error_response" | grep -n "^$" | head -1 | cut -d: -f1) | jq '.data | {success_count, error_count, errors}' 2>/dev/null
echo ""

# Test 4b: Non-existent SKU
echo "Test 4b: Non-existent SKU (should show error)..."
notfound_response=$(curl -s -i -X PUT http://localhost:8000/api/products \
  -H "Content-Type: application/json" \
  -d "[
    {
      \"sku\": \"NON-EXI-123456789\",
      \"name\": \"This should fail\"
    }
  ]")
echo "Response status:"
echo "$notfound_response" | head -1
echo "Response body:"
echo "$notfound_response" | tail -n +$(echo "$notfound_response" | grep -n "^$" | head -1 | cut -d: -f1) | jq '.data | {success_count, error_count, errors}' 2>/dev/null

echo ""
echo "🏁 BulkEditProducts test completed!"
echo "   - Check that multiple products were updated correctly"
echo "   - Verify X-Cache: BULK-REFRESHED header on successful updates" 
echo "   - Confirm error handling for missing SKUs and non-existent products"
echo "   - Note the success_count and error_count in responses"