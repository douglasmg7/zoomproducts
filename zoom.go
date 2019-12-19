package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type productZunka struct {
	ObjectID         primitive.ObjectID `bson:"_id,omitempty" xml:"-"`
	ID               string             `xml:"CODIGO"`
	Name             string             `bson:"storeProductTitle" xml:"NOME"`
	Department       string             `bson:"" xml:"DEPARTAMENTO"`
	Category         string             `bson:"storeProductCategory" xml:"SUBDEPARTAMENTO"`
	Detail           string             `bson:"storeProductDetail" xml:"DESCRICAO"`
	TechInfo         string             `bson:"storeProductTechnicalInformation" xml:"-"` // To get get ean.
	PriceFloat64     float64            `bson:"storeProductPrice" xml:"-"`
	Price            string             `bson:"" xml:"PRECO"`
	PriceFrom        string             `bson:"" xml:"PRECO_DE"`
	InstallmentQtd   int                `bson:"" xml:"NPARCELA"`
	InstallmentValue string             `bson:"" xml:"VPARCELA"`
	Url              string             `bson:"" xml:"URL"`
	UrlImage         string             `bson:"" xml:"URL_IMAGEM"`
	MPC              string             `bson:"" xml:"MPC"`    // MPC – (Manufacturer Part Number)
	EAN              string             `bson:"ean" xml:"EAN"` // EAN – (European Article Number)
	SKU              string             `bson:"" xml:"SKU"`    // SKU – (Stock Keeping Unit)
	Images           []string           `bson:"images" xml:"-"`
}

type productZoom struct {
	ObjectID         primitive.ObjectID `bson:"_id,omitempty" xml:"-"`
	ID               string             `xml:"CODIGO"`
	Name             string             `bson:"storeProductTitle" xml:"NOME"`
	Department       string             `bson:"" xml:"DEPARTAMENTO"`
	Category         string             `bson:"storeProductCategory" xml:"SUBDEPARTAMENTO"`
	Detail           string             `bson:"storeProductDetail" xml:"DESCRICAO"`
	TechInfo         string             `bson:"storeProductTechnicalInformation" xml:"-"` // To get get ean.
	PriceFloat64     float64            `bson:"storeProductPrice" xml:"-"`
	Price            string             `bson:"" xml:"PRECO"`
	PriceFrom        string             `bson:"" xml:"PRECO_DE"`
	InstallmentQtd   int                `bson:"" xml:"NPARCELA"`
	InstallmentValue string             `bson:"" xml:"VPARCELA"`
	Url              string             `bson:"" xml:"URL"`
	UrlImage         string             `bson:"" xml:"URL_IMAGEM"`
	MPC              string             `bson:"" xml:"MPC"`    // MPC – (Manufacturer Part Number)
	EAN              string             `bson:"ean" xml:"EAN"` // EAN – (European Article Number)
	SKU              string             `bson:"" xml:"SKU"`    // SKU – (Stock Keeping Unit)
	Images           []string           `bson:"images" xml:"-"`
}

// Products to update.
var productListU = []string{}
var updateZoomProductsTimer = time.NewTimer(time.Second * 1)

// var updateZoomProductsTimer *time.Timer
var secondsToUpdateZoomProducts time.Duration = 3

// Zunka product updated.
func productHandlerPost(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	body, err := ioutil.ReadAll(req.Body)
	HandleError(w, err)

	// Get product id.
	productId := string(body)
	// log.Printf("productId: %s", string(productId))

	// Check if id is valid.
	validID := regexp.MustCompile(`(?i)^[a-f\d]{24}$`)
	if validID.MatchString(productId) {
		productListU = append(productListU, productId)
	}
	updateZoomProductsTimer.Stop()
	updateZoomProductsTimer = time.AfterFunc(time.Second*secondsToUpdateZoomProducts, updateZoomProducts)
	log.Printf("productList: %v", productListU)
}

// Update zoom producs.
func updateZoomProducts() {
	log.Printf("updateZoomProducts: %v", productListU)
	productListU = []string{}
}
