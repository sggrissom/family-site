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

	"image/jpeg"

	"github.com/disintegration/imaging"
)

type AccessLevel int

const (
	OwnerLevel AccessLevel = iota
	FamilyLevel
	ViewerLevel
	PublicLevel
)

type Image struct {
	Id             int
	OwnerId        int
	FamilyId       int
	Filename       string
	Small_Filename string
	Access         AccessLevel
}

func PackImage(self *Image, buf *vpack.Buffer) {
	vpack.Version(1, buf)
	vpack.Int(&self.Id, buf)
	vpack.Int(&self.OwnerId, buf)
	vpack.Int(&self.FamilyId, buf)
	vpack.String(&self.Filename, buf)
	vpack.String(&self.Small_Filename, buf)
	vpack.IntEnum(&self.Access, buf)
}

var ImageBucket = vbolt.Bucket(&Info, "image", vpack.FInt, PackImage)

func SaveImage(tx *vbolt.Tx, image *Image) {
	vbolt.Write(tx, ImageBucket, image.Id, image)
}

func RegisterImagePages(mux *http.ServeMux) {
	mux.Handle("POST /post/upload-image", AuthHandler(ContextFunc(uploadImage)))
	mux.Handle("POST /person/upload/{id}", AuthHandler(ContextFunc(uploadPersonImage)))
	mux.Handle("GET /person/upload/delete/{id}", AuthHandler(ContextFunc(deletePersonImage)))
	mux.Handle("POST /family/upload/{id}", AuthHandler(ContextFunc(uploadFamilyImage)))
	mux.Handle("GET /family/upload/delete/{id}", AuthHandler(ContextFunc(deleteFamilyImage)))
	mux.Handle("GET /uploads/delete", AuthHandler(ContextFunc(deleteAllImages)))
	mux.Handle("GET /uploads/{id}", PublicHandler(ContextFunc(serveImage)))
}

func buildPath(filename string) (path string) {
	return filepath.Join("uploads", filename)
}

func SaveImageFile(context ResponseContext, fileParameter string, access AccessLevel) (image Image, err error) {
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

	origImage, err := imaging.Open(buildPath(filename))
	if err != nil {
		return image, err
	}

	resizedImage := imaging.Thumbnail(origImage, 150, 150, imaging.Lanczos)

	smallFilename := fmt.Sprintf("small-%s", filename)
	smallFile, err := os.Create(buildPath(smallFilename))
	if err != nil {
		return image, err
	}
	defer smallFile.Close()

	opts := jpeg.Options{Quality: 80}
	if err := jpeg.Encode(smallFile, resizedImage, &opts); err != nil {
		return image, err
	}

	vbolt.WithWriteTx(db, func(tx *vbolt.Tx) {
		image = Image{
			Id:             vbolt.NextIntId(tx, ImageBucket),
			OwnerId:        context.user.Id,
			FamilyId:       context.user.PrimaryFamilyId,
			Filename:       filename,
			Small_Filename: smallFilename,
			Access:         access,
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
	if len(image.Small_Filename) > 0 {
		if err := os.Remove(buildPath(image.Small_Filename)); err != nil {
			return err
		}
	}

	vbolt.WithWriteTx(db, func(tx *vbolt.Tx) {
		vbolt.Delete(tx, ImageBucket, imageId)
		tx.Commit()
	})

	return nil
}

func uploadImage(context ResponseContext) {
	image, err := SaveImageFile(context, "image", FamilyLevel)
	if err != nil {
		http.Error(context.w, "Error saving image", http.StatusBadRequest)
		return
	}

	response := map[string]string{"url": fmt.Sprintf("/uploads/%d", image.Id)}
	context.w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(context.w).Encode(response)
}

func uploadPersonImage(context ResponseContext) {
	image, err := SaveImageFile(context, "profilePic", FamilyLevel)
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

func uploadFamilyImage(context ResponseContext) {
	image, err := SaveImageFile(context, "profilePic", PublicLevel)
	if err != nil || image.Id == 0 {
		http.Error(context.w, "Error saving image", http.StatusBadRequest)
		return
	}

	id := context.r.PathValue("id")
	idVal, _ := strconv.Atoi(id)

	var family Family
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		family = getFamily(tx, idVal)
		if family.ImageId > 0 {
			DeleteImageFile(family.ImageId, tx)
		}
	})
	vbolt.WithWriteTx(db, func(tx *vbolt.Tx) {
		family.ImageId = image.Id
		vbolt.Write(tx, FamilyBucket, idVal, &family)
		tx.Commit()
	})

	http.Redirect(context.w, context.r, "/family/edit/"+id, http.StatusFound)
}

func deleteAllImages(context ResponseContext) {
	deleteIds := make([]int, 0)
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		vbolt.IterateAll(tx, ImageBucket, func(key int, value Image) bool {
			err := os.Remove(buildPath(value.Filename))
			if err != nil {
				http.Error(context.w, err.Error(), http.StatusBadRequest)
			}
			if len(value.Small_Filename) > 0 {
				err = os.Remove(buildPath(value.Small_Filename))
				if err != nil {
					http.Error(context.w, err.Error(), http.StatusBadRequest)
				}
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

	var image Image
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		vbolt.Read(tx, ImageBucket, idVal, &image)
		if len(image.Small_Filename) > 0 {
			filePath = buildPath(image.Small_Filename)
		} else {
			filePath = buildPath(image.Filename)
		}
	})

	if image.Access == OwnerLevel && image.OwnerId != context.user.Id {
		filePath = ""
	}
	if image.Access == FamilyLevel && image.FamilyId != context.user.PrimaryFamilyId {
		filePath = ""
	}

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

func deleteFamilyImage(context ResponseContext) {
	id := context.r.PathValue("id")
	idVal, _ := strconv.Atoi(id)

	var family Family
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		family = getFamily(tx, idVal)
		if family.ImageId > 0 {
			DeleteImageFile(family.ImageId, tx)
		}
	})
	vbolt.WithWriteTx(db, func(tx *vbolt.Tx) {
		family.ImageId = 0
		vbolt.Write(tx, FamilyBucket, family.Id, &family)
		tx.Commit()
	})

	http.Redirect(context.w, context.r, "/family/edit/"+id, http.StatusFound)
}
