package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
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
	// RunTests()

	// writeIndex([]IndexEntry{IndexEntry{}, IndexEntry{}, IndexEntry{}, IndexEntry{}, IndexEntry{}})
	// fmt.Println(readIndex())
	// s, _ := listFiles(".")
	// fmt.Println(s)
	// getStatus()

	// // Create a test file
	// testContent := "Hello, Git!\n"
	// os.WriteFile("test.txt", []byte(testContent), 0644)

	// Calculate hash
	name := "sample.go"
	sha, err := gitBlobHash(name)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	data, _ := os.ReadFile(name)
	mySha := hashData(byteObject(data, "blob"))

	fmt.Printf("Calculated SHA: %s\n", sha)
	fmt.Printf("My SHA:         %s\n", hex.EncodeToString(mySha))
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

// Return full data of file data
// Combines header (obj_type data_len) + 0 + data_from_file
func byteObject(data []byte, obj_type string) []byte {
	header := []byte(fmt.Sprintf("%s %d\x00", obj_type, len(data)))
	fullData := append(header, data...)
	// fmt.Println(header, data)
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

func writeObject(path string, data []byte, obj_type string) {
	fullData := byteObject(data, obj_type)
	hashData := fmt.Sprintf("%x", sha1.Sum(fullData))
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

func parseIndexEntry(data []byte) (IndexEntry, int, error) {
	if len(data) < 63 {
		return IndexEntry{}, 0, fmt.Errorf("data too short")
	}

	entry := IndexEntry{}

	entry.CtimeSeconds = binary.BigEndian.Uint32(data[0:4])
	entry.CtimeNanoseconds = binary.BigEndian.Uint32(data[4:8])
	entry.MtimeSeconds = binary.BigEndian.Uint32(data[8:12])
	entry.MtimeNanoseconds = binary.BigEndian.Uint32(data[12:16])
	entry.Dev = binary.BigEndian.Uint32(data[16:20])
	entry.Ino = binary.BigEndian.Uint32(data[20:24])
	entry.Mode = binary.BigEndian.Uint32(data[24:28])
	entry.Uid = binary.BigEndian.Uint32(data[28:32])
	entry.Gid = binary.BigEndian.Uint32(data[32:36])
	entry.FileSize = binary.BigEndian.Uint32(data[36:40])
	copy(entry.Sha[:], data[40:60])
	entry.Flags = binary.BigEndian.Uint16(data[60:62])

	nullIndex := -1
	for i := 62; i < len(data); i++ {
		if data[i] == byte(0) {
			nullIndex = i
			break
		}
	}

	if nullIndex == -1 {
		return IndexEntry{}, 0, fmt.Errorf("null terminator not found")
	}
	entry.Path = string(data[62:nullIndex])

	return entry, nullIndex + (8 - nullIndex%8), nil
}

func writeIndex(entries []IndexEntry) {
	buf := new(bytes.Buffer)
	//header
	binary.Write(buf, binary.BigEndian, []byte("DIRC"))
	binary.Write(buf, binary.BigEndian, uint32(2))
	binary.Write(buf, binary.BigEndian, uint32(len(entries)))
	//main data
	for _, en := range entries {
		binary.Write(buf, binary.BigEndian, en.bytes())
	}
	//sha1
	binary.Write(buf, binary.BigEndian, hashData(buf.Bytes()))
	err := os.WriteFile(path.Join(".git", "index"), buf.Bytes(), os.FileMode(0755))
	check(err)
	fmt.Printf("Wrote to index %x\n", buf.Bytes())
}

func readIndex() ([]IndexEntry, error) {
	data, err := os.ReadFile(path.Join(".git", "index"))
	if err != nil {
		return nil, err
	}

	if !reflect.DeepEqual(hashData(data[:len(data)-20]), data[len(data)-20:]) {
		return nil, fmt.Errorf("Check sums aren't equal")
	}

	if string(data[:4]) != "DIRC" {
		return nil, fmt.Errorf("Sync word(%s) doesn't equal to DIRC", string(data[:4]))
	}

	i := 12
	entry, index, err := parseIndexEntry(data[i:])
	entries := []IndexEntry{}
	for err == nil {
		entries = append(entries, entry)
		i += index
		entry, index, err = parseIndexEntry(data[i:])
	}
	return entries, nil
}

func listFiles(root string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		for _, d := range strings.Split(path, string(filepath.Separator)) {
			if d == ".git" {
				return nil
			}
		}

		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return files, nil
}

// return changed_Paths, newPaths, deletedPaths
func getStatus() ([]string, []string, []string) {
	allFilepaths, _ := listFiles(".")
	paths := make(map[string]bool)
	for _, fp := range allFilepaths {
		paths[fp] = false
	}

	// var entriesByPath map[string]IndexEntry
	indexEntries, _ := readIndex()

	changedPaths := []string{}
	newPaths := []string{}
	deletedPaths := []string{}
	for _, entry := range indexEntries {
		_, ok := paths[entry.Path]

		if !ok {
			deletedPaths = append(deletedPaths, entry.Path)
			fmt.Println("deleted : ", entry.Path)
		} else {
			paths[entry.Path] = true

			data, _ := os.ReadFile(entry.Path)

			if entry.Sha != [20]byte(hashData(byteObject(data, "blob"))) {
				// fmt.Println(entry.Path)
				// fmt.Println("index:   ", hex.EncodeToString(entry.Sha[:]))
				// fmt.Println("new cal: ", hex.EncodeToString(hashData(byteObject(data, "blob"))))

				DebugSHA(entry)
				fmt.Printf("mySHA (hex): %x\n", [20]byte(hashData(byteObject(data, "blob"))))
				s, _ := gitBlobHash(entry.Path)
				fmt.Printf("gptSHA (hex): %s\n", s)
				break
			}
		}
	}

	return changedPaths, newPaths, deletedPaths
}

func DebugSHA(entry IndexEntry) {
	fmt.Printf("Path: %s\n", entry.Path)
	fmt.Printf("SHA (hex): %x\n", entry.Sha)
	fmt.Printf("SHA (string): %s\n", hex.EncodeToString(entry.Sha[:]))

	// Verify this matches git's output
	fmt.Printf("Compare with: git ls-files --stage | grep %s\n", entry.Path)
}

func gitBlobHash(filename string) (string, error) {
	// 1. Read file as binary
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}

	// 2. Create Git blob header
	// Format: "blob " + decimal_size + "\0"
	header := fmt.Sprintf("blob %d\x00", len(content))

	// 3. Compute SHA1 of header + content
	hash := sha1.New()
	hash.Write([]byte(header))
	hash.Write(content)

	return hex.EncodeToString(hash.Sum(nil)), nil
}
