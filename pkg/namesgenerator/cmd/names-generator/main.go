package main

import (
	"fmt"

	"github.com/tiborvass/docker/pkg/namesgenerator"
)

func main() {
	fmt.Println(namesgenerator.GetRandomName(0))
}
