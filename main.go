package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	testWhole()

}

func cmdInit(path string) {
	for _, name := range []string{"objects", "refs", "refs/heads"} {
		// fmt.Println(filepath.Join(path, ".git", name))
		os.MkdirAll(filepath.Join(path, ".git", name), os.FileMode(0755))
	}
	err := os.WriteFile(filepath.Join(path, ".git", "HEAD"),
		[]byte("ref: refs/heads/master"), os.FileMode(0755))
	check(err)
	writeIndex([]IndexEntry{})
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
	// fmt.Printf("Wrote to index %x\n", buf.Bytes())
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

func getFiles(root string) ([]string, error) {
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
	allFilepaths, _ := getFiles(".")
	paths := make(map[string]bool)
	for _, fp := range allFilepaths {
		paths[fp] = false
	}

	indexEntries, _ := readIndex()

	changedPaths := []string{}
	newPaths := []string{}
	deletedPaths := []string{}
	for _, entry := range indexEntries {
		_, ok := paths[entry.Path]

		if !ok {
			deletedPaths = append(deletedPaths, entry.Path)
		} else {
			paths[entry.Path] = true

			data, _ := os.ReadFile(entry.Path)

			if entry.Sha != [20]byte(hashData(byteObject(data, "blob"))) {
				changedPaths = append(changedPaths, entry.Path)
			}
		}
	}

	for path, v := range paths {
		if !v {
			newPaths = append(newPaths, path)
		}
	}

	return changedPaths, newPaths, deletedPaths
}

// print files
// details=true - all info file mode, object type, hash, and filename
// details=false - only filenames
func lsFiles(details bool) {
	entries, err := readIndex()
	check(err)

	for _, entry := range entries {
		if details {
			stage := (entry.Flags >> 12) & 3
			fmt.Printf("%06o %x %d\t%s\n",
				entry.Mode, entry.Sha, stage, entry.Path)
		} else {
			fmt.Println(entry.Path)
		}
	}
}

func status() {
	ch, new, del := getStatus()
	fmt.Println("changed: ", ch)
	fmt.Println("new:     ", new)
	fmt.Println("del:     ", del)
}

func add(paths []string) {
	allEntries, _ := readIndex()

	entries := []IndexEntry{}
	for _, entry := range allEntries {
		shouldKeep := true
		for _, path := range paths {
			if entry.Path == path {
				shouldKeep = false
				break
			}
		}
		if shouldKeep {
			entries = append(entries, entry)
		}
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		check(err)

		fullData := byteObject(data, "blob")
		hashData := sha1.Sum(fullData)

		info, err := os.Stat(path)
		check(err)

		flags := uint16(len(path))
		if flags >= (1 << 12) {
			panic(fmt.Sprintf("path too long: %s", path))
		}

		entry := IndexEntry{
			CtimeSeconds:     0,
			CtimeNanoseconds: 0,
			MtimeSeconds:     uint32(info.ModTime().Unix()),
			MtimeNanoseconds: 0,
			FileSize:         uint32(info.Size()),
			Mode:             uint32(info.Mode()),
			Sha:              hashData,
			Flags:            flags,
			Path:             path,
		}

		entries = append(entries, entry)

		writeObject(".", data, "blob")
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})

	writeIndex(entries)
}

func writeTree() []byte {
	entries, err := readIndex()
	check(err)

	treeEntries := make([][]byte, 0)

	for _, entry := range entries {
		if strings.Contains(entry.Path, "/") {
			panic("currently only supports a single, top-level directory")
		}

		modePath := fmt.Sprintf("%o %s", entry.Mode, entry.Path)
		treeEntry := append([]byte(modePath), 0)
		treeEntry = append(treeEntry, entry.Sha[:]...)

		treeEntries = append(treeEntries, treeEntry)
	}

	sort.Slice(treeEntries, func(i, j int) bool {
		iPath := string(bytes.SplitN(treeEntries[i], []byte{0}, 2)[0])
		jPath := string(bytes.SplitN(treeEntries[j], []byte{0}, 2)[0])

		iParts := strings.SplitN(iPath, " ", 2)
		jParts := strings.SplitN(jPath, " ", 2)

		if len(iParts) > 1 && len(jParts) > 1 {
			return iParts[1] < jParts[1]
		}
		return iPath < jPath
	})

	var combined []byte
	for _, entry := range treeEntries {
		combined = append(combined, entry...)
	}

	writeObject(".", combined, "tree")

	fullData := byteObject(combined, "tree")
	hashData := sha1.Sum(fullData)

	return hashData[:]
}
