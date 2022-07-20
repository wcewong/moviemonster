/*
	HTTP server is started on port 6262.

	Instructions:
		(1) Edit values in config.go
		(2) go *.go

	Movie ID from TMDB (for testing purposes)
		500 - Reservoir Dogs
		501 - Grizzly Man
		502 - Fail Safe

	Of course, this should work with any of the IDs in TMDB. Note that TMDB appears to have
	"skipped" certain numbers so not every ID is populated
*/

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
)

func fetchMovie(movieId string) <-chan string {
	fmt.Println("Fetching movie ...")

	c := make(chan string)

	go func() {
		url := apiUrlPrefix + movieId + apiUrlSuffix
		fmt.Println(url)

		// Get data from TMDB's API
		response, err := http.Get(url)
		if err != nil {
			fmt.Print(err.Error())
			os.Exit(1)
		}

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Fatalln(err)
		}

		// Handle status code here
		c <- string(body)
	}()

	return c
}

// /movie/:movieId route handler

// Global var to hold our movies
var cache = make(map[string]Movie)

func getMovie(w http.ResponseWriter, _ *http.Request, p httprouter.Params) {
	movieId := p.ByName("movieId")

	// Query our cache for the data first
	if val, ok := cache[movieId]; ok {
		// Movie data is available. Serve from our local cache
		fmt.Println("Movie Id: " + movieId + " is in our local cache. Serving from cache")
		fmt.Println(val)

	} else {
		// Movie data is not in local cache. Fetch from TMDB
		fmt.Println("Movie Id: " + movieId + " is not in our local cache. Fetch from TMDB")

		movieJson := <-fetchMovie(movieId)
		movieBytes := []byte(movieJson)

		targetMovie := Movie{}
		err := json.Unmarshal(movieBytes, &targetMovie)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Store in local cache
		cache[movieId] = targetMovie
	}

	// Marshal to JSON and print data
	val := cache[movieId]
	results, err := json.Marshal(val)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Must cast to string because Marshal returns results as a byte array
	io.WriteString(w, string(results))

	// Save poster image to disk
	fmt.Println(imageUrlPrefix + val.PosterPath)
	url := imageUrlPrefix + val.PosterPath

	response, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		return
	}

	if response.StatusCode != 200 {
		fmt.Println("Received a non 200 HTTP response code")
		return
	}

	// Create an empty file
	posterPath := val.PosterPath

	// Strip leading / character from localFilename
	localFilename := posterPath[1:]

	file, err := os.Create(localFilename)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer file.Close()

	// Write bytes to the file
	_, err = io.Copy(file, response.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Done!")
}

// Root route handler
func getRoot(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	io.WriteString(w, "Movie monster, at your service")
}

func main() {
	// Set up routes
	router := httprouter.New()
	router.GET("/", getRoot)
	router.GET("/movie/:movieId", getMovie)

	// Spawn HTTP server
	fmt.Println("Starting HTTP server on port 6262")
	err := http.ListenAndServe(":6262", router)

	if errors.Is(err, http.ErrServerClosed) {
		fmt.Println("Server closed")
	} else if err != nil {
		fmt.Printf("Error starting server: %s\n", err)
		os.Exit(1)
	}
}
