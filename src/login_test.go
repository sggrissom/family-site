package main

import (
	"os"
	"testing"

	"go.hasen.dev/vbolt"
)

func TestUserCreation(t *testing.T) {
	testDBPath := "test.db"
	db := vbolt.Open(testDBPath)
	vbolt.InitBuckets(db, &Info)
	defer os.Remove(testDBPath)
	defer db.Close()

	// data for creating users
	reqs := []AddUserRequest{
		{Email: "admin@admin.com", Password: "admin123"},
		{Email: "someone@admin.com", Password: "admin123"},
		{Email: "someone@somewhere.com", Password: "somethingElse"},
	}
	failed_reqs := []AddUserRequest{
		{Email: "admin@admin.com", Password: "admin123"},
		{Email: "short@admin.com", Password: "admin"},
	}

	// create users
	for _, req := range reqs {
		err := AddUser(db, req)
		if err != nil {
			t.Fatalf("User Creation Failed: %v", err)
			return
		}
	}
	for _, req := range failed_reqs {
		err := AddUser(db, req)
		if err == nil {
			t.Fatalf("Invalid Creation did not fail: %v", req)
			return
		}
	}

	expectedUsers := map[int]User{
		0: {Email: "admin@admin.com"},
		1: {Email: "someone@admin.com"},
		2: {Email: "someone@somewhere.com"},
	}

	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		users := GetAllUsers(tx)
		for index, user := range users {
			expectedUser := expectedUsers[index]
			if expectedUser.Email != user.Email {
				t.Fatalf("emails don't match: %s, %s", expectedUser.Email, user.Email)
			}
		}
	})
}
