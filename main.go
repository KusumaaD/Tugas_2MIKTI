package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// OrderProcessor interface untuk pemrosesan pesanan
type OrderProcessor interface {
	Process(order *Order) error
	ValidateOrder(order *Order) error
}

// DataValidator interface untuk validasi data
type DataValidator interface {
	Validate() error
}

// MenuItem merepresentasikan item dalam menu
type MenuItem struct {
	Name     string
	Price    float64
	Quantity int
}

// Order merepresentasikan pesanan
type Order struct {
	Items     []*MenuItem
	Total     float64
	Payment   float64
	Change    float64
	Encrypted string
}

// RestaurantOrderProcessor implementasi dari OrderProcessor
type RestaurantOrderProcessor struct {
	wg       sync.WaitGroup
	orders   chan *Order
	results  chan *Order
	timeout  time.Duration
}

// menuList menyimpan daftar menu (unexported)
var menuList = map[string]float64{
	"nasi goreng": 25000,
	"ayam bakar":  30000,
}

// validateInput menggunakan regexp untuk validasi input
func validateInput(input interface{}) error {
	switch v := input.(type) {
	case string:
		if matched, _ := regexp.MatchString(`^[a-zA-Z\s]+$`, v); !matched {
			return fmt.Errorf("input hanya boleh berisi huruf dan spasi")
		}
	case float64:
		if matched, _ := regexp.MatchString(`^\d+(\.\d{2})?$`, fmt.Sprintf("%.2f", v)); !matched {
			return fmt.Errorf("format angka tidak valid")
		}
	default:
		return fmt.Errorf("tipe data tidak didukung")
	}
	return nil
}

func NewOrder() *Order {
	return &Order{
		Items: make([]*MenuItem, 0),
	}
}

// AddItem menambahkan item ke pesanan menggunakan pointer
func (o *Order) AddItem(name string, price float64, quantity int) {
	item := &MenuItem{
		Name:     name,
		Price:    price,
		Quantity: quantity,
	}
	o.Items = append(o.Items, item)
	o.calculateTotal()
}

// calculateTotal menghitung total pesanan (unexported method)
func (o *Order) calculateTotal() {
	o.Total = 0
	for _, item := range o.Items {
		o.Total += item.Price * float64(item.Quantity)
	}
}

// NewRestaurantOrderProcessor membuat processor baru
func NewRestaurantOrderProcessor() *RestaurantOrderProcessor {
	return &RestaurantOrderProcessor{
		orders:   make(chan *Order, 10), // buffered channel
		results:  make(chan *Order, 10),
		timeout:  5 * time.Second,
	}
}

func (p *RestaurantOrderProcessor) ProcessOrder(order *Order) {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		
		select {
		case <-time.After(p.timeout):
			fmt.Println("Timeout processing order")
		case p.orders <- order:
			// Enkripsi dan proses pesanan
			orderDetails := fmt.Sprintf("Total: %.2f, Payment: %.2f, Change: %.2f", 
				order.Total, order.Payment, order.Change)
			order.Encrypted = base64.StdEncoding.EncodeToString([]byte(orderDetails))
			p.results <- order
		}
	}()
}

func main() {
	// Defer untuk memastikan pesan "Program selesai" selalu dicetak
	defer func() {
		fmt.Println("\nMenggunakan bantuan di gnulinux lab...")
		fmt.Println("Program selesai")
	}()

	// Recover dari panic
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Error: %v\n", r)
		}
	}()

	processor := NewRestaurantOrderProcessor()
	reader := bufio.NewReader(os.Stdin)
	order := NewOrder()

	for {
		fmt.Println("\nMenu:")
		for name, price := range menuList {
			fmt.Printf("- %s: Rp%.2f\n", strings.Title(name), price)
		}
		fmt.Printf("\nMasukkan nama item [ketik 'selesai' untuk menyelesaikan]\n")

		fmt.Print("Pilihan: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "selesai" {
			break
		}

		// Validasi input menggunakan interface kosong dan type assertion
		if err := validateInput(interface{}(input)); err != nil {
			panic(err)
		}

		price, exists := menuList[input]
		if !exists {
			panic(fmt.Sprintf("Menu '%s' tidak tersedia", input))
		}

		fmt.Print("Masukkan jumlah: ")
		qtyStr, _ := reader.ReadString('\n')
		qtyStr = strings.TrimSpace(qtyStr)
		qty, err := strconv.Atoi(qtyStr)
		if err != nil {
			panic("Jumlah tidak valid")
		}

		order.AddItem(strings.Title(input), price, qty)
	}

	// Menampilkan pesanan
	fmt.Println("\nPesanan Anda:")
	for _, item := range order.Items {
		fmt.Printf("- %s (x%d)\n", item.Name, item.Quantity)
	}
	fmt.Printf("Total Harga: Rp%.2f\n", order.Total)

	// Memproses pembayaran
	fmt.Print("\nMasukkan jumlah uang: ")
	paymentStr, _ := reader.ReadString('\n')
	paymentStr = strings.TrimSpace(paymentStr)
	payment, err := strconv.ParseFloat(paymentStr, 64)
	if err != nil {
		panic("Jumlah pembayaran tidak valid")
	}

	if payment < order.Total {
		panic("Pembayaran kurang!")
	}

	order.Payment = payment
	order.Change = payment - order.Total

	// Proses pesanan menggunakan goroutine
	processor.ProcessOrder(order)

	// Tunggu semua goroutine selesai
	processor.wg.Wait()
	close(processor.orders)
	close(processor.results)

	// Ambil hasil proses
	processedOrder := <-processor.results

	// Menampilkan hasil akhir
	fmt.Printf("\nUang yang dibayar: Rp%.2f\n", processedOrder.Payment)
	fmt.Printf("Kembalian: Rp%.2f\n", processedOrder.Change)
	fmt.Printf("Pesanan (encoded format): %s\n", processedOrder.Encrypted)
}