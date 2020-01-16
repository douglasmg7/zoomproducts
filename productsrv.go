package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var address string

var client *mongo.Client
var err error
var logPath, xmlZoomPath string

// Development mode.
var dev bool
var initTime time.Time

// Newest updated time product processed.
var newestProductUpdatedAt time.Time

// Newest updated time product processed temp.
var newestProductUpdatedAtTemp time.Time

// Newest updated time product processed db param name.
const NEWEST_PRODUCT_UPDATED_AT = "productsrv-newest-product-updated-at"

// Brazil time location.
var brLocation *time.Location

func init() {
	// Brazil location.
	brLocation, err = time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		panic(err)
	}

	// Listern address.
	address = ":8082"

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
	logFile, err := os.OpenFile(path.Join(logPath, "productsrv.log"), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}

	// Log configuration.
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
	// log.SetFlags(log.LstdFlags)
	// log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetFlags(log.Ldate | log.Lmicroseconds)

	// Run mode.
	mode := "production"
	if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "dev") {
		dev = true
		mode = "development"
	}

	// Log start.
	log.Printf("** Starting productsrv in %v mode (version %s) **\n", mode, version)
}

func checkError(err error) bool {
	if err != nil {
		// notice that we're using 1, so it will actually log where
		// the error happened, 0 = this function, we don't want that.
		function, file, line, _ := runtime.Caller(1)
		log.Printf("[error] [%s] [%s:%d] %v", filepath.Base(file), runtime.FuncForPC(function).Name(), line, err)
		return true
	}
	return false
}

func main() {

	checkZoomProductsConsistency()
	log.Println("End")
	return

	// MongoDB config.
	client, err = mongo.NewClient(options.Client().ApplyURI(zunkaSiteMongoDBConnectionString))

	// MongoDB client.
	ctxClient, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctxClient)
	if err != nil {
		log.Fatalf("Error. Could not connect to mongodb. %v\n", err)
	}

	// Ping mongoDB.
	ctxPing, _ := context.WithTimeout(context.Background(), 2*time.Second)
	err = client.Ping(ctxPing, readpref.Primary())
	if err != nil {
		log.Fatalf("Error. Could not ping mongodb. %v\n", err)
	}

	// Init router.
	router := httprouter.New()
	router.GET("/productsrv", checkZoomAuthorization(indexHandler))

	getNewestProductUpdatedAt()
	go startZoomProductUpdate()

	// Create server.
	server := &http.Server{
		Addr:    address,
		Handler: router,
		// ErrorLog:     logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	// Gracegull shutdown.
	serverStopFinish := make(chan bool, 1)
	serverStopRequest := make(chan os.Signal, 1)
	signal.Notify(serverStopRequest, os.Interrupt)
	go gracefullShutdown(server, serverStopRequest, serverStopFinish)

	log.Println("listen address", address)
	// log.Fatal(http.ListenAndServe(address, newLogger(router)))
	if err = server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error: Could not listen on %s. %v\n", address, err)
	}
	<-serverStopFinish
	log.Println("Server stopped")
}

func gracefullShutdown(server *http.Server, serverStopRequest <-chan os.Signal, serverStopFinish chan<- bool) {
	<-serverStopRequest
	log.Println("Server is shutting down...")

	stopZoomProductUpdate()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	server.SetKeepAlivesEnabled(false)
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Could not gracefully shutdown the server: %v\n", err)
	}
	close(serverStopFinish)
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
		// log.Printf("try  , %v %v, user: %v, pass: %v, ok: %v", req.Method, req.URL.Path, user, pass, ok)
		// log.Printf("want , %v %v, user: %v, pass: %v", req.Method, req.URL.Path, zoomUser(), zoomPass())
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

func checkFatalError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

/**************************************************************************************************
* Last time products was retrived from db.
**************************************************************************************************/

// Get newest product updated at from db.
func getNewestProductUpdatedAt() {
	collection := client.Database("zunka").Collection("params")

	ctxFind, _ := context.WithTimeout(context.Background(), 3*time.Second)
	// D: A BSON document. This type should be used in situations where order matters, such as MongoDB commands.
	// M: An unordered map. It is the same as D, except it does not preserve order.
	// A: A BSON array.
	// E: A single element inside a D.
	// options.Find().SetProjection(bson.D{{"storeProductTitle", true}, {"_id", false}}),
	// {'storeProductCommercialize': true, 'storeProductTitle': {$regex: /\S/}, 'storeProductQtd': {$gt: 0}, 'storeProductPrice': {$gt: 0}};
	filter := bson.D{
		{"name", NEWEST_PRODUCT_UPDATED_AT},
		// {"storeProductQtd", bson.D{
		// {"$gt", 0},
		// }},
	}
	// findOptions := options.Find()
	// findOptions.SetProjection(bson.D{
	// {"_id", false},
	// {"value", true},
	// })
	var result struct {
		Value time.Time `bson:"value"`
	}
	// cur, err := collection.FindOne(ctxFind, filter, findOptions).Decode(&result)
	err := collection.FindOne(ctxFind, filter).Decode(&result)
	if err == mongo.ErrNoDocuments {
		log.Printf("Not exist param %s into db.", NEWEST_PRODUCT_UPDATED_AT)
		newestProductUpdatedAt = time.Time{}
	} else if err != nil {
		log.Fatalf("Error. Could not retrive param %s from db. %v\n", NEWEST_PRODUCT_UPDATED_AT, err)
	}
	log.Printf("Retrive %s from db: %v", NEWEST_PRODUCT_UPDATED_AT, result.Value.Local())
	newestProductUpdatedAt = result.Value
}

// Save newest product updated at into db.
func updateNewestProductUpdatedAt() {
	if newestProductUpdatedAt.Equal(newestProductUpdatedAtTemp) {
		return
	}
	newestProductUpdatedAt = newestProductUpdatedAtTemp
	collection := client.Database("zunka").Collection("params")
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	filter := bson.M{"name": NEWEST_PRODUCT_UPDATED_AT}
	update := bson.M{
		"$set": bson.M{"value": newestProductUpdatedAt},
	}
	_, err := collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		log.Fatalf("Could not save %s into db.", NEWEST_PRODUCT_UPDATED_AT)
	}
	log.Printf("Saved %s into db: %v", NEWEST_PRODUCT_UPDATED_AT, newestProductUpdatedAt.In(brLocation))
}
