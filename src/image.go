package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.hasen.dev/vbolt"
	"go.hasen.dev/vpack"
)

type Image struct {
	Id       int
	OwnerId  int
	FamilyId int
	Path     string
	Filename string
}

func PackImage(self *Image, buf *vpack.Buffer) {
	vpack.Version(1, buf)
	vpack.Int(&self.Id, buf)
	vpack.Int(&self.OwnerId, buf)
	vpack.Int(&self.FamilyId, buf)
	vpack.String(&self.Path, buf)
	vpack.String(&self.Filename, buf)
}

var ImageBucket = vbolt.Bucket(&Info, "image", vpack.FInt, PackImage)

func SaveImage(tx *vbolt.Tx, image *Image) {
	vbolt.Write(tx, ImageBucket, image.Id, image)
}

func RegisterImagePages(mux *http.ServeMux) {
	mux.Handle("POST /post/upload-image", AuthHandler(ContextFunc(uploadImage)))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))
}

func uploadImage(context ResponseContext) {
	context.r.Body = http.MaxBytesReader(context.w, context.r.Body, 10<<23)

	if err := context.r.ParseMultipartForm(10 << 23); err != nil {
		http.Error(context.w, "File too large", http.StatusBadRequest)
		return
	}

	file, handler, err := context.r.FormFile("image")
	if err != nil {
		http.Error(context.w, "Error retrieving file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	filename := fmt.Sprintf("%d-%s", time.Now().Unix(), handler.Filename)
	savePath := filepath.Join("uploads", filename)

	if err := os.MkdirAll("uploads", os.ModePerm); err != nil {
		http.Error(context.w, "Error creating directory", http.StatusInternalServerError)
		return
	}

	dst, err := os.Create(savePath)
	if err != nil {
		http.Error(context.w, "Error saving file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(context.w, "Error saving file", http.StatusInternalServerError)
		return
	}

	fileURL := fmt.Sprintf("/uploads/%s", filename)
	response := map[string]string{"url": fileURL}
	context.w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(context.w).Encode(response)
}
