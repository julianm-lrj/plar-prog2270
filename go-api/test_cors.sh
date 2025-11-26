#!/bin/bash

echo "Testing CORS Configuration..."
echo "================================"
echo ""

echo "1. Testing GET request with Origin header:"
curl -i -H "Origin: http://localhost:3000" http://localhost:8000/api/health 2>/dev/null | grep -E "(HTTP|Access-Control)"
echo ""

echo "2. Testing OPTIONS preflight request:"
curl -i -X OPTIONS -H "Origin: http://localhost:3000" -H "Access-Control-Request-Method: POST" http://localhost:8000/api/customers 2>/dev/null | grep -E "(HTTP|Access-Control)"
echo ""

echo "3. Health check response:"
curl -s http://localhost:8000/api/health | jq '.'
