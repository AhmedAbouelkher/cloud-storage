package main

import (
	"fmt"
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

	OpenInMemoryCache()

	if err := OpenDBConnection(); err != nil {
		panic(err)
	}

	r := mux.NewRouter()
	logger := log.New(os.Stdout, "", log.LstdFlags)

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		SendJson(w, http.StatusOK, Payload{
			"message":            "Welcome to Cloud Storage!",
			"description":        "A simple object storage with golang",
			"postman_collection": "https://github.com/AhmedAbouelkher/cloud-storage/blob/master/postman/cloud-storage.postman_collection.json",
		})
	}).Methods(http.MethodGet)

	r.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		SendJson(w, http.StatusOK, Payload{"message": "pong"})
	}).Methods(http.MethodGet)

	// Buckets
	r.HandleFunc("/bucket", HandleBucketCreation).Methods(http.MethodPost)
	r.HandleFunc("/bucket/{name}", HandleBucketDeletion).Methods(http.MethodDelete)
	r.HandleFunc("/bucket/{name}/objects", HandleObjectsFetch).Methods(http.MethodGet)

	// Objects
	r.HandleFunc("/object", HandleObjectCreation).Methods(http.MethodPost)
	r.HandleFunc("/objects", HandleObjectsCreation).Methods(http.MethodPost)
	r.HandleFunc("/object/{uuid}/external", HandleGeneratingSharableLink).Methods(http.MethodPost)
	r.HandleFunc("/object/{uuid}", HandleObjectDeletion).Methods(http.MethodDelete)
	r.HandleFunc("/object/{uuid}", HandleObjectFetch).Methods(http.MethodGet)

	// Resources
	r.HandleFunc("/resource/s3", HandleResourceFetchWithS3).Methods(http.MethodGet)
	r.HandleFunc("/resource/s3/objects", HandleResourceFilesFetchWithS3).Methods(http.MethodGet)
	r.HandleFunc("/resource", HandleResourceCreation).Methods(http.MethodPost)
	r.HandleFunc("/resource", HandleResourceDeletion).Methods(http.MethodDelete)

	// Object Serving
	r.PathPrefix("/share").HandlerFunc(HandleServingRequestedObject).Methods(http.MethodGet)
	r.PathPrefix("/").HandlerFunc(HandleServingDirectObject).Methods(http.MethodGet)

	// middlewares
	r.Use(func(n http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("content-type", "application/json")
			w.Header().Set("cache-control", "no-cache")
			n.ServeHTTP(w, r)
		})
	})
	r.Use(NewLogMiddleware(logger).Func())

	router := func() http.Handler {
		rps := 5.0
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

	fmt.Printf("Server is starting on port %s [%s] \n", os.Getenv("PORT"), srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
