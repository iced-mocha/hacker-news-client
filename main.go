package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/iced-mocha/shared/models"
	"github.com/patrickmn/go-cache"
)

const (
	defaultPostCount = 20
	port             = ":4000"
	feedURL          = "https://hacker-news.firebaseio.com/v0/topstories.json"
	postURL          = "https://hacker-news.firebaseio.com/v0/item/"
	baseURL          = "http://hacker-news-client" + port
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

func subtract(dataset []int, toRemove []int) []int {
	inToRemove := make(map[int]bool)
	for _, id := range toRemove {
		inToRemove[id] = true
	}

	toReturn := make([]int, 0)
	for _, id := range dataset {
		if !inToRemove[id] {
			toReturn = append(toReturn, id)
		}
	}

	return toReturn
}

func getNextIDs(allIDs, viewedIDs []int, count int) []int {
	unviewedIDs := subtract(allIDs, viewedIDs)
	if len(unviewedIDs) >= count {
		return unviewedIDs[:count]
	}
	return unviewedIDs
}

func GetPosts(w http.ResponseWriter, r *http.Request, c *cache.Cache, id func() string) {
	var err error

	queryParams := r.URL.Query()
	postCountToReturn := defaultPostCount
	if arr, ok := queryParams["count"]; ok && len(arr) > 0 {
		postCountToReturn, err = strconv.Atoi(arr[0])
	}
	if err != nil {
		http.Error(w, "'count' could not be converted to an integer", http.StatusNotFound)
		return
	}

	var pagingToken string
	if arr, ok := queryParams["continue"]; ok && len(arr) > 0 {
		pagingToken = arr[0]
	}

	resp, err := http.Get(feedURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	var allPostIDs []int
	err = json.NewDecoder(resp.Body).Decode(&allPostIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	viewedIDs := make([]int, 0)
	if v, ok := c.Get(pagingToken); ok {
		viewedIDs = v.([]int)
	}

	nextIDs := getNextIDs(allPostIDs, viewedIDs, postCountToReturn)

	var nextURL string
	viewedIDs = append(viewedIDs, nextIDs...)
	if len(subtract(allPostIDs, viewedIDs)) != 0 {
		nextToken := id()
		nextURL = baseURL + "/v1/posts?continue=" + nextToken + "&count=" + strconv.Itoa(postCountToReturn)
		c.Set(nextToken, viewedIDs, cache.DefaultExpiration)
	}

	respChans := make([]chan Response, 0)
	for _, id := range nextIDs {
		cID := id
		respChan := make(chan Response)
		respChans = append(respChans, respChan)
		go func() {
			resp, err := http.Get(postURL + strconv.Itoa(cID) + ".json")
			respChan <- Response{resp, err}
		}()
	}

	var postsToReturn []models.Post
	for _, respChan := range respChans {
		response := <-respChan
		err := response.err
		resp := response.resp
		defer resp.Body.Close()
		if err != nil {
			log.Printf("error getting post: %v", err)
			continue
		}
		var post Post
		err = json.NewDecoder(resp.Body).Decode(&post)
		if err != nil {
			log.Printf("error decoding response body %v: %v", resp.Body, err)
			continue
		}
		postToReturn := models.Post{
			ID:       strconv.Itoa(post.ID),
			Date:     time.Unix(post.Time, 0),
			Author:   post.By,
			Title:    post.Title,
			PostLink: "https://news.ycombinator.com/item?id=" + strconv.Itoa(post.ID),
			URL: post.URL,
			Platform: models.PlatformHackerNews}
		postsToReturn = append(postsToReturn, postToReturn)
	}

	cRes := models.ClientResp{postsToReturn, nextURL}
	res, err := json.Marshal(cRes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}

func main() {
	var idCounter int32
	id := func() string {
		idCounter++
		return strconv.FormatInt(int64(idCounter), 32)
	}

	var c *cache.Cache
	c = cache.New(30*time.Minute, 45*time.Minute)

	f := func(w http.ResponseWriter, r *http.Request) {
		GetPosts(w, r, c, id)
	}

	r := mux.NewRouter()
	r.HandleFunc("/v1/posts", f).Methods(http.MethodGet)
	log.Printf("starting server on port " + port)
	log.Fatal(http.ListenAndServe(port, r))
}
