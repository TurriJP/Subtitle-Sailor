package main

import (
	"fmt"
	"os"
	"github.com/opensubtitlescli/moviehash"
)

func GetHash(filename string) string{
	f, _ := os.Open(filename)
	h, _ := moviehash.Sum(f)
	fmt.Println(h)
	return h
}