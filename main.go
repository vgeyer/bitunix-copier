package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/tradingiq/bitunix-client/bitunix"
	"github.com/tradingiq/bitunix-client/model"
)

type Config struct {
	SourceAPI    string
	SourceSecret string
	DestAPI      string
	DestSecret   string
}

type OrderCopier struct {
	sourceClient *bitunix.ApiClient
	destClient   *bitunix.ApiClient
	config       *Config
}

type OrderHandler struct {
	copier *OrderCopier
}

func (h *OrderHandler) SubscribeOrder(order *model.OrderChannelMessage) {
	fmt.Printf("Received order: %+v\n", order)
	
	if order.Status == "NEW" || order.Status == "PARTIALLY_FILLED" {
		err := h.copier.copyOrder(order)
		if err != nil {
			log.Printf("Failed to copy order: %v", err)
		}
	}
}

func NewOrderCopier(config *Config) (*OrderCopier, error) {
	sourceClient := bitunix.NewApiClient(config.SourceAPI, config.SourceSecret)
	destClient := bitunix.NewApiClient(config.DestAPI, config.DestSecret)
	
	return &OrderCopier{
		sourceClient: sourceClient,
		destClient:   destClient,
		config:       config,
	}, nil
}

func (oc *OrderCopier) copyOrder(sourceOrder *model.OrderChannelMessage) error {
	fmt.Printf("Copying order: Symbol=%s, Side=%s, Quantity=%f, Price=%f\n", 
		sourceOrder.Symbol, sourceOrder.Side, sourceOrder.Quantity, sourceOrder.Price)
	
	orderBuilder := bitunix.NewOrderBuilder(
		model.ParseSymbol(sourceOrder.Symbol),
		model.TradeSide(sourceOrder.Side),
		model.SideOpen,
		sourceOrder.Quantity,
	).WithOrderType(model.OrderType(sourceOrder.Type)).WithPrice(sourceOrder.Price)
	
	order, err := orderBuilder.Build()
	if err != nil {
		return fmt.Errorf("failed to build order: %v", err)
	}
	
	ctx := context.Background()
	result, err := oc.destClient.PlaceOrder(ctx, &order)
	if err != nil {
		return fmt.Errorf("failed to place order on destination account: %v", err)
	}
	
	fmt.Printf("Successfully copied order to destination account: %+v\n", result)
	return nil
}

func (oc *OrderCopier) startMonitoring(ctx context.Context) error {
	ws, err := bitunix.NewPrivateWebsocket(ctx, oc.config.SourceAPI, oc.config.SourceSecret)
	if err != nil {
		return fmt.Errorf("failed to create websocket: %v", err)
	}
	defer ws.Disconnect()
	
	if err := ws.Connect(); err != nil {
		return fmt.Errorf("failed to connect websocket: %v", err)
	}
	
	orderHandler := &OrderHandler{copier: oc}
	
	err = ws.SubscribeOrders(orderHandler)
	if err != nil {
		return fmt.Errorf("failed to subscribe to orders: %v", err)
	}
	
	fmt.Println("Started monitoring orders on source account...")
	
	return ws.Stream()
}

func loadConfig() *Config {
	return &Config{
		SourceAPI:    getEnvOrDefault("SOURCE_API_KEY", ""),
		SourceSecret: getEnvOrDefault("SOURCE_SECRET_KEY", ""),
		DestAPI:      getEnvOrDefault("DEST_API_KEY", ""),
		DestSecret:   getEnvOrDefault("DEST_SECRET_KEY", ""),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	fmt.Println("Starting Bitunix Order Copier...")
	
	// Load environment variables from .env file if it exists
	err := godotenv.Load()
	if err != nil {
		fmt.Println("No .env file found, using system environment variables")
	}
	
	config := loadConfig()
	
	if config.SourceAPI == "" || config.SourceSecret == "" || config.DestAPI == "" || config.DestSecret == "" {
		log.Fatal("Please set environment variables: SOURCE_API_KEY, SOURCE_SECRET_KEY, DEST_API_KEY, DEST_SECRET_KEY")
	}
	
	copier, err := NewOrderCopier(config)
	if err != nil {
		log.Fatalf("Failed to create order copier: %v", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		fmt.Println("\nShutting down...")
		cancel()
	}()
	
	err = copier.startMonitoring(ctx)
	if err != nil {
		log.Fatalf("Failed to start monitoring: %v", err)
	}
	
	fmt.Println("Order copier stopped")
}