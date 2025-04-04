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

	"go.hasen.dev/vbolt"
	"go.hasen.dev/vpack"
)

type Image struct {
	Id       int
	OwnerId  int
	FamilyId int
	Filename string
}

func PackImage(self *Image, buf *vpack.Buffer) {
	vpack.Version(1, buf)
	vpack.Int(&self.Id, buf)
	vpack.Int(&self.OwnerId, buf)
	vpack.Int(&self.FamilyId, buf)
	vpack.String(&self.Filename, buf)
}

var ImageBucket = vbolt.Bucket(&Info, "image", vpack.FInt, PackImage)

func SaveImage(tx *vbolt.Tx, image *Image) {
	vbolt.Write(tx, ImageBucket, image.Id, image)
}

func RegisterImagePages(mux *http.ServeMux) {
	mux.Handle("POST /post/upload-image", AuthHandler(ContextFunc(uploadImage)))
	mux.Handle("POST /person/upload/{id}", AuthHandler(ContextFunc(uploadPersonImage)))
	mux.Handle("GET /person/upload/delete/{id}", AuthHandler(ContextFunc(deletePersonImage)))
	mux.Handle("GET /uploads/delete", AuthHandler(ContextFunc(deleteAllImages)))
	mux.Handle("GET /uploads/{id}", PublicHandler(ContextFunc(serveImage)))
}

func buildPath(filename string) (path string) {
	return filepath.Join("uploads", filename)
}

func SaveImageFile(context ResponseContext, fileParameter string) (image Image, err error) {
	context.r.Body = http.MaxBytesReader(context.w, context.r.Body, 10<<23)

	if err := context.r.ParseMultipartForm(10 << 23); err != nil {
		return image, err
	}

	file, handler, err := context.r.FormFile(fileParameter)
	if err != nil {
		return image, err
	}
	defer file.Close()

	filename := fmt.Sprintf("%d-%s", time.Now().Unix(), handler.Filename)

	if err = os.MkdirAll("uploads", os.ModePerm); err != nil {
		return image, err
	}

	dst, err := os.Create(buildPath(filename))
	if err != nil {
		return image, err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		return image, err
	}

	vbolt.WithWriteTx(db, func(tx *vbolt.Tx) {
		image = Image{
			Id:       vbolt.NextIntId(tx, ImageBucket),
			OwnerId:  context.user.Id,
			FamilyId: context.user.PrimaryFamilyId,
			Filename: filename,
		}

		SaveImage(tx, &image)
		tx.Commit()
	})

	return image, nil
}

func DeleteImageFile(imageId int, readTx *vbolt.Tx) (err error) {
	var image Image
	vbolt.Read(readTx, ImageBucket, imageId, &image)

	if err := os.Remove(buildPath(image.Filename)); err != nil {
		return err
	}

	vbolt.WithWriteTx(db, func(tx *vbolt.Tx) {
		vbolt.Delete(tx, ImageBucket, imageId)
		tx.Commit()
	})

	return nil
}

func uploadImage(context ResponseContext) {
	image, err := SaveImageFile(context, "image")
	if err != nil {
		http.Error(context.w, "Error saving image", http.StatusBadRequest)
		return
	}

	response := map[string]string{"url": fmt.Sprintf("/uploads/%d", image.Id)}
	context.w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(context.w).Encode(response)
}

func uploadPersonImage(context ResponseContext) {
	image, err := SaveImageFile(context, "profilePic")
	if err != nil || image.Id == 0 {
		http.Error(context.w, "Error saving image", http.StatusBadRequest)
		return
	}

	id := context.r.PathValue("id")
	idVal, _ := strconv.Atoi(id)

	var person Person
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		person = getPerson(tx, idVal)
		if person.ImageId > 0 {
			DeleteImageFile(person.ImageId, tx)
		}
	})
	vbolt.WithWriteTx(db, func(tx *vbolt.Tx) {
		person.ImageId = image.Id
		vbolt.Write(tx, PersonBucket, person.Id, &person)
		tx.Commit()
	})

	http.Redirect(context.w, context.r, "/person/"+id, http.StatusFound)
}

func deleteAllImages(context ResponseContext) {
	deleteIds := make([]int, 0)
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		vbolt.IterateAll(tx, ImageBucket, func(key int, value Image) bool {
			err := os.Remove(buildPath(value.Filename))
			if err != nil {
				http.Error(context.w, err.Error(), http.StatusBadRequest)
			}
			deleteIds = append(deleteIds, key)
			return true
		})
	})
	vbolt.WithWriteTx(db, func(tx *vbolt.Tx) {
		for _, deleteId := range deleteIds {
			vbolt.Delete(tx, ImageBucket, deleteId)
		}
		tx.Commit()
	})

	http.Redirect(context.w, context.r, "/", http.StatusFound)
}

func serveImage(context ResponseContext) {
	id := context.r.PathValue("id")
	idVal, _ := strconv.Atoi(id)
	filePath := ""

	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		var image Image
		vbolt.Read(tx, ImageBucket, idVal, &image)

		if image.OwnerId == context.user.Id {
			filePath = buildPath(image.Filename)
		}
	})

	if filePath == "" {
		http.Error(context.w, "cannot show image", http.StatusBadRequest)
	} else {
		http.ServeFile(context.w, context.r, filePath)
	}
}

func deletePersonImage(context ResponseContext) {
	id := context.r.PathValue("id")
	idVal, _ := strconv.Atoi(id)

	var person Person
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		person = getPerson(tx, idVal)
		if person.ImageId > 0 {
			DeleteImageFile(person.ImageId, tx)
		}
	})
	vbolt.WithWriteTx(db, func(tx *vbolt.Tx) {
		person.ImageId = 0
		vbolt.Write(tx, PersonBucket, person.Id, &person)
		tx.Commit()
	})

	http.Redirect(context.w, context.r, "/person/"+id, http.StatusFound)
}
