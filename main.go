package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/gorilla/mux"
)

func main() {
	if err := OpenEnv(); err != nil {
		panic(err)
	}

	if err := OpenDBConnection(); err != nil {
		panic(err)
	}

	r := mux.NewRouter()
	logger := log.New(os.Stdout, "", log.LstdFlags)

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		SendJson(w, http.StatusOK, Payload{"message": "Welcome to the API!"})
	}).Methods("GET")

	r.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		SendJson(w, http.StatusOK, Payload{"message": "pong"})
	}).Methods("GET")

	// Buckets
	r.HandleFunc("/bucket", HandleBucketCreation).Methods(http.MethodPost)
	r.HandleFunc("/bucket/{name}", HandleBucketDeletion).Methods(http.MethodDelete)
	r.HandleFunc("/bucket/{name}/objects", HandleObjectsFetch).Methods(http.MethodGet)

	// Objects
	r.HandleFunc("/object", HandleObjectCreation).Methods(http.MethodPost)
	r.HandleFunc("/object/{uuid}/external", HandleGeneratingSharableLink).Methods(http.MethodPost)
	r.HandleFunc("/object/{uuid}", HandleObjectDeletion).Methods(http.MethodDelete)
	r.HandleFunc("/object/{uuid}", HandleObjectFetch).Methods(http.MethodGet)

	// Object share
	r.HandleFunc("/share/{bucket}/{uuid}", HandleServingRequestedObject).Methods(http.MethodGet)

	// middlewares
	r.Use(func(n http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			n.ServeHTTP(w, r)
		})
	})
	r.Use(NewLogMiddleware(logger).Func())

	router := func() http.Handler {
		rps := 1.0
		l := tollbooth.NewLimiter(rps, &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour})
		return tollbooth.LimitHandler(l, r)
	}()

	addr := GetAddr()
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	SetTLSConfigs(srv.TLSConfig)

	log.Printf("Server is starting on port %s [%s] \n", os.Getenv("PORT"), srv.Addr)

	log.Fatal(srv.ListenAndServe())
}
