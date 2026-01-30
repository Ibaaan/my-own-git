package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/binary"
	"errors"
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
	// cmdInit("test")
	// writeData("test", []byte("Hello World"), "blob")
	// fmt.Println(byteObject([]byte("Hello World"), "blob"))
	//"test/.git/objects/5e/1c309dae7f45e0f39b1bf3ac3cd9db12e7d689"
	// s, b := readData("test/.git/objects/5e/1c309dae7f45e0f39b1bf3ac3cd9db12e7d689")
	// fmt.Println(findObject("5e"))
	TestIndexEntryBytes()
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

func readData(path string) ([]byte, string) {
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
	return mainData, objType
}

func findObject(sha1Prefix string) (string, error) {
	if len(sha1Prefix) < 2 {
		return "", errors.New("sha prefix should be 2 or longer")
	}

	objDir := filepath.Join(".git", "objects", sha1Prefix[:2])
	files, err := os.ReadDir(objDir)
	if err != nil {
		return "", err
	}

	if len(files) != 1 {
		return "", fmt.Errorf("there are (%d) obj's with prefix %s, should be only 1",
			len(files), sha1Prefix)
	}
	return filepath.Join(objDir, files[0].Name()), nil
}

func readObject(sha1Prefix string) ([]byte, string) {
	path, err := findObject(sha1Prefix)
	check(err)
	return readData(path)
}

func catFile(mode string, sha1Prefix string) {
	// Someday will print smth
}

type IndexEntry struct {
	CtimeSeconds     uint32
	CtimeNanoseconds uint32
	MtimeSeconds     uint32
	MtimeNanoseconds uint32
	Dev              uint32
	Ino              uint32
	Mode             uint32
	Uid              uint32
	Gid              uint32
	FileSize         uint32
	Sha              [20]byte
	Flags            uint16
	Path             string
}

func (e *IndexEntry) bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, e.CtimeSeconds)
	binary.Write(buf, binary.BigEndian, e.CtimeNanoseconds)
	binary.Write(buf, binary.BigEndian, e.MtimeSeconds)
	binary.Write(buf, binary.BigEndian, e.MtimeNanoseconds)
	binary.Write(buf, binary.BigEndian, e.Dev)
	binary.Write(buf, binary.BigEndian, e.Ino)
	binary.Write(buf, binary.BigEndian, e.Mode)
	binary.Write(buf, binary.BigEndian, e.Uid)
	binary.Write(buf, binary.BigEndian, e.Gid)
	binary.Write(buf, binary.BigEndian, e.FileSize)
	binary.Write(buf, binary.BigEndian, e.Sha)
	binary.Write(buf, binary.BigEndian, e.Flags)
	binary.Write(buf, binary.BigEndian, []byte(e.Path))
	binary.Write(buf, binary.BigEndian, byte(0))

	pos := 63 + len([]byte(e.Path))
	for pos%8 != 0 {
		binary.Write(buf, binary.BigEndian, byte(0))
		pos++
	}

	return buf.Bytes()
}

func byteToIndexEntry(byteIndex []byte) *IndexEntry {
	// TODO: implement from byte to index
	return &IndexEntry{}
}
