package main

import (
	"fmt"
	"os"
)

func testWhole() {
	fmt.Println("Init a repo")
	cmdInit(".")

	fmt.Println("Creating a.txt | Hello World! > a.txt")
	os.WriteFile("a.txt", []byte("Hello World!\n"), os.FileMode(0755))
	printIndexAndStatus()

	fmt.Println("Adding a.txt to index")
	add([]string{"a.txt"})
	printIndexAndStatus()

	fmt.Println("Changing a.txt | Not hello > a.txt")
	os.WriteFile("a.txt", []byte("Not hello\n"), os.FileMode(0755))
	printIndexAndStatus()

	fmt.Println("Adding a.txt to index")
	add([]string{"a.txt"})
	printIndexAndStatus()
}

func printIndexAndStatus() {
	fmt.Println("\n\n")
	fmt.Println("Index:")
	lsFiles(true)
	fmt.Println("\nStatus:")
	status()
	fmt.Println("\n\n")
}
