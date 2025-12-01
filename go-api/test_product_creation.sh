#!/bin/bash

echo "Testing Product Creation with Redis Caching..."
echo "=============================================="
echo ""

# Sample product data
cat << 'EOF' > /tmp/sample_products.json
[
  {
    "name": "Premium Wireless Headphones",
    "description": "High-quality wireless headphones with noise cancellation and 30-hour battery life.",
    "category": "Electronics",
    "subcategory": "Audio",
    "brand": "TechSound",
    "price": 299.99,
    "currency": "CAD",
    "images": [
      "https://example.com/headphones1.jpg",
      "https://example.com/headphones2.jpg"
    ],
    "attributes": {
      "color": "Black",
      "wireless": "true",
      "battery_life": "30 hours",
      "noise_cancellation": "true",
      "weight": "250g"
    },
    "tags": ["wireless", "headphones", "audio", "premium", "noise-cancelling"]
  },
  {
    "name": "Organic Cotton T-Shirt",
    "description": "Comfortable organic cotton t-shirt in various colors and sizes.",
    "category": "Clothing",
    "subcategory": "Shirts",
    "brand": "EcoWear",
    "price": 29.99,
    "currency": "CAD",
    "images": [
      "https://example.com/tshirt1.jpg",
      "https://example.com/tshirt2.jpg"
    ],
    "attributes": {
      "material": "100% Organic Cotton",
      "size": "M",
      "color": "Navy Blue",
      "care": "Machine wash cold",
      "fit": "Regular"
    },
    "tags": ["organic", "cotton", "clothing", "sustainable", "casual"]
  }
]
EOF

echo "Sample products to create:"
cat /tmp/sample_products.json | jq '.'
echo ""

echo "Sending POST request to create products..."
response=$(curl -s -X POST http://localhost:8000/api/products \
  -H "Content-Type: application/json" \
  -d @/tmp/sample_products.json)

echo "Response:"
echo "$response" | jq '.'

# Clean up
rm /tmp/sample_products.json