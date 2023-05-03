package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"time"
)

var CONNECTED_NODE_ADDRESS string = "http://127.0.0.1:8000"
var posts []map[string]interface{}

func fetchPosts() {
	resp, err := http.Get(fmt.Sprintf("%s/chain", CONNECTED_NODE_ADDRESS))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var chainData map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&chainData)
	if err != nil {
		return
	}

	content := []map[string]interface{}{}
	chain := chainData["chain"].([]interface{})
	for _, blockData := range chain {
		block := blockData.(map[string]interface{})
		for _, txData := range block["transactions"].([]interface{}) {
			tx := txData.(map[string]interface{})
			tx["index"] = block["index"]
			tx["hash"] = block["previous_hash"]
			content = append(content, tx)
		}
	}

	posts = content
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fetchPosts()
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Title         string
		Posts         []map[string]interface{}
		NodeAddress   string
		ReadableTime  func(float64) string
	}{
		Title:        "YourNet: Decentralized content sharing",
		Posts:        posts,
		NodeAddress:  CONNECTED_NODE_ADDRESS,
		ReadableTime: timestampToString,
	}

	tmpl.Execute(w, data)
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
	postContent := r.FormValue("content")
	author := r.FormValue("author")

	post := map[string]string{
		"author":  author,
		"content": postContent,
	}
	postData, _ := json.Marshal(post)

	newTxAddress := fmt.Sprintf("%s/new_transaction", CONNECTED_NODE_ADDRESS)
	resp, err := http.Post(newTxAddress, "application/json", bytes.NewBuffer(postData))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func timestampToString(epochTime float64) string {
	return time.Unix(int64(epochTime), 0).Format("15:04")
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/submit", submitHandler)

	fmt.Println("Listening on port 8080")
	http.ListenAndServe(":8080", nil)
}
