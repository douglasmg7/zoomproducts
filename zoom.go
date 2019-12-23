package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type productZunka struct {
	ObjectID primitive.ObjectID `bson:"_id,omitempty"`
	Name     string             `bson:"storeProductTitle" xml:"NOME"`
	Category string             `bson:"storeProductCategory"`
	Detail   string             `bson:"storeProductDetail"`
	TechInfo string             `bson:"storeProductTechnicalInformation"` // To get get ean.
	Price    float64            `bson:"storeProductPrice"`
	EAN      string             `bson:"ean"` // EAN – (European Article Number)
	Lenght   int                `bson:"storeProductLength"`
	Height   int                `bson:"storeProductHeight"`
	Width    int                `bson:"storeProductWidth"`
	Weight   int                `bson:"storeProductWeight"`
	Quantity int                `bson:"storeProductQtd"`
	Active   bool               `bson:"storeProductActive"`
	Images   []string           `bson:"images"`
}
type urlImageZoom struct {
	Main bool   `json:"main"`
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
	FreeShipping  bool   `json:"free_shipping"`
	BasePrice     string `json:"base_price"`
	Price         string `json:"price"`
	Installments  struct {
		AmountMonths int    `json:"amount_months"`
		Price        string `json:"price"` // Price by month
	} `json:"installments"`
	Quantity     int  `json:"quantity"`
	Availability bool `json:"availability"`
	Dimensions   struct {
		CrossDocking int    `json:"cross_docking"`
		Height       string `json:"height"`
		Lenght       string `json:"lenght"`
		Weight       string `json:"weight"`
		Width        string `json:"width"`
	} `json:"stock_info"`
	UrlImages []urlImageZoom `json:"url_images"`
	Url       string         `json:"url"`
}

// Zoom tieck.
var zoomTicker *time.Ticker

// Start zoom product update.
func startZoomProductUpdate() {
	zoomTicker = time.NewTicker(time.Second * 3)
	for {
		select {
		case <-zoomTicker.C:
			updateZoomProducts()
		}
	}
}

// Stop zoom product update.
func stopZoomProductUpdate() {
	zoomTicker.Stop()
	log.Println("Zoom update products stopped")
}

// Update zoom producs.
func updateZoomProducts() {
	zoomProducts := getZoomProdutctsToUpdate()
	// jsonZoomProducts, err := json.Marshal(zoomProducts)
	jsonZoomProducts, err := json.MarshalIndent(zoomProducts, "", "    ")
	if err != nil {
		log.Fatalf("Error creating json zoom products. %v", err)
	}
	log.Println("zoom-products:", string(jsonZoomProducts))
}

// Get zoom products to update.
func getZoomProdutctsToUpdate() (results []productZoom) {
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
	})
	// todo - comment.
	findOptions.SetLimit(12)
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
		prodZoom.Dimensions.CrossDocking = 0 // ?
		prodZoom.Dimensions.Lenght = strconv.Itoa(prodZunka.Lenght)
		prodZoom.Dimensions.Height = strconv.Itoa(prodZunka.Height)
		prodZoom.Dimensions.Width = strconv.Itoa(prodZunka.Width)
		prodZoom.Dimensions.Weight = strconv.Itoa(prodZunka.Weight)
		// Free shipping.
		prodZoom.FreeShipping = false
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
		prodZoom.Availability = prodZunka.Active
		prodZoom.Url = "https://www.zunka.com.br/product/" + prodZoom.ID
		// Images.
		log.Println("len(prodZoom.UrlImages):", len(prodZoom.UrlImages))
		// prodZoom.UrlImages = []productZoom.urlImages{}
		for index, urlImage := range prodZunka.Images {
			if index == 0 {
				prodZoom.UrlImages = append(prodZoom.UrlImages, urlImageZoom{true, "https://www.zunka.com.br/img/" + prodZoom.ID + "/" + urlImage})
			} else {
				prodZoom.UrlImages = append(prodZoom.UrlImages, urlImageZoom{false, "https://www.zunka.com.br/img/" + prodZoom.ID + "/" + urlImage})
			}
		}
		// prodZunka.UrlImage = "https://www.zunka.com.br/img/" + prodZunka.ID + "/" + prodZunka.Images[0]
		results = append(results, prodZoom)
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
	return results
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
