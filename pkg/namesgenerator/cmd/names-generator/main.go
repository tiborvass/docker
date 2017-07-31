package main

import (
	"fmt"

	"github.com/moby/moby-core/pkg/namesgenerator"
)

func main() {
	fmt.Println(namesgenerator.GetRandomName(0))
}
