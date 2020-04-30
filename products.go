package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	ZOOM_TICKET_DEADLINE_MIN        = 60
	TIME_TO_CHECK_PRODUCTS_MIN_S    = 1
	TIME_TO_CHECK_PRODUCTS_MIN      = 5
	TIME_TO_CHECK_CONCISTENCY_MIN_S = 10
	TIME_TO_CHECK_CONCISTENCY_MIN   = 20
	TIME_TO_CHECK_TICKETS_MIN_S     = 2
	TIME_TO_CHECK_TICKETS_MIN       = 5
)

var muxUpdateZoomProducts sync.Mutex
var checkProductsTimer, checkConsistencyTimer, checkTicketsTimer *time.Timer

type productZunka struct {
	ObjectID      primitive.ObjectID `bson:"_id,omitempty"`
	Name          string             `bson:"storeProductTitle" xml:"NOME"`
	Category      string             `bson:"storeProductCategory"`
	Detail        string             `bson:"storeProductDetail"`
	TechInfo      string             `bson:"storeProductTechnicalInformation"` // To get get ean.
	Price         float64            `bson:"storeProductPrice"`
	EAN           string             `bson:"ean"` // EAN – (European Article Number)
	Length        int                `bson:"storeProductLength"`
	Height        int                `bson:"storeProductHeight"`
	Width         int                `bson:"storeProductWidth"`
	Weight        int                `bson:"storeProductWeight"`
	Quantity      int                `bson:"storeProductQtd"`
	Commercialize bool               `bson:"storeProductCommercialize"`
	MarketZoom    bool               `bson:"marketZoom"`
	Images        []string           `bson:"images"`
	UpdatedAt     time.Time          `bson:"updatedAt"`
	DeletedAt     time.Time          `bson:"deletedAt"`
}

type urlImageZoom struct {
	Main string `json:"main"`
	Url  string `json:"url"`
}

type productZoom struct {
	ID string `json:"id"`
	// SKU           string `json:"sku"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Department    string `json:"department"`
	SubDepartment string `json:"sub_department"`
	EAN           string `json:"ean"` // EAN – (European Article Number)
	// FreeShipping  string   `json:"free_shipping"`
	FreeShipping bool     `json:"free_shipping"`
	BasePrice    float64  `json:"base_price"` // Not used by marketplace
	Price        float64  `json:"price"`
	Installments struct { // Not used by marketplace
		AmountMonths int     `json:"amount_months"`
		Price        float64 `json:"price"` // Price by month
	} `json:"installments"`
	Quantity     int  `json:"quantity"`
	Availability bool `json:"availability"`
	// Availability string `json:"availability"`
	Dimensions struct {
		CrossDocking int    `json:"cross_docking"` // Days
		Height       string `json:"height"`        // M
		Length       string `json:"length"`        // M
		Width        string `json:"width"`         // M
		Weight       string `json:"weight"`        // KG
	} `json:"stock_info"`
	UrlImages    []urlImageZoom `json:"url_images"`
	Url          string         `json:"url"`
	MarketZoom   bool           `json:"-"`
	UpdatedAt    time.Time      `json:"-"`
	DeletedAt    time.Time      `json:"-"`
	NeverExisted bool           `json:"-"`
}

// Check if product received is equal.
func (p *productZoom) Equal(pr *productZoomR) bool {
	// ID.
	if pr.ID != p.ID {
		log.Printf("[debug] [product] Product ID %+v. ID not equal. Zunka: %+v, Zoom: %+v\n", p.ID, p.ID, pr.ID)
		return false
	}
	// Free shipping.
	if pr.FreeShipping != p.FreeShipping {
		log.Printf("[debug] [product] Product ID %+v. Free shipping not equal. Zunka: %+v, Zoom: %+v\n", p.ID, p.FreeShipping, pr.FreeShipping)
		return false
	}
	// Price.
	if pr.Price != p.Price {
		log.Printf("[debug] [product] Product ID %+v. Price not equal. Zunka: %+v, Zoom: %+v\n", p.ID, p.Price, pr.Price)
		return false
	}
	// Quantity.
	if pr.Quantity != p.Quantity {
		log.Printf("[debug] [product] Product ID %+v. Quantity not equal. Zunka: %+v, Zoom: %+v\n", p.ID, p.Quantity, pr.Quantity)
		return false
	}
	// Url.
	if pr.Url != p.Url {
		log.Printf("[debug] [product] Product ID %+v. Url not equal. Zunka: %+v, Zoom: %+v\n", p.ID, p.Url, pr.Url)
		return false
	}
	return true
}

// Specific for receive product from Zoom.
type productZoomR struct {
	ID string `json:"id"`
	// SKU           string `json:"sku"`
	Name string `json:"name"`
	// Description   string `json:"description"`
	// Department    string `json:"department"`
	// SubDepartment string `json:"sub_department"`
	// EAN           string `json:"ean"` // EAN – (European Article Number)
	FreeShipping bool     `json:"free_shipping"`
	BasePrice    float64  `json:"base_price"` // Not used by marketplace
	Price        float64  `json:"price"`
	Installments struct { // Not used by marketplace
		AmountMonths int     `json:"amount_months"`
		Price        float64 `json:"price"` // Price by month
	} `json:"installments"`
	Quantity int `json:"quantity"`
	// Availability string `json:"availability"`
	// Dimensions   struct {
	// CrossDocking int    `json:"cross_docking"` // Days
	// Height       string `json:"height"`        // M
	// Length       string `json:"length"`        // M
	// Width        string `json:"width"`         // M
	// Weight       string `json:"weight"`        // KG
	// } `json:"stock_info"`
	// UrlImages []urlImageZoom `json:"url_images"`
	Url    string `json:"url"`
	Active bool   `json:"active"`
}

// To use with chan.
type productZoomRAOk struct {
	Products *[]productZoomR
	Ok       bool
}
type productZoomAOk struct {
	Products *[]productZoom
	Ok       bool
}

// "url_image":"https://i.zst.com.br/thumbs/49/3e/28/993091954.jpg",
// "active":true},

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
	ProductID    string   `json:"product_id"`
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
	ProductsID []string
}
type zoomTicketResult struct {
	ProductID string `json:"product_id"`
	Status    int    `json:"status"`
	Message   string `json:"message"`
}

// Tickets to check.
var zoomTickets map[string]*zoomTicket

// Check consistency.
func checkConsistency() {
	muxUpdateZoomProducts.Lock()
	defer muxUpdateZoomProducts.Unlock()

	if len(zoomTickets) > 0 {
		log.Printf(":: Will not checking consistency, waiting to finish %d ticket(s).", len(zoomTickets))
		checkConsistencyTimer = time.AfterFunc(time.Minute*TIME_TO_CHECK_CONCISTENCY_MIN, checkConsistency)
		return
	}
	log.Println(":: Checking consistency...")

	cZoomR := make(chan productZoomRAOk)
	cZoomDb := make(chan productZoomAOk)

	go getZoomProducts(cZoomR)
	go getAllZunkaProducts(cZoomDb)

	prodZoomDBAOk, prodZoomRAOK := <-cZoomDb, <-cZoomR

	if prodZoomDBAOk.Ok && prodZoomRAOK.Ok {
		// log.Printf("\tZunka products count: %v", len(*prodZoomDBAOk.Products))
		// log.Printf("\tZoom Products count: %v", len(*prodZoomRAOK.Products))

		productsToUpdate := []productZoom{}
		productsToRemove := []productZoom{}

		// Check if zoom have all products.
		for _, prodDB := range *prodZoomDBAOk.Products {
			// Product deleted or not marked to market Zoom.
			if !prodDB.DeletedAt.IsZero() || !prodDB.MarketZoom {
				continue
			}
			// Product exist.
			productExistAndEqual := false
			for _, prodR := range *prodZoomRAOK.Products {
				if prodDB.ID == prodR.ID {
					if prodDB.Equal(&prodR) && prodR.Active {
						productExistAndEqual = true
					}
					break
				}
			}
			if !productExistAndEqual {
				productsToUpdate = append(productsToUpdate, prodDB)
			}
		}

		// Check if product was deleted  or not marked to Zoom market on zunka server.
		for _, prodR := range *prodZoomRAOK.Products {
			// Product considered as removed on zoom server when active is false.
			if !prodR.Active {
				continue
			}
			productFound := false
			for _, prodDB := range *prodZoomDBAOk.Products {
				// Product exist on db.
				if prodDB.ID == prodR.ID {
					// Product deleted or not marked to market Zoom.
					if !prodDB.DeletedAt.IsZero() || !prodDB.MarketZoom {
						productsToRemove = append(productsToRemove, productZoom{
							ID:         prodDB.ID,
							DeletedAt:  prodDB.DeletedAt,
							MarketZoom: prodDB.MarketZoom,
						})
					}
					productFound = true
					break
				}
			}
			// Product never existed on Zunka db.
			if !productFound {
				productsToRemove = append(productsToRemove, productZoom{
					ID:           prodR.ID,
					NeverExisted: true,
				})
			}
		}

		productsToUpdateList := []string{}
		for _, prod := range productsToUpdate {
			productsToUpdateList = append(productsToUpdateList, prod.ID)
		}
		log.Printf("\tProducts to update (%d): %s", len(productsToUpdate), strings.Join(productsToUpdateList, ", "))

		productsToRemoveList := []string{}
		for _, prod := range productsToRemove {
			productsToRemoveList = append(productsToRemoveList, prod.ID)
		}
		log.Printf("\tProducts to remove (%d): %s", len(productsToRemove), strings.Join(productsToRemoveList, ", "))

		// todo - Uncomment begin.
		// c := make(chan bool)

		// go updateZoomProducts(productsToUpdate, c)
		// go removeZoomProducts(productsToRemove, c)

		// // Newest updatedAt product time.
		// if <-c == true && <-c == true {
		// log.Println("\tZoom products consistency check finished.")
		// }
		// todo - Uncomment finish.

		// b, err := json.MarshalIndent(prodZoomRAOK.Products[8], "", "    ")
		// checkError(err)
		// // log.Println("Products all: ", products)
		// log.Println("Product: ", string(b))
	}
	checkConsistencyTimer = time.AfterFunc(time.Minute*TIME_TO_CHECK_CONCISTENCY_MIN, checkConsistency)
}

// Update zoom product.
func updateOneZoomProduct() {

	// zoomProducts :=  getChangedZunkaProducts()
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

// Try update prducts from failed ticket.
func retryFailedUpdateProducts(productsID []string) {
	muxUpdateZoomProducts.Lock()
	defer muxUpdateZoomProducts.Unlock()

	log.Printf(":: Retring failed update products...\n")
	// for _, id := range productsID {
	// log.Printf("	product ID: %v", id)
	// }

	// Get zoom products.
	zoomProdA := getZunkaProductsByID(productsID)
	// log.Printf("Changed zoom products: %+v", zoomProdA)

	c := make(chan bool)

	go updateZoomProducts(zoomProdA, c)
	go removeZoomProducts(zoomProdA, c)

	<-c
	<-c
}

// Update zoom producs.
func checkProducts() {
	muxUpdateZoomProducts.Lock()
	defer muxUpdateZoomProducts.Unlock()

	// Get zoom products changed.
	zoomProdA := getChangedZunkaProducts()
	// log.Printf("Changed zoom products: %+v", zoomProdA)

	if len(zoomProdA) > 0 {
		log.Println(":: Products changed...")
		c := make(chan bool)

		go updateZoomProducts(zoomProdA, c)
		go removeZoomProducts(zoomProdA, c)

		// Newest updatedAt product time.
		if <-c == true && <-c == true {
			updateNewestProductUpdatedAt()
		}
	} else {
		log.Println("No changed products.")
	}
	checkProductsTimer = time.AfterFunc(time.Minute*TIME_TO_CHECK_PRODUCTS_MIN, checkProducts)
}

// Update zoom products at zoom server.
func updateZoomProducts(prodA []productZoom, c chan bool) {
	var ticket zoomTicket

	p := struct {
		Products []productZoom `json:"products"`
	}{
		Products: []productZoom{},
	}
	for _, product := range prodA {
		// Update only not deleted products and marked to zoom market place.
		if product.DeletedAt.IsZero() && product.MarketZoom {
			log.Printf("\tProduct %v changed, UpdatedAt: %v\n", product.ID, product.UpdatedAt.In(brLocation))
			p.Products = append(p.Products, product)
			ticket.ProductsID = append(ticket.ProductsID, product.ID)
		}
	}
	// Nothing to do.
	if len(p.Products) == 0 {
		c <- true
		return
	}

	// // Log request.
	// zoomProductsJSONPretty, err := json.MarshalIndent(p, "", "    ")
	// log.Println("zoomProductsJSONPretty:", string(zoomProductsJSONPretty))

	zoomProductsJSON, err := json.Marshal(p)
	if checkError(err) {
		c <- false
		return
	}
	log.Println("zoomProductsJSON:", string(zoomProductsJSON))

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
	ticket.ReceivedAt = time.Now()
	log.Printf("\tTicket %v added (updated products)", ticket.ID)
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
		// Remove deleted products and not marked to zoom market place.
		if !product.DeletedAt.IsZero() || !product.MarketZoom || product.NeverExisted {
			if product.NeverExisted {
				// Never existed.
				log.Printf("\tProduct %v removed, Never existed on Zunka db.", product.ID)
			} else if !product.DeletedAt.IsZero() {
				// Deleted.
				log.Printf("\tProduct %v removed, DeletedAt: %v\n", product.ID, product.DeletedAt.In(brLocation))
			} else if !product.MarketZoom {
				// Unmarked to zoom market.
				log.Printf("\tProduct %v removed, unmarked to zoom market place, UpdatedAt: %v\n", product.ID, product.UpdatedAt.In(brLocation))
			}
			productIDA = append(productIDA, productID{ID: product.ID})
			ticket.ProductsID = append(ticket.ProductsID, product.ID)
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
	ticket.ReceivedAt = time.Now()
	log.Printf("\tTicket %v added (removed products)", ticket.ID)
	c <- true
}

/******************************************************************************
* TICKET
******************************************************************************/
// Check if tickets finished.
func checkTickets() {
	muxUpdateZoomProducts.Lock()
	defer muxUpdateZoomProducts.Unlock()

	// log.Println(":: Checking tickets...")

	ticketsIDToRemove := []string{}
	// Range of tikcets.
	for k, v := range zoomTickets {
		// Give up get ticket result and check zoom products consistency.
		elapsedTimeInSeconds := time.Since(v.ReceivedAt).Seconds()
		if elapsedTimeInSeconds > ZOOM_TICKET_DEADLINE_MIN*60 {
			// Set ticket to be deleted and retry update products.
			ticketsIDToRemove = append(ticketsIDToRemove, k)
			log.Printf("Give up ticket %v, TickCount: %d, Elapsed time: %.1f s\n", v.ID, v.TickCount, elapsedTimeInSeconds)
			// go retryFailedUpdateProducts(v.ProductsID)
			continue
		}
		// Checkt ticket.
		v.TickCount = v.TickCount + 1
		log.Printf(":: Checking ticket %v, TickCount: %d, Elapsed time: %.1f s\n", v.ID, v.TickCount, elapsedTimeInSeconds)
		receipt, err := getZoomReceipt(k)
		if err != nil {
			log.Println(fmt.Sprintf("\tError getting zoom ticket. %v\n.", err))
			continue
		}
		// Finished.
		if receipt.Finished {
			notSuccessfulProductsId := []string{}
			// log.Printf("Ticket zoom finished. ID: %v, Receipt: %v\n", v.ID, receipt)
			log.Printf("\tTicket %v finished\n", v.ID)
			for _, result := range receipt.Results {
				log.Printf("\tProductID: %s, Status: %d, Message: %s, WarnMessages: %s\n", result.ProductID, result.Status, result.Message, result.WarnMessages)
				// Product update failed.
				// 404, trying to delete nonexistent product.
				if result.Status != 200 && result.Status != 201 && result.Status != 404 {
					notSuccessfulProductsId = append(notSuccessfulProductsId, result.ProductID)
				}
			}
			// Retry not successful updated products.
			if len(notSuccessfulProductsId) > 0 {
				// go retryFailedUpdateProducts(notSuccessfulProductsId)
			}
			ticketsIDToRemove = append(ticketsIDToRemove, k)
		}
	}
	// Remove completed or failed tickets.
	for _, ticketId := range ticketsIDToRemove {
		delete(zoomTickets, ticketId)
		// log.Printf("\tTicket %v removed\n", ticketId)
	}
	checkTicketsTimer = time.AfterFunc(time.Minute*TIME_TO_CHECK_TICKETS_MIN, checkTickets)
}

/******************************************************************************
* ZOOM PRODUCTS AND RECEIPTS
******************************************************************************/
// Get products from zoom webservice.
func getZoomProducts(c chan productZoomRAOk) {

	type PaginationType struct {
		CurrentPage     int `json:"current_page"`
		ProductsPerPage int `json:"products_per_page"`
		TotalProducts   int `json:"total_products"`
	}

	pageProducts := struct {
		Pagination PaginationType `json:"pagination"`
		Products   []productZoomR `json:"products"`
	}{
		Pagination: PaginationType{},
		Products:   []productZoomR{},
	}

	result := productZoomRAOk{
		Ok:       false,
		Products: &pageProducts.Products,
	}

	// Request products.
	client := &http.Client{}
	// req, err := http.NewRequest("GET", "http://merchant.zoom.com.br/api/merchant/products", nil)
	// req, err := http.NewRequest("GET", "https://staging-merchant.zoom.com.br/api/merchant/products", nil)
	req, err := http.NewRequest("GET", zoomHost()+"/products", nil)
	if checkError(err) {
		c <- result
		return
	}
	req.Header.Set("Content-Type", "application/json")

	req.SetBasicAuth(zoomUser(), zoomPass())
	res, err := client.Do(req)
	if checkError(err) {
		c <- result
		return
	}

	// Result.
	defer res.Body.Close()
	resBody, err := ioutil.ReadAll(res.Body)
	if checkError(err) {
		c <- result
		return
	}
	// log.Printf("Page and products: %s", string(resBody))

	// No 200 status.
	if res.StatusCode != 200 {
		checkError(errors.New(fmt.Sprintf("Error getting products from zoom server.\n\nstatus: %v\n\nbody: %v", res.StatusCode, string(resBody))))
		c <- result
		return
	}

	// Log body pageResult.
	// log.Printf("body: %s", string(resBody))

	err = json.Unmarshal(resBody, &pageProducts)
	if checkError(err) {
		c <- result
		return
	}

	if pageProducts.Pagination.TotalProducts >= 500 {
		log.Println("[alert] Zoom products per page more than 500!")
	}

	// log.Printf("pageResult: %v", pageResult)

	// Log active products.
	productsActiveCount := 0
	productsActiveList := []string{}
	for _, product := range *result.Products {
		if product.Active {
			productsActiveList = append(productsActiveList, product.ID)
			productsActiveCount++
		}
	}
	log.Printf("\tActive Zoom products  (%d): %s", productsActiveCount, strings.Join(productsActiveList, ", "))

	// Log not active products.
	productsNotActiveCount := 0
	productsNotActiveList := []string{}
	for _, product := range *result.Products {
		if !product.Active {
			productsNotActiveList = append(productsNotActiveList, product.ID)
			productsNotActiveCount++
		}
	}
	// log.Printf("\tNot active zoom products (%d): %s", productsNotActiveCount, strings.Join(productsNotActiveList, ", "))
	log.Printf("\tNot active zoom products (%d)", productsNotActiveCount)

	result.Ok = true
	c <- result
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

	// // Log receipt.
	// log.Printf("zoom receipt: %+v\n", receipt)

	return receipt, nil
}

/******************************************************************************
* ZUNKA PRODUCTS
******************************************************************************/
// Get Zunka products changed.
func getChangedZunkaProducts() (products []productZoom) {
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
		// Because if product was changed to that state, it must be deleted on zoom server. So not use this anymore.
		// {"storeProductPrice", bson.D{
		// {"$gt", 0},
		// }},
		// {"storeProductTitle", bson.D{
		// {"$regex", `\S`},
		// }},
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
		{"storeProductCommercialize", true},
		{"storeProductPrice", true},
		{"storeProductQtd", true},
		{"ean", true},
		{"images", true},
		{"marketZoom", true},
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
		err := cur.Decode(&prodZunka)
		checkFatalError(err)

		prodZoom := *convertProductZunkaToZoom(&prodZunka)
		products = append(products, prodZoom)
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
	return products
}

// Get all Zunka products.
func getAllZunkaProducts(c chan productZoomAOk) {
	result := productZoomAOk{
		Ok:       false,
		Products: &[]productZoom{},
	}
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
		// {"storeProductPrice", bson.D{
		// {"$gt", 0},
		// }},
		// {"storeProductTitle", bson.D{
		// {"$regex", `\S`},
		// }},
		// {"updatedAt", bson.D{
		// {"$gt", newestProductUpdatedAt},
		// }},
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
		{"storeProductCommercialize", true},
		{"storeProductPrice", true},
		{"storeProductQtd", true},
		{"ean", true},
		{"marketZoom", true},
		{"images", true},
		{"updatedAt", true},
		{"deletedAt", true},
	})
	// todo - comment.
	// findOptions.SetLimit(12)
	cur, err := collection.Find(ctxFind, filter, findOptions)
	if checkError(err) {
		c <- result
		return
	}

	defer cur.Close(ctxFind)
	for cur.Next(ctxFind) {
		prodZunka := productZunka{}
		err := cur.Decode(&prodZunka)
		if checkError(err) {
			c <- result
			return
		}

		prodZoom := convertProductZunkaToZoom(&prodZunka)
		// log.Printf("prodZoom.ID: %+v\n", prodZoom.ID)
		*result.Products = append(*result.Products, *prodZoom)
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
	// log.Printf("Products count: %v\n", len(*result.Products))
	validProductsCount := 0
	validProductsList := []string{}
	for _, product := range *result.Products {
		if product.Availability && product.MarketZoom {
			validProductsList = append(validProductsList, product.ID)
			validProductsCount++
		}
	}
	log.Printf("\tActive Zunka products (%d): %s", validProductsCount, strings.Join(validProductsList, ", "))

	result.Ok = true
	c <- result
}

// Get Zunka products changed.
func getZunkaProductsByID(productsID []string) (products []productZoom) {
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

	objectIDs := []primitive.ObjectID{}
	for _, id := range productsID {
		objId, err := primitive.ObjectIDFromHex(id)
		checkFatalError(err)
		objectIDs = append(objectIDs, objId)
	}

	// log.Printf("objectIDs: %+v", objectIDs)

	filter := bson.D{
		// {"storeProductPrice", bson.D{{"$gt", 0}}},
		// {"storeProductTitle", bson.D{{"$regex", `\S`}}},
		{"_id", bson.D{{"$in", objectIDs}}},
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
		{"storeProductCommercialize", true},
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
		err := cur.Decode(&prodZunka)
		checkFatalError(err)

		prodZoom := *convertProductZunkaToZoom(&prodZunka)
		products = append(products, prodZoom)
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
	return products
}

// Convert Zunka product to Zoom product.
func convertProductZunkaToZoom(prodZunka *productZunka) (prodZoom *productZoom) {
	prodZoom = &productZoom{}
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
	prodZoom.FreeShipping = false
	// prodZoom.FreeShipping = "false"
	// EAN.
	if prodZunka.EAN == "" {
		prodZunka.EAN = findEan(prodZunka.TechInfo)
	}
	prodZoom.EAN = prodZunka.EAN
	// Price from.
	// prodZoom.Price = fmt.Sprintf("%.2f", prodZunka.Price)
	prodZoom.Price = prodZunka.Price
	// prodZoom.Price = strings.ReplaceAll(prodZoom.Price, ".", ",")
	prodZoom.BasePrice = prodZoom.Price
	// Installments.
	prodZoom.Installments.AmountMonths = 3
	// prodZoom.Installments.Price = fmt.Sprintf("%.2f", float64(int((prodZunka.Price/3)*100))/100)
	prodZoom.Installments.Price = prodZunka.Price
	prodZoom.Quantity = prodZunka.Quantity
	if prodZunka.Commercialize && (prodZunka.Quantity > 0) && (prodZunka.Price > 0) && (prodZunka.Name != "") {
		prodZoom.Availability = true
	}
	if prodZunka.Commercialize && prodZunka.MarketZoom {
		prodZoom.MarketZoom = true
	}
	// prodZoom.Availability = strconv.FormatBool(prodZunka.Active)
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
	return prodZoom
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
