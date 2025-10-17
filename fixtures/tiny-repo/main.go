package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/api/users", usersHandler)
	http.HandleFunc("/health", healthHandler)

	fmt.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to Tiny API")
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	users := []string{"Alice", "Bob", "Charlie"}
	for _, user := range users {
		fmt.Fprintf(w, "User: %s\n", user)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}
