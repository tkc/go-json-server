package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
)

func dumpJsonRequestHandlerFunc(w http.ResponseWriter, req *http.Request) {

	if req.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	length, err := strconv.Atoi(req.Header.Get("Content-Length"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Print(length)

	body := make([]byte, length)
	length, err = req.Body.Read(body)

	if err != nil && err != io.EOF {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var jsonBody map[string]interface{}
	err = json.Unmarshal(body[:length], &jsonBody)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Printf("%v\n", jsonBody)
	// w.WriteHeader(http.StatusOK)
}

func main() {
	http.HandleFunc("/json", dumpJsonRequestHandlerFunc)
	http.ListenAndServe(":8080", nil)
}
