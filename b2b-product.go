package main

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var port string

var client *mongo.Client
var err error
var logPath, xmlZoomPath string

// Development mode.
var dev bool
var initTime time.Time

func init() {
	// Service port.
	port = "8082"

	// Path for log.
	zunkaPathdata := os.Getenv("ZUNKAPATH")
	if zunkaPathdata == "" {
		panic("ZUNKAPATH not defined.")
	}
	logPath := path.Join(zunkaPathdata, "log", "b2b-product")
	// Path for xml.
	zunkaPathDist := os.Getenv("ZUNKA_SITE_PATH")
	if zunkaPathDist == "" {
		panic("ZUNK_SITE_APATH not defined.")
	}
	xmlZoomPath = path.Join(zunkaPathDist, "dist/xml/zoom")
	// Create path.
	os.MkdirAll(logPath, os.ModePerm)
	os.MkdirAll(xmlZoomPath, os.ModePerm)

	// Log file.
	logFile, err := os.OpenFile(path.Join(logPath, "b2b-product.log"), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}

	// Log configuration.
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
	log.SetFlags(log.Ldate | log.Lmicroseconds)

	// Run mode.
	mode := "production"
	if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "dev") {
		dev = true
		mode = "development"
	}

	// Log start.
	log.Printf("*** Starting b2b-product in %v mode (version %s) ***\n", mode, version)
}

func main() {
	// MongoDB config.
	client, err = mongo.NewClient(options.Client().ApplyURI(zunkaSiteMongoDBConnectionString))
	// MongoDB client.
	ctxClient, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctxClient)
	checkFatalError(err)

	// Ping mongoDB.
	ctxPing, _ := context.WithTimeout(context.Background(), 2*time.Second)
	err = client.Ping(ctxPing, readpref.Primary())
	checkFatalError(err)

	// Init router.
	router := httprouter.New()
	router.GET("/b2b-product", checkZoomAuthorization(indexHandler))

	// Zoom API.
	router.POST("/b2b-product/product", checkZoomAuthorization(productHandlerPost))

	// // Disconnect from mongoDB.
	// err = client.Disconnect(ctxClient)
	// checkFatalError(err)
	log.Println("listen port", port)
	log.Fatal(http.ListenAndServe(":"+port, newLogger(router)))
}

/**************************************************************************************************
* Authorization middleware
**************************************************************************************************/

// Authorization.
func checkZoomAuthorization(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
		user, pass, ok := req.BasicAuth()
		if ok && user == zoomUser() && pass == zoomPass() {
			h(w, req, p)
			return
		}
		log.Printf("try  , %v %v, user: %v, pass: %v, ok: %v", req.Method, req.URL.Path, user, pass, ok)
		log.Printf("want , %v %v, user: %v, pass: %v", req.Method, req.URL.Path, zoomUser(), zoomPass())
		// Unauthorised.
		w.Header().Set("WWW-Authenticate", `Basic realm="Please enter your username and password for this service"`)
		w.WriteHeader(401)
		w.Write([]byte("Unauthorised\n"))
		return
	}
}

// Authorization.
func checkZunkaSiteAuthorization(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
		user, pass, ok := req.BasicAuth()
		if ok && user == zunkaSiteUser() && pass == zunkaSitePass() {
			h(w, req, p)
			return
		}
		log.Printf("Unauthorized access, %v %v, user: %v, pass: %v, ok: %v", req.Method, req.URL.Path, user, pass, ok)
		log.Printf("authorization      , %v %v, user: %v, pass: %v", req.Method, req.URL.Path, zunkaSiteUser(), zunkaSitePass())
		// Unauthorised.
		w.Header().Set("WWW-Authenticate", `Basic realm="Please enter your username and password for this service"`)
		w.WriteHeader(401)
		w.Write([]byte("Unauthorised.\n"))
		return
	}
}

/**************************************************************************************************
* Logger middleware
**************************************************************************************************/

// Logger struct.
type logger struct {
	handler http.Handler
}

// Handle interface.
// todo - why DELETE is logging twice?
func (l *logger) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// log.Printf("%s %s - begin", req.Method, req.URL.Path)
	start := time.Now()
	l.handler.ServeHTTP(w, req)
	log.Printf("%s %s %v", req.Method, req.URL.Path, time.Since(start))
	// log.Printf("header: %v", req.Header)
}

// New logger.
func newLogger(h http.Handler) *logger {
	return &logger{handler: h}
}

func apiGetProducts() {
	// Request products.
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://merchant.zoom.com.br/api/merchant/products", nil)
	req.Header.Set("Content-Type", "application/json")
	checkFatalError(err)

	// Devlopment.
	req.SetBasicAuth("zoomteste_zunka", "H2VA79Ug4fjFsJb")
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
		log.Fatalf("Error ao solicitar a criação do produtos no servidor zoom.\n\nstatus: %v\n\nbody: %v", res.StatusCode, string(resBody))
		return
	}
	// Log body result.
	log.Printf("body: %s", string(resBody))
}

func checkFatalError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
