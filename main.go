package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/nfnt/resize"
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("photo")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), filepath.Ext(handler.Filename))

	f, err := os.OpenFile(filepath.Join("uploads", filename), os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		http.Error(w, "Error saving the file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	_, err = io.Copy(f, file)
	if err != nil {
		http.Error(w, "Error copying file", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "File uploaded successfully: %s", filename)
}

func galleryHandler(w http.ResponseWriter, r *http.Request) {
	files, err := filepath.Glob("uploads/*")
	if err != nil {
		http.Error(w, "Error reading the gallery", http.StatusInternalServerError)
		return
	}

	templatePath := filepath.Join("html", "gallery.html")

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		http.Error(w, "Error rendering the gallery", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, files)
}

func listGalleryHandler(w http.ResponseWriter, r *http.Request) {
	files, err := filepath.Glob("uploads/*")
	if err != nil {
		http.Error(w, "Error reading the gallery", http.StatusInternalServerError)
		return
	}

	var htmlBuffer bytes.Buffer
	htmlBuffer.WriteString("<html><body>")

	for _, file := range files {
		imageData, err := ioutil.ReadFile(file)
		if err != nil {
			http.Error(w, "Error reading image file", http.StatusInternalServerError)
			return
		}

		img, _, err := image.Decode(bytes.NewReader(imageData))
		if err != nil {
			http.Error(w, "Error decoding image", http.StatusInternalServerError)
			return
		}

		resizedImg := resize.Resize(uint(img.Bounds().Dx()/4), uint(img.Bounds().Dy()/4), img, resize.Lanczos3)

		var resizedBuffer bytes.Buffer
		if err := png.Encode(&resizedBuffer, resizedImg); err != nil {
			http.Error(w, "Error encoding resized image", http.StatusInternalServerError)
			return
		}
		base64ResizedImage := base64.StdEncoding.EncodeToString(resizedBuffer.Bytes())

		imageTag := fmt.Sprintf(`<img src="data:image/png;base64,%s" alt="%s" />`, base64ResizedImage, filepath.Base(file))

		htmlBuffer.WriteString(imageTag)
	}

	htmlBuffer.WriteString("</body></html>")

	w.Header().Set("Content-Type", "text/html")
	htmlBuffer.WriteTo(w)
}

func main() {
	os.Mkdir("uploads", os.ModePerm)

	r := mux.NewRouter()

	r.HandleFunc("/upload", uploadHandler).Methods("POST")
	r.HandleFunc("/", galleryHandler)
	r.HandleFunc("/api/gallery", listGalleryHandler).Methods("GET")

	port := ":8080"
	fmt.Printf("Server is running on http://localhost%s\n", port)
	http.ListenAndServe(port, r)
}
