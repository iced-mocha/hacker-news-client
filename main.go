package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	//"github.com/iced-mocha/shared/models"
)

const (
	DEF_POST_COUNT = 20
)

type Response struct {
	resp *http.Response
	err  error
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/v1/posts", GetPosts).Methods("GET")

	log.Fatal(http.ListenAndServe(":4000", r))
}

func GetPosts(w http.ResponseWriter, r *http.Request) {
	var err error
	resp, err := http.Get("https://hacker-news.firebaseio.com/v0/beststories.json")
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	var postIds []int
	err = json.NewDecoder(resp.Body).Decode(&postIds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
        fmt.Printf("got list of post ids\n")
	queryParams := r.URL.Query()
	postCountToReturn := DEF_POST_COUNT
	if arr, ok := queryParams["count"]; ok && len(arr) > 0 {
		postCountToReturn, err = strconv.Atoi(arr[0])
		if err != nil {
			http.Error(w, "'count' could not be converted to an integer", http.StatusNotFound)
		}
		return
	}
	if postCountToReturn > len(postIds) {
		postCountToReturn = len(postIds)
	}
        fmt.Printf("posts are gonna be retreived\n")
	respChan := make(chan Response, postCountToReturn)
	for i := 0; i < postCountToReturn; i++ {
		go func() {
			resp, err := http.Get("https://hacker-news.firebaseio.com/v0/beststories.json")
			fmt.Printf("sending data into response channel\n")
			respChan <- Response{resp, err}
		}()
	}
	// var postsToReturn models.Post
	for i := 0; i < postCountToReturn; i++ {
		fmt.Printf("getting data from response channel\n")
		response := <-respChan
		if response.err != nil {
			continue
		}
		var post interface{}
		err = json.NewDecoder(resp.Body).Decode(&post)
		if err != nil {
			continue
		}
		fmt.Printf("Response is %v\n", post)
	}
}
