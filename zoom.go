package main

import (
	"log"
	"time"

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
	log.Printf("updateZoomProducts\n")
}
