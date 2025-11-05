package tools
package main

import (
	"encoding/json"
	"fmt"
	package main

	import (
		"encoding/json"
		"fmt"
		"log"
		"net"
		"net/http"
	)

	func main() {
		h := http.NewServeMux()

		h.HandleFunc("/api/v3/user/repos", func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query().Get("page")
			w.Header().Set("Content-Type", "application/json")
			fmt.Printf("Server received: %s %s?%s\n", r.Method, r.URL.Path, r.URL.RawQuery)
			switch q {
			case "", "1":
				w.Header().Set("Link", `<http://`+r.Host+`/api/v3/user/repos?affiliation=owner&page=2&per_page=100>; rel="next", <http://`+r.Host+`/api/v3/user/repos?affiliation=owner&page=2&per_page=100>; rel="last"`)
				w.WriteHeader(200)
				_ = json.NewEncoder(w).Encode([]map[string]interface{}{
					{
						"id":        1,
						"name":      "repo-1",
						"full_name": "user/repo-1",
						"html_url":  "https://github.com/user/repo-1",
						"owner":     map[string]interface{}{"login": "user"},
					},
				})
			case "2":
				w.WriteHeader(200)
				_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
			default:
				w.WriteHeader(200)
				_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
			}
		})

		h.HandleFunc("/api/v3/repos/user/repo-1/actions/runs", func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("Server received workflow runs: %s %s?%s\n", r.Method, r.URL.Path, r.URL.RawQuery)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"workflow_runs": []map[string]interface{}{},
				"total_count":   0,
			})
		})

		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Fatal(err)
		}
		addr := ln.Addr().String()
		log.Printf("Mock server listening on http://%s", addr)
		hServer := &http.Server{Handler: h}
		go func() {
			if err := hServer.Serve(ln); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server error: %v", err)
			}
		}()

		package main

		import (
			"encoding/json"
			"fmt"
			"log"
			"net"
			"net/http"
		)

		func main() {
			h := http.NewServeMux()

			h.HandleFunc("/api/v3/user/repos", func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query().Get("page")
				w.Header().Set("Content-Type", "application/json")
				fmt.Printf("Server received: %s %s?%s\n", r.Method, r.URL.Path, r.URL.RawQuery)
				switch q {
				case "", "1":
					w.Header().Set("Link", `<http://`+r.Host+`/api/v3/user/repos?affiliation=owner&page=2&per_page=100>; rel="next", <http://`+r.Host+`/api/v3/user/repos?affiliation=owner&page=2&per_page=100>; rel="last"`)
					w.WriteHeader(200)
					_ = json.NewEncoder(w).Encode([]map[string]interface{}{
						{
							"id":        1,
							"name":      "repo-1",
							"full_name": "user/repo-1",
							"html_url":  "https://github.com/user/repo-1",
							"owner":     map[string]interface{}{"login": "user"},
						},
					})
				case "2":
					w.WriteHeader(200)
					_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
				default:
					w.WriteHeader(200)
					_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
				}
			})

			h.HandleFunc("/api/v3/repos/user/repo-1/actions/runs", func(w http.ResponseWriter, r *http.Request) {
				fmt.Printf("Server received workflow runs: %s %s?%s\n", r.Method, r.URL.Path, r.URL.RawQuery)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(200)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"workflow_runs": []map[string]interface{}{},
					"total_count":   0,
				})
			})

			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				log.Fatal(err)
			}
			addr := ln.Addr().String()
			log.Printf("Mock server listening on http://%s", addr)
			hServer := &http.Server{Handler: h}
			go func() {
				if err := hServer.Serve(ln); err != nil && err != http.ErrServerClosed {
					log.Fatalf("Server error: %v", err)
				}
			}()

			// block forever
			select {}
		}
