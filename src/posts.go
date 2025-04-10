package main

import (
	"net/http"
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
	FamilyId  int
	EntryDate time.Time

	Content string
}

func PackPost(self *Post, buf *vpack.Buffer) {
	vpack.Version(1, buf)
	vpack.Int(&self.Id, buf)
	vpack.Int(&self.PersonId, buf)
	vpack.Int(&self.FamilyId, buf)
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
			"People": getPeopleInFamily(tx, context.user.PrimaryFamilyId),
		})
	})
}

func editPostPage(context ResponseContext) {
	vbolt.WithReadTx(db, func(tx *bolt.Tx) {
		id := context.r.PathValue("id")
		idVal, _ := strconv.Atoi(id)
		RenderTemplateWithData(context, "posts-add", map[string]interface{}{
			"People": getPeopleInFamily(tx, context.user.PrimaryFamilyId),
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

	var person Person
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		person = getPerson(tx, personId)
	})

	entry := Post{
		Id:        id,
		PersonId:  personId,
		FamilyId:  person.FamilyId,
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
