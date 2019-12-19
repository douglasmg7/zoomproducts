package main

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Just to create correct xml structure.
type _products struct {
	XMLName xml.Name `xml:"PRODUTOS"`
	A       []product
}

type product struct {
	XMLName          xml.Name           `xml:"PRODUTO"`
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

func createZoomXml() {
	initTime := time.Now()
	// Get all products to commercialize.
	products := getProdutcts()
	log.Printf("Zoom XML: %v products to be commercialized.", len(products))

	// log.Println(products[0])
	saveXML(products)

	// Time to create XML file.
	timeToProcess := time.Since(initTime)
	log.Printf("Zoom XML: time to create xml file %.0fms\n\n", float64(timeToProcess)/1e6)
}

// Get all products to commercialize.
func getProdutcts() (results []product) {
	collection := client.Database("zunka").Collection("products")

	ctxFind, _ := context.WithTimeout(context.Background(), 3*time.Second)
	// D: A BSON document. This type should be used in situations where order matters, such as MongoDB commands.
	// M: An unordered map. It is the same as D, except it does not preserve order.
	// A: A BSON array.
	// E: A single element inside a D.
	// options.Find().SetProjection(bson.D{{"storeProductTitle", true}, {"_id", false}}),
	// {'storeProductCommercialize': true, 'storeProductTitle': {$regex: /\S/}, 'storeProductQtd': {$gt: 0}, 'storeProductPrice': {$gt: 0}};
	filter := bson.D{
		{"storeProductCommercialize", true},
		{"storeProductQtd", bson.D{
			{"$gt", 0},
		}},
		{"storeProductPrice", bson.D{
			{"$gt", 0},
		}},
		{"storeProductTitle", bson.D{
			{"$regex", `\S`},
		}},
	}
	findOptions := options.Find()
	findOptions.SetProjection(bson.D{
		{"storeProductTitle", true},
		{"storeProductCategory", true},
		{"storeProductDetail", true},
		{"storeProductTechnicalInformation", true},
		{"storeProductPrice", true},
		{"ean", true},
		{"images", true},
		{"dealerName", true},
	})
	// findOptions.SetLimit(1)
	cur, err := collection.Find(ctxFind, filter, findOptions)
	checkFatalError(err)

	defer cur.Close(ctxFind)
	for cur.Next(ctxFind) {
		// var result bson.M
		result := product{
			Department:     "Informática",
			InstallmentQtd: 3,
		}
		err := cur.Decode(&result)
		checkFatalError(err)
		// Mounted fields.
		// ID.
		result.ID = result.ObjectID.Hex()
		// EAN.
		if result.EAN == "" {
			result.EAN = findEan(result.TechInfo)
		}
		// Price from.
		result.Price = fmt.Sprintf("%.2f", result.PriceFloat64)
		result.Price = strings.ReplaceAll(result.Price, ".", ",")
		result.PriceFrom = result.Price
		// Installments.
		result.InstallmentValue = fmt.Sprintf("%.2f", float64(int((result.PriceFloat64/3)*100))/100)
		result.InstallmentValue = strings.ReplaceAll(result.InstallmentValue, ".", ",")
		result.Url = "https://www.zunka.com.br/product/" + result.ID
		// Images.
		if len(result.Images) > 0 {
			result.UrlImage = "https://www.zunka.com.br/img/" + result.ID + "/" + result.Images[0]
		} else {
			result.UrlImage = ""
		}
		// log.Println(result)
		// log.Println("EAN:", findEan(result.TechInfo))
		// log.Println("TechInfo:", result.TechInfo)
		// log.Println("ObjectID:", result.ObjectID)
		// log.Println("ObjectID (string):", result.ObjectID.Hex())
		// log.Println("ID:", result.ID)
		// log.Println("Name:", result.Name)
		// log.Println("Detail:", result.Detail)
		// log.Println("Category:", result.Category)
		// log.Println("Price:", result.Price)
		// log.Println("Images:", result.Images)
		results = append(results, result)
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
	return results
}

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

func saveXML(products []product) {
	updateXMLFile := true
	fileNameSent := "zoom-produtos.xml"
	fileNameNew := "zoom-produtos-" + time.Now().Format("2006-nov-02-150405") + ".xml"

	prods := _products{
		A: products,
	}
	// Create xml.
	xmlFile, _ := xml.MarshalIndent(prods, "", "    ")
	// Add xml header.
	xmlFile = []byte(xml.Header + string(xmlFile))
	// Save with current time name.
	err := ioutil.WriteFile(path.Join(xmlZoomPath, fileNameNew), xmlFile, 0644)
	checkFatalError(err)

	// Check if new file is different from last sent.
	xmlFileSent, err := ioutil.ReadFile(path.Join(xmlZoomPath, fileNameSent))
	if err != nil {
		log.Println("XML file not exist.")
	} else {
		if bytes.Equal(xmlFile, xmlFileSent) {
			updateXMLFile = false
			log.Println("XML not changed.")
		}
	}
	// Save xml file.
	log.Printf("Saving XML file %v ...", path.Join(xmlZoomPath, fileNameNew))
	// Send xml file to zoom webservice.
	if updateXMLFile {
		// Save xml as last modified.
		log.Printf("Saving XML file %v ...", path.Join(xmlZoomPath, fileNameSent))
		err = ioutil.WriteFile(path.Join(xmlZoomPath, fileNameSent), xmlFile, 0644)
		checkFatalError(err)
	}
}
