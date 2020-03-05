package main

import (
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// Handler error.
func HandleError(w http.ResponseWriter, err error) {
	if err != nil {
		// http.Error(w, "Some thing wrong", 404)
		if production {
			http.Error(w, "Alguma coisa deu errado", http.StatusInternalServerError)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		log.Println(err.Error())
		return
	}
}

// Index handler.
func indexHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	w.WriteHeader(200)
	w.Write([]byte("Hello!\n"))
}
