package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	http.Handle("/", http.FileServer(http.Dir("public")))

	http.HandleFunc("/video", handleVideo)

	addr := "localhost:4242"
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func handleVideo(w http.ResponseWriter, r *http.Request) {

	log.Printf("HandleVideo")

	file, err := os.Open("alps.mp4")
	if err != nil {
		http.Error(w, "Video not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "video/mp4")

	stat, _ := file.Stat()
	http.ServeContent(w, r, stat.Name(), stat.ModTime(), file)
}
