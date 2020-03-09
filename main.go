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
var logPath string

// Production mode.
var production bool

// Init time.
var initTime time.Time

// Newest updated time product processed.
var newestProductUpdatedAt time.Time

// Newest updated time product processed temp.
var newestProductUpdatedAtTemp time.Time

// Newest updated time product processed db param name.
const LAST_PRODUCT_UPDATED_TIME = "ZOOMPRODUCTS-last-product-updated-time"

// Brazil time location.
var brLocation *time.Location

func init() {
	// Check if production mode.
	if os.Getenv("RUN_MODE") == "production" {
		production = true
	}

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
	logPath := path.Join(zunkaPathdata, "log", "zoom")
	// Path for xml.
	zunkaPathDist := os.Getenv("ZUNKA_SITE_PATH")
	if zunkaPathDist == "" {
		panic("ZUNK_SITE_APATH not defined.")
	}
	// Create path.
	os.MkdirAll(logPath, os.ModePerm)

	// Log file.
	logFile, err := os.OpenFile(path.Join(logPath, "zoomproducts.log"), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}

	// Log configuration.
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
	// log.SetFlags(log.LstdFlags)
	// log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetFlags(log.Ldate | log.Lmicroseconds)
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
	// Log start.
	runMode := "development"
	if production {
		runMode = "production"
	}
	log.Printf("Running in %v mode (version %s)\n", runMode, version)

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
	zoomTickets = map[string]*zoomTicket{}
	// checkConsistency()
	checkConsistencyTimer = time.AfterFunc(time.Minute*TIME_TO_CHECK_CONCISTENCY_MIN_S, checkConsistency)
	checkTicketsTimer = time.AfterFunc(time.Minute*TIME_TO_CHECK_TICKETS_MIN_S, checkTickets)
	checkProductsTimer = time.AfterFunc(time.Minute*TIME_TO_CHECK_PRODUCTS_MIN_S, checkProducts)

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
	go shutdown(server, serverStopRequest, serverStopFinish)

	log.Printf("listen address: %s", address[1:])
	// log.Fatal(http.ListenAndServe(address, newLogger(router)))
	if err = server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error: Could not listen on %s. %v\n", address, err)
	}
	<-serverStopFinish
	log.Println("Server stopped")
}

func shutdown(server *http.Server, serverStopRequest <-chan os.Signal, serverStopFinish chan<- bool) {
	<-serverStopRequest
	log.Println("Server is shutting down...")
	// Stop timers.
	if checkProductsTimer != nil {
		checkProductsTimer.Stop()
	}
	if checkConsistencyTimer != nil {
		checkConsistencyTimer.Stop()
	}
	if checkTicketsTimer != nil {
		checkTicketsTimer.Stop()
	}

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
	filter := bson.D{
		{"name", LAST_PRODUCT_UPDATED_TIME},
	}
	var result struct {
		Value time.Time `bson:"value"`
	}
	// cur, err := collection.FindOne(ctxFind, filter, findOptions).Decode(&result)
	err := collection.FindOne(ctxFind, filter).Decode(&result)
	if err == mongo.ErrNoDocuments {
		log.Printf("No %s into db.", LAST_PRODUCT_UPDATED_TIME)
		newestProductUpdatedAt = time.Time{}
	} else if err != nil {
		log.Fatalf("[Error] Could not get %s from db. %v\n", LAST_PRODUCT_UPDATED_TIME, err)
	}
	log.Printf("%s: %v", LAST_PRODUCT_UPDATED_TIME, result.Value.Local())
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
	filter := bson.M{"name": LAST_PRODUCT_UPDATED_TIME}
	update := bson.M{
		"$set": bson.M{"value": newestProductUpdatedAt},
	}
	_, err := collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		log.Fatalf("Could not save %s into db.", LAST_PRODUCT_UPDATED_TIME)
	}
	log.Printf("Saved %s into db: %v", LAST_PRODUCT_UPDATED_TIME, newestProductUpdatedAt.In(brLocation))
}
