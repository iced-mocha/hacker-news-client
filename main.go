package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/iced-mocha/shared/models"
)

const (
	defaultPostCount = 20
	feedURL          = "https://hacker-news.firebaseio.com/v0/beststories.json"
	postURL          = "https://hacker-news.firebaseio.com/v0/item/"
)

type Response struct {
	resp *http.Response
	err  error
}

type Post struct {
	By          string
	Descendants int
	ID          int
	Kids        []int
	Score       int
	Time        int64
	Title       string
	Type        string
	URL         string
}

func GetPosts(w http.ResponseWriter, r *http.Request) {
	var err error

	resp, err := http.Get(feedURL)
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

	queryParams := r.URL.Query()
	postCountToReturn := defaultPostCount
	if arr, ok := queryParams["count"]; ok && len(arr) > 0 {
		postCountToReturn, err = strconv.Atoi(arr[0])
	}
	if err != nil {
		http.Error(w, "'count' could not be converted to an integer", http.StatusNotFound)
		return
	}
	if postCountToReturn > len(postIds) {
		postCountToReturn = len(postIds)
	}

	respChan := make(chan Response, postCountToReturn)
	for i := 0; i < postCountToReturn; i++ {
		id := postIds[i]
		go func() {
			resp, err := http.Get(postURL + strconv.Itoa(id) + ".json")
			respChan <- Response{resp, err}
		}()
	}

	var postsToReturn []models.Post
	for i := 0; i < postCountToReturn; i++ {
		response := <-respChan
		err := response.err
		resp := response.resp
		if err != nil {
			continue
		}
		var post Post
		err = json.NewDecoder(resp.Body).Decode(&post)
		if err != nil {
			continue
		}
		postToReturn := models.Post{
			ID:       strconv.Itoa(post.ID),
			Date:     time.Unix(post.Time, 0),
			Author:   post.By,
			Title:    post.Title,
			PostLink: post.URL,
			Platform: models.PlatformHackerNews}
		postsToReturn = append(postsToReturn, postToReturn)
	}

	res, err := json.Marshal(postsToReturn)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/v1/posts", GetPosts).Methods(http.MethodGet)
	log.Fatal(http.ListenAndServe(":4000", r))
}
