# E-Commerce Analytics API

A high-performance Go API for e-commerce analytics platform with MongoDB, Redis caching, and AI-powered insights.

## 🚀 Quick Start

### Prerequisites
- Go 1.21+ 
- MongoDB (local) or MongoDB Atlas
- Redis 7.0+
- OpenAI API key (for AI features)

### Environment Setup
1. Copy environment template:
```bash
cp .env.example .env
```

2. Configure environment variables:
```env
# Server Configuration
PORT=8080
ENV=development

# MongoDB Configuration
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=ecommerce

# Redis Configuration  
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# AI Integration
OPENAI_API_KEY=sk-your-openai-api-key-here
AI_MODEL=gpt-4

# Logging
LOG_LEVEL=info
```

### Installation & Running
```bash
# Install dependencies
go mod download

# Build the application
go build -o server cmd/main.go

# Run the server
./server

# Or run directly
go run cmd/main.go
```

Server will start on `http://localhost:8080`

## 📡 API Endpoints

### Health Check
```
GET /api/health
```

### Search
```
GET /api/search?q=query&category=Electronics&limit=10
```

### Products
```
GET    /api/products              # List products (with filters)
POST   /api/products              # Create product
GET    /api/products/:id          # Get product by ID  
PUT    /api/products/:id          # Update product
DELETE /api/products/:id          # Delete product
```

### Categories
```
GET /api/categories               # List all categories
```

### Orders
```
GET    /api/orders                # List orders
POST   /api/orders                # Create order
GET    /api/orders/:id            # Get order details
PUT    /api/orders/:id            # Update order
DELETE /api/orders/:id            # Delete order
```

### Customers
```
GET    /api/customers             # List all customers
POST   /api/customers             # Create customer
GET    /api/customers/:id         # Get customer details
DELETE /api/customers/:id         # Delete customer
GET    /api/customers/:id/orders  # Customer order history
```

### Shopping Cart (Redis-based)
```
GET    /api/cart/:sessionId       # Get cart contents
POST   /api/cart/:sessionId/items # Add item to cart
PUT    /api/cart/:sessionId/items/:sku # Update cart item
DELETE /api/cart/:sessionId/items/:sku # Remove from cart
DELETE /api/cart/:sessionId       # Clear entire cart
```

### Analytics
```
GET /api/analytics/sales?period=daily&start=2025-11-01&end=2025-11-30
GET /api/analytics/top-products?sort=revenue&limit=10
GET /api/analytics/inventory?alerts=true&threshold=10
GET /api/analytics/customers?segment=all
```

### AI-Powered Analytics
```
GET /api/ai/sales-report?period=weekly
GET /api/ai/customer-insights?segment=high-value  
GET /api/ai/inventory-report?category=Electronics
GET /api/ai/product-analysis?sku=ELEC-LAPTOP-001
```

## 🔧 Configuration

### MongoDB Setup
The API supports both local MongoDB and MongoDB Atlas:

**Local MongoDB:**
```env
MONGODB_URI=mongodb://localhost:27017
```

**MongoDB Atlas:**
```env
MONGODB_URI=mongodb+srv://username:password@cluster.xxxxx.mongodb.net/ecommerce?retryWrites=true&w=majority
```

### Redis Configuration
**Local Redis:**
```env
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
```

**Redis Cloud:**
```env
REDIS_ADDR=redis-12345.c123.us-east-1-1.ec2.cloud.redislabs.com:12345
REDIS_PASSWORD=your-redis-password
REDIS_DB=0
```

### AI Integration Setup
1. Get OpenAI API key from [OpenAI Platform](https://platform.openai.com/)
2. Set environment variable:
```env
OPENAI_API_KEY=sk-your-actual-api-key-here
```

## 📊 Sample Data Loading

### Load Sample Data
```bash
# Load products (300+ items)
curl -X POST http://localhost:8080/api/data/load/products

# Load customers (20+ profiles)  
curl -X POST http://localhost:8080/api/data/load/customers

# Load orders (sample orders)
curl -X POST http://localhost:8080/api/data/load/orders
```

### Manual Data Files
Sample data files are located in `../data/`:
- `products_300.json` - 300+ diverse products
- `customers_20.json` - 20 customer profiles
- `orders_5.json` - Sample order data
- `reviews_8.json` - Product reviews
- `inventory_logs_12.json` - Inventory change logs

## 🧪 Testing

### API Testing with curl
```bash
# Test health endpoint
curl http://localhost:8080/api/health

# Search products
curl "http://localhost:8080/api/search?q=laptop&limit=5"

# Get analytics
curl "http://localhost:8080/api/analytics/top-products?limit=10"

# Test cart operations
curl -X POST http://localhost:8080/api/cart/session123/items \
  -H "Content-Type: application/json" \
  -d '{"sku":"ELEC-LAPTOP-001","quantity":1}'
```

### Load Testing
```bash
# Install vegeta (load testing tool)
go install github.com/tsenart/vegeta@latest

# Test product endpoint
echo "GET http://localhost:8080/api/products" | vegeta attack -duration=30s -rate=100 | vegeta report
```

## 🚀 Performance Features

### Redis Caching
- **Product Cache:** 10-minute TTL for product details
- **Cart Sessions:** 2-hour TTL for shopping carts  
- **Analytics Cache:** 15-minute TTL for aggregated results
- **Search Cache:** 5-minute TTL for search results

### MongoDB Optimization
- **Strategic Indexes:** 5+ compound indexes for common queries
- **Aggregation Pipelines:** Optimized for real-time analytics
- **Connection Pooling:** Efficient database connection management

### Cache-Aside Pattern
```go
// Example: Product caching with fallback
func GetProduct(id string) (*Product, error) {
    // 1. Try Redis cache first
    cached, err := redis.GetProduct(id)
    if err == nil {
        return cached, nil
    }
    
    // 2. Fallback to MongoDB
    product, err := mongo.FindProduct(id)
    if err != nil {
        return nil, err
    }
    
    // 3. Cache for future requests
    redis.SetProduct(id, product, 10*time.Minute)
    return product, nil
}
```

## 📈 Monitoring & Debugging

### Logging
Set log level in environment:
```env
LOG_LEVEL=debug  # debug, info, warn, error
```

### Health Monitoring
```bash
# Check application health
curl http://localhost:8080/api/health

# Response includes:
# - Database connectivity
# - Redis connectivity  
# - AI service status
# - Memory usage
# - Uptime
```

### Performance Metrics
```bash
# Get cache hit rates
curl http://localhost:8080/api/metrics/cache

# Database query performance  
curl http://localhost:8080/api/metrics/database

# API response times
curl http://localhost:8080/api/metrics/endpoints
```

## 🏗️ Architecture

### Project Structure
```
cmd/
├── main.go                 # Application entry point
internal/
├── router/
│   ├── engine.go          # Route definitions
│   └── handler.go         # HTTP handlers
pkg/
├── mongo/
│   ├── analytics.go       # Analytics aggregations
│   └── helpers.go         # Database operations
├── redis/
│   └── helpers.go         # Cache operations
├── models/
│   ├── product.go         # Data models
│   ├── order.go
│   ├── customer.go
│   └── cart.go
└── ai/
    ├── client.go          # AI service client
    ├── reports.go         # AI report generation
    └── prompts.go         # AI prompt templates
```

### Key Design Patterns
- **Repository Pattern:** Clean data layer separation
- **Cache-Aside:** Performance optimization with Redis
- **Dependency Injection:** Testable service architecture
- **Middleware:** Cross-cutting concerns (logging, CORS, auth)

## 🔍 Troubleshooting

### Common Issues

**MongoDB Connection:**
```bash
# Check MongoDB status
mongo --eval "db.runCommand('ping')"

# Check Atlas connectivity
mongosh "mongodb+srv://cluster.xxxxx.mongodb.net/test"
```

**Redis Connection:**
```bash
# Check Redis status
redis-cli ping

# Monitor Redis operations
redis-cli monitor
```

**API Issues:**
```bash
# Check server logs
tail -f /var/log/ecommerce-api.log

# Test specific endpoint
curl -v http://localhost:8080/api/products
```

### Performance Issues
1. **Slow Queries:** Check MongoDB indexes with explain()
2. **Cache Misses:** Monitor Redis hit ratios
3. **Memory Usage:** Use Go pprof for profiling

### Development Tips
- Use `air` for auto-reloading during development
- Enable debug logging for detailed request tracing
- Use MongoDB Compass for database inspection
- Use RedisInsight for cache monitoring

## 📝 API Documentation

For complete API documentation with examples, see:
- Postman collection: `../postman/ecommerce_api.postman_collection.json`
- OpenAPI spec: `docs/api-spec.yaml`

## 🤝 Contributing

1. Follow Go coding standards
2. Add tests for new features  
3. Update documentation
4. Use conventional commits

## 📄 License

This project is part of PROG2270 Advanced Data Systems coursework at Conestoga College.