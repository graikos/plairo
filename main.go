package main

import (
	"fmt"
	"plairo/db"
)

func main() {
	fmt.Println("Initial main.")
	fmt.Println([]byte("obfuscate_key"))
	fmt.Printf("%x\n", db.ConstructObfKeyKey())
}
