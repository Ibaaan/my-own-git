package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {

	// cmd_init("test")
	// h := sha1.New()
	// io.WriteString(h, "His money is twice tainted:")
	// // io.WriteString(h, " 	")
	// fmt.Printf("% x", h.Sum(nil))

	// fmt.Printf("%x\n", zlibCommpress([]byte("Hello World")))
	// v := zlibCommpress([]byte("Hello World"))
	// fmt.Println(string(zlibDecommpress(v)))
	// cmdInit("test")
	writeData("test", []byte("Hello World"), "blob")
	// fmt.Println(byteObject([]byte("Hello World"), "blob"))
	//"test/.git/objects/5e/1c309dae7f45e0f39b1bf3ac3cd9db12e7d689"
	readData("test/.git/objects/5e/1c309dae7f45e0f39b1bf3ac3cd9db12e7d689")
}

func cmdInit(path string) {
	for _, name := range []string{"objects", "refs", "refs/heads"} {
		fmt.Println(filepath.Join(path, ".git", name))
		os.MkdirAll(filepath.Join(path, ".git", name), os.FileMode(0755))
	}
	err := os.WriteFile(filepath.Join(path, ".git", "HEAD"),
		[]byte("ref: refs/heads/master"), os.FileMode(0755))
	check(err)
}

func hashData(data []byte) []byte {
	h := sha1.New()
	h.Write(data)
	return h.Sum(nil)
}

func byteObject(data []byte, obj_type string) []byte {
	header := []byte(fmt.Sprintf("%s %d", obj_type, len(data)))
	fullData := append(header, []byte{0x00}...)
	fullData = append(fullData, data...)
	fmt.Println(header, data)
	return fullData
}

func zlibCommpress(data []byte) []byte {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write(data)
	w.Close()
	return b.Bytes()
}

func zlibDecommpress(data []byte) []byte {
	var b bytes.Buffer
	b.Write(data)

	zr, err := zlib.NewReader(&b)
	check(err)
	zr.Close()

	decompressed, err := io.ReadAll(zr)
	// fmt.Println("Real Bytes  ", decompressed)
	return decompressed
}

func writeData(path string, data []byte, obj_type string) {
	fullData := byteObject(data, obj_type)
	hashData := fmt.Sprintf("%x", hashData(fullData))
	filePath := filepath.Join(path, ".git", "objects", hashData[:2])
	err := os.MkdirAll(filePath, os.FileMode(0755))
	check(err)
	filePath = filepath.Join(filePath, hashData[2:])
	err = os.WriteFile(filePath, zlibCommpress(fullData), os.FileMode(0755))
	check(err)
}

func readData(path string) (string, []byte) {
	fileData, _ := os.ReadFile(path)
	decompressedData := zlibDecommpress(fileData)

	idx := bytes.IndexByte(decompressedData, 0x00)
	header := decompressedData[:idx]
	mainData := decompressedData[idx+1:]

	tmp := strings.Split(string(header), " ")
	objType := tmp[0]
	dataLen := tmp[1]
	if i, err := strconv.Atoi(dataLen); i != len(mainData) {
		panic(err)
	}
	return objType, mainData
}
