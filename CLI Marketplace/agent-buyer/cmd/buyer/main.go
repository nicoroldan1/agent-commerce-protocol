package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/nicroldan/ans/agent-buyer/internal/client"
	"github.com/nicroldan/ans/shared/ace"
)

func main() {
	registryURL := flag.String("registry", "http://localhost:8080", "Registry URL")
	storeURL := flag.String("store", "", "Direct store well-known URL (skips registry)")
	apiKey := flag.String("key", "", "ACE API key for the store")
	provider := flag.String("provider", "stripe", "Payment provider")
	flag.Parse()

	fmt.Println("=== ANS Demo Agent Buyer ===")
	fmt.Println()

	// Step 1: Discover a store
	var wellKnownURL string

	if *storeURL != "" {
		fmt.Println("Step 1: Discovering store...")
		fmt.Printf("  Discovering store at %s...\n", *storeURL)
		wellKnownURL = *storeURL
	} else {
		fmt.Println("Step 1: Discovering stores...")
		fmt.Printf("  Querying registry at %s...\n", *registryURL)

		reg := client.NewRegistryClient(*registryURL)
		stores, err := reg.SearchStores("", "", "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error querying registry: %v\n", err)
			fmt.Fprintf(os.Stderr, "  Hint: Is the registry running? Check the URL.\n")
			os.Exit(1)
		}

		if len(stores.Data) == 0 {
			fmt.Fprintln(os.Stderr, "  No stores found in registry.")
			fmt.Fprintln(os.Stderr, "  Hint: Register a store first using the ace-server /registry/v1/stores endpoint.")
			os.Exit(1)
		}

		fmt.Printf("  Found %d store(s):\n", len(stores.Data))
		for _, s := range stores.Data {
			fmt.Printf("    - %q (%s) - %s\n", s.Name, s.ID, s.HealthStatus)
		}
		fmt.Println("  Selecting first store...")
		wellKnownURL = stores.Data[0].WellKnownURL
	}
	fmt.Println()

	// Step 2: Connect to store via well-known URL
	fmt.Println("Step 2: Connecting to store...")
	aceClient := client.NewACEClient("", *apiKey) // base URL set after discovery
	discovery, err := aceClient.Discover(wellKnownURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Error discovering store: %v\n", err)
		fmt.Fprintf(os.Stderr, "  Hint: Check that the well-known URL is correct and the store is running.\n")
		os.Exit(1)
	}

	fmt.Printf("  Store: %q\n", discovery.Name)
	fmt.Printf("  Version: %s\n", discovery.Version)
	fmt.Printf("  Capabilities: %s\n", strings.Join(discovery.Capabilities, ", "))
	fmt.Printf("  Currencies: %s\n", strings.Join(discovery.Currencies, ", "))
	fmt.Println()

	// Now create a properly configured ACE client with the discovered base URL
	aceClient = client.NewACEClient(discovery.ACEBaseURL, *apiKey)

	// Step 3: Browse catalog
	fmt.Println("Step 3: Browsing catalog...")
	products, err := aceClient.ListProducts("", 0, 20)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Error listing products: %v\n", err)
		os.Exit(1)
	}

	if len(products.Data) == 0 {
		fmt.Fprintln(os.Stderr, "  No products found in catalog.")
		os.Exit(1)
	}

	fmt.Printf("  Found %d product(s):\n", products.Total)
	for i, p := range products.Data {
		fmt.Printf("    %d. %s - %s (%s)\n", i+1, p.Name, formatMoney(p.Price), p.ID)
	}

	// Select up to 2 products
	selectedCount := 2
	if len(products.Data) < selectedCount {
		selectedCount = len(products.Data)
	}
	selected := products.Data[:selectedCount]
	fmt.Printf("  Selecting first %d product(s)...\n", selectedCount)
	fmt.Println()

	// Step 4: Create cart and add items
	fmt.Println("Step 4: Creating cart...")
	cart, err := aceClient.CreateCart()
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Error creating cart: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Cart created: %s\n", cart.ID)

	for i, p := range selected {
		qty := 1
		if i == 1 {
			qty = 2 // add 2 of the second product for variety
		}
		fmt.Printf("  Adding %q x%d...\n", p.Name, qty)

		cart, err = aceClient.AddCartItem(cart.ID, ace.AddCartItemRequest{
			ProductID: p.ID,
			Quantity:  qty,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error adding item to cart: %v\n", err)
			fmt.Fprintf(os.Stderr, "  Hint: The product may be out of stock.\n")
			os.Exit(1)
		}
	}

	fmt.Printf("  Cart total: %s\n", formatMoney(cart.Total))
	fmt.Println()

	// Step 5: Place order
	fmt.Println("Step 5: Placing order...")
	order, err := aceClient.CreateOrder(cart.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Error creating order: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("  Order created: %s\n", order.ID)
	fmt.Printf("  Status: %s\n", order.Status)
	fmt.Printf("  Items: %d item(s), total %s\n", len(order.Items), formatMoney(order.Total))
	fmt.Println()

	// Step 6: Initiate payment
	fmt.Println("Step 6: Initiating payment...")
	payment, err := aceClient.Pay(order.ID, *provider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Error initiating payment: %v\n", err)
		if strings.Contains(err.Error(), "auth") || strings.Contains(err.Error(), "401") {
			fmt.Fprintln(os.Stderr, "  Hint: Check your API key (--key flag).")
		}
		os.Exit(1)
	}

	fmt.Printf("  Payment initiated: %s\n", payment.ID)
	fmt.Printf("  Provider: %s\n", payment.Provider)
	if payment.PaymentURL != "" {
		fmt.Printf("  Payment URL: %s\n", payment.PaymentURL)
	}
	fmt.Println()

	// Step 7: Check payment status
	fmt.Println("Step 7: Checking payment status...")
	payStatus, err := aceClient.PaymentStatus(order.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Error checking payment status: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("  Payment status: %s\n", payStatus.Status)

	// Fetch updated order to show final status
	updatedOrder, err := aceClient.GetOrder(order.ID)
	if err != nil {
		fmt.Printf("  Order status: (could not refresh: %v)\n", err)
	} else {
		fmt.Printf("  Order status: %s\n", updatedOrder.Status)
	}
	fmt.Println()

	fmt.Println("=== Demo complete! Full purchase flow successful. ===")
}

// formatMoney converts a Money value to a human-readable string like "$79.99".
func formatMoney(m ace.Money) string {
	whole := m.Amount / 100
	cents := m.Amount % 100
	if cents < 0 {
		cents = -cents
	}

	symbol := m.Currency
	switch m.Currency {
	case "USD":
		symbol = "$"
	case "EUR":
		symbol = "\u20ac"
	case "ARS":
		symbol = "AR$"
	case "BRL":
		symbol = "R$"
	}

	return fmt.Sprintf("%s%d.%02d", symbol, whole, cents)
}
