package main

import (
	"fmt"
	"time"

	"julianmorley.ca/con-plar/prog2270/pkg/models"
)

func main() {
	// Test SKU generation uniqueness
	fmt.Println("Testing SKU Generation...")
	fmt.Println("========================")

	// Create multiple product requests quickly to test uniqueness
	products := []models.CreateProductRequest{
		{Name: "Test Product 1", Brand: "TestBrand", Category: "Electronics"},
		{Name: "Test Product 2", Brand: "TestBrand", Category: "Electronics"},
		{Name: "Test Product 3", Brand: "TestBrand", Category: "Electronics"},
		{Name: "Test Product 4", Brand: "TestBrand", Category: "Electronics"},
		{Name: "Test Product 5", Brand: "TestBrand", Category: "Electronics"},
	}

	skus := make(map[string]bool)
	duplicates := 0

	for i, product := range products {
		sku := product.GenerateSKU()
		fmt.Printf("Product %d SKU: %s\n", i+1, sku)

		if skus[sku] {
			fmt.Printf("⚠️  DUPLICATE SKU FOUND: %s\n", sku)
			duplicates++
		}
		skus[sku] = true

		// Small delay to ensure different timestamps
		time.Sleep(1 * time.Nanosecond)
	}

	fmt.Printf("\nResults: %d unique SKUs generated, %d duplicates\n", len(skus), duplicates)
	if duplicates == 0 {
		fmt.Println("✅ All SKUs are unique!")
	} else {
		fmt.Println("❌ Duplicates found - need to improve algorithm")
	}
}
