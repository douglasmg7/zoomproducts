package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	ZOOM_TICK_INTERVAL           = 3
	ZOOM_TICKET_DEADLINE_SECONDS = 10
)

type productZunka struct {
	ObjectID  primitive.ObjectID `bson:"_id,omitempty"`
	Name      string             `bson:"storeProductTitle" xml:"NOME"`
	Category  string             `bson:"storeProductCategory"`
	Detail    string             `bson:"storeProductDetail"`
	TechInfo  string             `bson:"storeProductTechnicalInformation"` // To get get ean.
	Price     float64            `bson:"storeProductPrice"`
	EAN       string             `bson:"ean"` // EAN – (European Article Number)
	Length    int                `bson:"storeProductLength"`
	Height    int                `bson:"storeProductHeight"`
	Width     int                `bson:"storeProductWidth"`
	Weight    int                `bson:"storeProductWeight"`
	Quantity  int                `bson:"storeProductQtd"`
	Active    bool               `bson:"storeProductActive"`
	Images    []string           `bson:"images"`
	UpdatedAt time.Time          `bson:"updatedAt"`
	DeletedAt time.Time          `bson:"deletedAt"`
}

type urlImageZoom struct {
	Main string `json:"main"`
	Url  string `json:"url"`
}

type productZoom struct {
	ID string `json:"id"`
	// SKU           string `json:"sku"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Department    string   `json:"department"`
	SubDepartment string   `json:"sub_department"`
	EAN           string   `json:"ean"` // EAN – (European Article Number)
	FreeShipping  string   `json:"free_shipping"`
	BasePrice     string   `json:"base_price"` // Not used by marketplace
	Price         string   `json:"price"`
	Installments  struct { // Not used by marketplace
		AmountMonths int    `json:"amount_months"`
		Price        string `json:"price"` // Price by month
	} `json:"installments"`
	Quantity     int    `json:"quantity"`
	Availability string `json:"availability"`
	Dimensions   struct {
		CrossDocking int    `json:"cross_docking"` // Days
		Height       string `json:"height"`        // M
		Length       string `json:"length"`        // M
		Width        string `json:"width"`         // M
		Weight       string `json:"weight"`        // KG
	} `json:"stock_info"`
	UrlImages []urlImageZoom `json:"url_images"`
	Url       string         `json:"url"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt time.Time      `json:"-"`
}

// To get zoom receipt information using the ticket number.
type zoomReceipt struct {
	// ticket           string    `json:"ticket"`
	// scrollId         string    `json:"scrollId"`
	Finished         bool                `json:"finished"`
	Quantity         int                 `json:"quantity"`
	RequestTimestamp zoomTime            `json:"requestTimestamp"`
	Results          []zoomReceiptResult `json:"results"`
}
type zoomReceiptResult struct {
	ProductId    string   `json:"product_id"`
	Status       int      `json:"status"`
	Message      string   `json:"message"`
	WarnMessages []string `json:"warning_messages"`
}

// To unmarshal diferent data format.
type zoomTime struct {
	time.Time
}

func (t *zoomTime) UnmarshalJSON(j []byte) error {
	s := string(j)
	s = strings.Trim(s, `"`)
	newTime, err := time.Parse("2006-01-02T15:04:05", s)
	if err != nil {
		return err
	}
	t.Time = newTime
	return nil
}

// To get result of product insertion or edit.
type zoomTicket struct {
	ID         string             `json:"ticket"`
	Results    []zoomTicketResult `json:"results"`
	ReceivedAt time.Time
	TickCount  int // Number of ticks before get finish from zoom server.
}
type zoomTicketResult struct {
	ProductId string `json:"product_id"`
	Status    int    `json:"status"`
	Message   string `json:"message"`
}

// Tickets to check.
var zoomTickets map[string]*zoomTicket

// Zoom tickets.
var zoomTicker *time.Ticker

// Ticket deadline ticks.
var zoomTicketDeadlineTicks = 100

func init() {
	zoomTicketDeadlineTicks = int(math.Ceil(float64(ZOOM_TICKET_DEADLINE_SECONDS) / ZOOM_TICK_INTERVAL))
	// log.Println("ZOOM_TICKET_DEADLINE_TICKS:", zoomTicketDeadlineTicks)
}

// Start zoom product update.
func startZoomProductUpdate() {
	zoomTickets = map[string]*zoomTicket{}
	zoomTicker = time.NewTicker(time.Second * ZOOM_TICK_INTERVAL)
	for {
		select {
		case <-zoomTicker.C:
			log.Println(":: Tick")
			// updateOneZoomProduct()
			updateManyZoomProducts()
		}
	}
}

// Stop zoom product update.
func stopZoomProductUpdate() {
	zoomTicker.Stop()
	log.Println("Zoom update products stopped")
}

// Update zoom product.
func updateOneZoomProduct() {

	// zoomProducts :=  getZoomProductsChanged()
	// if len(zoomProducts) == 0 {
	// return
	// }

	// // Just for conference.
	// zoomProductJSONPretty, err := json.MarshalIndent(zoomProducts[0], "", "    ")
	// log.Println("zoomProductJSONPretty:", string(zoomProductJSONPretty))

	// zoomProductJSON, err := json.Marshal(zoomProducts[0])
	// // _, err := json.MarshalIndent(zoomProducts, "", "    ")
	// if err != nil {
	// log.Fatalf("Error creating json zoom products. %v", err)
	// }

	// // Request products.
	// client := &http.Client{}
	// req, err := http.NewRequest("POST", zoomHost()+"/product", bytes.NewBuffer(zoomProductJSON))
	// req.Header.Set("Content-Type", "application/json")
	// checkFatalError(err)

	// req.SetBasicAuth(zoomUser(), zoomPass())
	// res, err := client.Do(req)
	// checkFatalError(err)

	// defer res.Body.Close()
	// checkFatalError(err)

	// // Result.
	// resBody, err := ioutil.ReadAll(res.Body)
	// checkFatalError(err)
	// // No 200 status.
	// if res.StatusCode != 200 {
	// log.Fatalf("Error ao solicitar a criação do produto no servidor zoom, status: %v, body: %v", res.StatusCode, string(resBody))
	// return
	// }
	// // Log body result.
	// log.Printf("body: %s", string(resBody))

	// // Newest updatedAt product time.
	// updateNewestProductUpdatedAt()
}

// Update zoom producs.
func updateManyZoomProducts() {
	// Check if tickets fineshed.
	checkZoomTicketsFinish()

	// Get zoom products changed.
	zoomProd := getZoomProductsChanged()

	c := make(chan bool)

	go updateZoomProducts(zoomProd, c)
	go removeZoomProducts(zoomProd, c)

	// Newest updatedAt product time.
	if <-c == true && <-c == true {
		updateNewestProductUpdatedAt()
	}
}

// Update zoom products at zoom server.
func updateZoomProducts(prodA []productZoom, c chan bool) {
	var ticket zoomTicket

	p := struct {
		Products []productZoom `json:"products"`
	}{
		Products: []productZoom{},
	}
	// Select not deleted products.
	for _, product := range prodA {
		if product.DeletedAt.IsZero() {
			log.Printf("Product changed. ID: %v, UpdatedAt: %v\n", product.ID, product.UpdatedAt.In(brLocation))
			p.Products = append(p.Products, product)
		}
	}
	// Nothing to do.
	if len(p.Products) == 0 {
		c <- true
		return
	}

	// Log request.
	// zoomProductsJSONPretty, err := json.MarshalIndent(p, "", "    ")
	// log.Println("zoomProductsJSONPretty:", string(zoomProductsJSONPretty))

	zoomProductsJSON, err := json.Marshal(p)
	if checkError(err) {
		c <- false
		return
	}

	// Request products.
	client := &http.Client{}
	req, err := http.NewRequest("POST", zoomHost()+"/products", bytes.NewBuffer(zoomProductsJSON))
	req.Header.Set("Content-Type", "application/json")
	if checkError(err) {
		c <- false
		return
	}

	req.SetBasicAuth(zoomUser(), zoomPass())
	res, err := client.Do(req)
	if checkError(err) {
		c <- false
		return
	}
	defer res.Body.Close()

	// Result.
	resBody, err := ioutil.ReadAll(res.Body)
	if checkError(err) {
		c <- false
		return
	}

	// No success.
	if res.StatusCode != 200 && res.StatusCode != 201 {
		err = errors.New(fmt.Sprintf("Not received status 200 neither 201. status: %v, body: %v", res.StatusCode, string(resBody)))
		_ = checkError(err)
		c <- false
		return
	}
	// Log body result.
	// log.Printf("body: %s", string(resBody))

	// Get ticket.
	err = json.Unmarshal(resBody, &ticket)
	if checkError(err) {
		c <- false
		return
	}

	zoomTickets[ticket.ID] = &ticket
	log.Printf("Ticket zoom added (update). ID: %v", ticket.ID)
	c <- true
}

// Remove zoom products at zoom server.
func removeZoomProducts(prodA []productZoom, c chan bool) {
	var ticket zoomTicket

	// Product id.
	type productID struct {
		ID string `json:"id"`
	}

	productIDA := []productID{}
	for _, product := range prodA {
		if !product.DeletedAt.IsZero() {
			log.Printf("Product removed. ID: %v, DeletedAt: %v\n", product.ID, product.DeletedAt.In(brLocation))
			productIDA = append(productIDA, productID{ID: product.ID})
		}
	}
	// Nothing to do.
	if len(productIDA) == 0 {
		c <- true
		return
	}

	// zoomProductsJSONPretty, err := json.MarshalIndent(zoomProdUpdateA, "", "    ")
	p := struct {
		Products []productID `json:"products"`
	}{
		Products: productIDA,
	}

	// Log request.
	// zoomProductsJSONPretty, err := json.MarshalIndent(p, "", "    ")
	// log.Println("zoomProductsJSONPretty:", string(zoomProductsJSONPretty))

	zoomProductsJSON, err := json.Marshal(p)
	if checkError(err) {
		c <- false
		return
	}

	// Request products.
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", zoomHost()+"/products", bytes.NewBuffer(zoomProductsJSON))
	req.Header.Set("Content-Type", "application/json")
	if checkError(err) {
		c <- false
		return
	}

	req.SetBasicAuth(zoomUser(), zoomPass())
	res, err := client.Do(req)
	if checkError(err) {
		c <- false
		return
	}
	defer res.Body.Close()

	// Result.
	resBody, err := ioutil.ReadAll(res.Body)
	if checkError(err) {
		c <- false
		return
	}

	// No success.
	if res.StatusCode != 200 && res.StatusCode != 201 {
		err = errors.New(fmt.Sprintf("Not received status 200 neither 201. status: %v, body: %v", res.StatusCode, string(resBody)))
		_ = checkError(err)
		c <- false
		return
	}
	// Log body result.
	// log.Printf("body: %s", string(resBody))

	// Get ticket.
	err = json.Unmarshal(resBody, &ticket)
	if checkError(err) {
		c <- false
		return
	}

	zoomTickets[ticket.ID] = &ticket
	log.Printf("Ticket zoom added (remove). ID: %v", ticket.ID)
	c <- true
}

// Check if tickets finished.
func checkZoomTicketsFinish() {
	for k, v := range zoomTickets {
		if v.TickCount > zoomTicketDeadlineTicks {
			delete(zoomTickets, k)
			log.Println("v.ReceivedAt:", v.ReceivedAt)
			log.Println("now:", time.Now())
			log.Println("since", time.Since(v.ReceivedAt))
			log.Printf("Removed zoom ticket ID %v (no answer from zoom server ater %d seconds).\n", v.ID, time.Since(v.ReceivedAt).Seconds)
			continue
		}
		v.TickCount = v.TickCount + 1
		log.Printf("Checking zoom ticket. ID: %v, TickCount: %d \n", v.ID, v.TickCount)
		receipt, err := getZoomReceipt(k)
		if err != nil {
			log.Println(fmt.Sprintf("Error getting zoom ticket. %v\n.", err))
			continue
		}
		// Finished.
		if receipt.Finished {
			log.Printf("Ticket zoom finished. ID: %v, Receipt: %v\n", v.ID, receipt)
			// for result := range receipt.Results {
			// log.Printf("Ticket finished: %s, Result:%v\n", k, result)
			// }
			delete(zoomTickets, k)
		}
	}
}

func getZoomProducts() {
	// Request products.
	client := &http.Client{}
	// req, err := http.NewRequest("GET", "http://merchant.zoom.com.br/api/merchant/products", nil)
	// req, err := http.NewRequest("GET", "https://staging-merchant.zoom.com.br/api/merchant/products", nil)
	req, err := http.NewRequest("GET", zoomHost()+"/products", nil)
	req.Header.Set("Content-Type", "application/json")
	checkFatalError(err)

	// Devlopment.
	// req.SetBasicAuth("zoomteste_zunka", "H2VA79Ug4fjFsJb")
	req.SetBasicAuth(zoomUser(), zoomPass())
	// Production.
	// req.SetBasicAuth("zunka_informatica*", "h8VbfoRoMOSgZ2B")
	res, err := client.Do(req)
	checkFatalError(err)

	defer res.Body.Close()
	checkFatalError(err)

	// Result.
	resBody, err := ioutil.ReadAll(res.Body)
	checkFatalError(err)
	// No 200 status.
	if res.StatusCode != 200 {
		log.Fatalf("Error getting products from zoom server.\n\nstatus: %v\n\nbody: %v", res.StatusCode, string(resBody))
		return
	}
	// Log body result.
	log.Printf("body: %s", string(resBody))
}

// Get receipt information.
func getZoomReceipt(ticketId string) (receipt zoomReceipt, err error) {
	// Request products.
	client := &http.Client{}
	// log.Println("host:", zoomHost()+"/receipt/"+ticketId)
	req, err := http.NewRequest("GET", zoomHost()+"/receipt/"+ticketId, nil)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return receipt, errors.New(fmt.Sprintf("Error creating ticket request.  %v\n", err))
	}

	req.SetBasicAuth(zoomUser(), zoomPass())
	res, err := client.Do(req)
	if err != nil {
		return receipt, errors.New(fmt.Sprintf("Error creating zoom receipt request. %v\n", err))
	}
	defer res.Body.Close()

	// Result.
	resBody, err := ioutil.ReadAll(res.Body)
	if res.StatusCode != 200 && res.StatusCode != 201 {
		return receipt, errors.New(fmt.Sprintf("Error reading ticket from zoom server.  %v\n", err))
	}

	// No 200 status.
	if res.StatusCode != 200 && res.StatusCode != 201 {
		return receipt, errors.New(fmt.Sprintf("Error getting ticket from zoom server. status: %d\n, body: %s", res.StatusCode, string(resBody)))
	}
	// Log body result.
	// log.Printf("Ticket body: %s", string(resBody))

	err = json.Unmarshal(resBody, &receipt)
	if err != nil {
		return receipt, errors.New(fmt.Sprintf("Error unmarshal zoom receipt. %v", err))
	}

	// Log receipt.
	// log.Printf("zoom receipt: %+v\n", receipt)

	return receipt, nil
}

// Get zoom products changed.
func getZoomProductsChanged() (products []productZoom) {
	// To save the new one when finish.
	newestProductUpdatedAtTemp = newestProductUpdatedAt

	collection := client.Database("zunka").Collection("products")

	ctxFind, _ := context.WithTimeout(context.Background(), 3*time.Second)
	// D: A BSON document. This type should be used in situations where order matters, such as MongoDB commands.
	// M: An unordered map. It is the same as D, except it does not preserve order.
	// A: A BSON array.
	// E: A single element inside a D.
	// options.Find().SetProjection(bson.D{{"storeProductTitle", true}, {"_id", false}}),
	// {'storeProductCommercialize': true, 'storeProductTitle': {$regex: /\S/}, 'storeProductQtd': {$gt: 0}, 'storeProductPrice': {$gt: 0}};
	filter := bson.D{
		// {"storeProductCommercialize", true},
		// {"storeProductQtd", bson.D{
		// {"$gt", 0},
		// }},
		{"storeProductPrice", bson.D{
			{"$gt", 0},
		}},
		{"storeProductTitle", bson.D{
			{"$regex", `\S`},
		}},
		{"updatedAt", bson.D{
			{"$gt", newestProductUpdatedAt},
		}},
	}
	findOptions := options.Find()
	findOptions.SetProjection(bson.D{
		{"_id", true},
		{"storeProductTitle", true},
		{"storeProductCategory", true},
		{"storeProductDetail", true},
		{"storeProductTechnicalInformation", true}, // To get EAN if not have EAN.
		{"storeProductLength", true},
		{"storeProductHeight", true},
		{"storeProductWidth", true},
		{"storeProductWeight", true},
		{"storeProductActive", true},
		{"storeProductPrice", true},
		{"storeProductQtd", true},
		{"ean", true},
		{"images", true},
		{"updatedAt", true},
		{"deletedAt", true},
	})
	// todo - comment.
	// findOptions.SetLimit(12)
	cur, err := collection.Find(ctxFind, filter, findOptions)
	checkFatalError(err)

	defer cur.Close(ctxFind)
	for cur.Next(ctxFind) {
		prodZunka := productZunka{}
		prodZoom := productZoom{}
		err := cur.Decode(&prodZunka)
		checkFatalError(err)
		// Mounted fields.
		// ID.
		prodZoom.ID = prodZunka.ObjectID.Hex()
		// Name.
		prodZoom.Name = prodZunka.Name
		// Description.
		prodZoom.Description = prodZunka.Detail
		// Department.
		prodZoom.Department = "Informática"
		// Sub department.
		prodZoom.SubDepartment = prodZunka.Category
		// Dimensions.
		prodZoom.Dimensions.CrossDocking = 2 // ?
		prodZoom.Dimensions.Length = fmt.Sprintf("%.3f", float64(prodZunka.Length)/100)
		prodZoom.Dimensions.Height = fmt.Sprintf("%.3f", float64(prodZunka.Height)/100)
		prodZoom.Dimensions.Width = fmt.Sprintf("%.3f", float64(prodZunka.Width)/100)
		prodZoom.Dimensions.Weight = strconv.Itoa(prodZunka.Weight)
		// Free shipping.
		prodZoom.FreeShipping = "false"
		// EAN.
		if prodZunka.EAN == "" {
			prodZunka.EAN = findEan(prodZunka.TechInfo)
		}
		prodZoom.EAN = prodZunka.EAN
		// Price from.
		prodZoom.Price = fmt.Sprintf("%.2f", prodZunka.Price)
		// prodZoom.Price = strings.ReplaceAll(prodZoom.Price, ".", ",")
		prodZoom.BasePrice = prodZoom.Price
		// Installments.
		prodZoom.Installments.AmountMonths = 3
		prodZoom.Installments.Price = fmt.Sprintf("%.2f", float64(int((prodZunka.Price/3)*100))/100)
		prodZoom.Quantity = prodZunka.Quantity
		prodZoom.Availability = strconv.FormatBool(prodZunka.Active)
		prodZoom.Url = "https://www.zunka.com.br/product/" + prodZoom.ID
		// Images.
		for index, urlImage := range prodZunka.Images {
			if index == 0 {
				prodZoom.UrlImages = append(prodZoom.UrlImages, urlImageZoom{"true", "https://www.zunka.com.br/img/" + prodZoom.ID + "/" + urlImage})
			} else {
				prodZoom.UrlImages = append(prodZoom.UrlImages, urlImageZoom{"false", "https://www.zunka.com.br/img/" + prodZoom.ID + "/" + urlImage})
			}
		}
		prodZoom.UpdatedAt = prodZunka.UpdatedAt
		prodZoom.DeletedAt = prodZunka.DeletedAt
		// Set newest updated time.
		if prodZunka.UpdatedAt.After(newestProductUpdatedAtTemp) {
			// log.Println("time updated")
			newestProductUpdatedAtTemp = prodZunka.UpdatedAt
		}
		products = append(products, prodZoom)
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
	return products
}

// Find EAN from string.
func findEan(s string) string {
	lines := strings.Split(s, "\n")
	// (?i) case-insensitive flag.
	r := regexp.MustCompile(`(?i).*ean.*`)
	for _, line := range lines {
		if r.MatchString(line) {
			return strings.TrimSpace(strings.Split(line, ";")[1])
		}
	}
	return ""
}
