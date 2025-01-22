package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	"go.hasen.dev/generic"
	"go.hasen.dev/vbolt"
	"go.hasen.dev/vpack"
)

type Post struct {
	Id        int
	PersonId  int
	EntryDate time.Time

	Content string
}

func PackPost(self *Post, buf *vpack.Buffer) {
	vpack.Version(1, buf)
	vpack.Int(&self.Id, buf)
	vpack.Int(&self.PersonId, buf)
	vpack.UnixTime(&self.EntryDate, buf)
	vpack.String(&self.Content, buf)
}

var PostBucket = vbolt.Bucket(&Info, "posts", vpack.FInt, PackPost)

func SavePost(tx *vbolt.Tx, post *Post) {
	vbolt.Write(tx, PostBucket, post.Id, post)
}

func getAllPosts(tx *vbolt.Tx) (posts []Post) {
	vbolt.IterateAll(tx, PostBucket, func(key int, value Post) bool {
		generic.Append(&posts, value)
		return true
	})
	return posts
}

func getPost(tx *vbolt.Tx, id int) (post Post) {
	vbolt.Read(tx, PostBucket, id, &post)
	return post
}

func RegisterPostPages(mux *http.ServeMux) {
	mux.HandleFunc("GET /posts", func(w http.ResponseWriter, r *http.Request) {
		vbolt.WithReadTx(db, func(tx *bolt.Tx) {
			RenderTemplate(w, "posts", getAllPosts(tx))
		})
	})
	mux.HandleFunc("GET /posts/add", func(w http.ResponseWriter, r *http.Request) {
		vbolt.WithReadTx(db, func(tx *bolt.Tx) {
			RenderTemplate(w, "posts-add", struct {
				People []Person
				Post   Post
			}{
				People: getAllPeople(tx),
			})
		})
	})
	mux.HandleFunc("GET /posts/edit/{id}", func(w http.ResponseWriter, r *http.Request) {
		vbolt.WithReadTx(db, func(tx *bolt.Tx) {
			id := r.PathValue("id")
			idVal, _ := strconv.Atoi(id)
			RenderTemplate(w, "posts-add", struct {
				People []Person
				Post   Post
			}{
				People: getAllPeople(tx),
				Post:   getPost(tx, idVal),
			})
		})
	})
	mux.HandleFunc("GET /posts/delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		vbolt.WithWriteTx(db, func(tx *bolt.Tx) {
			id := r.PathValue("id")
			idVal, _ := strconv.Atoi(id)
			vbolt.Delete(tx, PostBucket, idVal)
			vbolt.TxCommit(tx)
		})

		http.Redirect(w, r, "/posts", http.StatusFound)
	})
	mux.HandleFunc("POST /posts/add", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		entryDate := r.FormValue("entryDate")
		personId, _ := strconv.Atoi(r.FormValue("personId"))
		content := r.FormValue("quill-content")
		id, _ := strconv.Atoi(r.FormValue("id"))

		entryDateTime, _ := time.Parse("2006-01-02", entryDate)

		entry := Post{
			Id:        id,
			PersonId:  personId,
			EntryDate: entryDateTime,
			Content:   content,
		}

		vbolt.WithWriteTx(db, func(tx *bolt.Tx) {
			if entry.Id == 0 {
				entry.Id = vbolt.NextIntId(tx, PostBucket)
			}
			vbolt.Write(tx, PostBucket, entry.Id, &entry)
			vbolt.TxCommit(tx)
		})

		http.Redirect(w, r, "/posts", http.StatusFound)
	})
	mux.HandleFunc("POST /post/upload-image", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 10<<23)

		if err := r.ParseMultipartForm(10 << 23); err != nil {
			http.Error(w, "File too large", http.StatusBadRequest)
			return
		}

		file, handler, err := r.FormFile("image")
		if err != nil {
			http.Error(w, "Error retrieving file", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		filename := fmt.Sprintf("%d-%s", time.Now().Unix(), handler.Filename)
		savePath := filepath.Join("uploads", filename)

		if err := os.MkdirAll("uploads", os.ModePerm); err != nil {
			http.Error(w, "Error creating directory", http.StatusInternalServerError)
			return
		}

		dst, err := os.Create(savePath)
		if err != nil {
			http.Error(w, "Error saving file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()
		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, "Error saving file", http.StatusInternalServerError)
			return
		}

		fileURL := fmt.Sprintf("/uploads/%s", filename)
		response := map[string]string{"url": fileURL}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))
}
