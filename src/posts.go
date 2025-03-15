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
	mux.Handle("GET /posts", PublicHandler(ContextFunc(postsPage)))
	mux.Handle("GET /posts/add", AuthHandler(ContextFunc(addPostPage)))
	mux.Handle("GET /posts/edit/{id}", AuthHandler(ContextFunc(editPostPage)))
	mux.Handle("GET /posts/delete/{id}", AuthHandler(ContextFunc(deletePost)))
	mux.Handle("POST /posts/add", AuthHandler(ContextFunc(savePost)))
	mux.Handle("POST /post/upload-image", AuthHandler(ContextFunc(uploadImage)))

	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))
}

func postsPage(context ResponseContext) {
	vbolt.WithReadTx(db, func(tx *bolt.Tx) {
		RenderTemplateWithData(context, "posts", map[string]interface{}{
			"Posts": getAllPosts(tx),
		})
	})
}
func addPostPage(context ResponseContext) {
	vbolt.WithReadTx(db, func(tx *bolt.Tx) {
		RenderTemplateWithData(context, "posts-add", map[string]interface{}{
			"People": getAllPeople(tx),
		})
	})
}
func editPostPage(context ResponseContext) {
	vbolt.WithReadTx(db, func(tx *bolt.Tx) {
		id := context.r.PathValue("id")
		idVal, _ := strconv.Atoi(id)
		RenderTemplateWithData(context, "posts-add", map[string]interface{}{
			"People": getAllPeople(tx),
			"Post":   getPost(tx, idVal),
		})
	})
}

func deletePost(context ResponseContext) {
	vbolt.WithWriteTx(db, func(tx *bolt.Tx) {
		id := context.r.PathValue("id")
		idVal, _ := strconv.Atoi(id)
		vbolt.Delete(tx, PostBucket, idVal)
		vbolt.TxCommit(tx)
	})

	http.Redirect(context.w, context.r, "/posts", http.StatusFound)
}
func savePost(context ResponseContext) {
	context.r.ParseForm()
	entryDate := context.r.FormValue("entryDate")
	personId, _ := strconv.Atoi(context.r.FormValue("personId"))
	content := context.r.FormValue("quill-content")
	id, _ := strconv.Atoi(context.r.FormValue("id"))

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

	http.Redirect(context.w, context.r, "/posts", http.StatusFound)
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
