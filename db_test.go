package fstack

import (
	"fmt"
	"testing"
	"time"
)

func TestSimpleDB(t *testing.T) {
	var (
		header = []byte("headers")
		data   = []byte("body of simple message")
	)
	db, err := NewDatabase("./test-data/db", 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for i := 0; i < 100; i++ {
		_, err = db.Get(fmt.Sprintf("system %v #1111", i)).Push(header, data)
		if err != nil {
			t.Fatal(err)
		}
	}

	err = db.Clean()
	if err != nil {
		t.Fatal(err)
	}
}
