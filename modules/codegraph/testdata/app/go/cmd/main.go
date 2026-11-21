package main

import (
	"fmt"

	"example.com/app/internal/repo"
)

func main() {
	r := repo.New()
	if err := r.Persist(); err != nil {
		fmt.Println(err)
	}
	run()
}

func run() {}
