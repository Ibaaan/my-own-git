package main

import "fmt"

func TestIndexEntryBytes() {
	entry := &IndexEntry{
		CtimeSeconds: 1600000000,
		MtimeSeconds: 1600000000,
		Mode:         0x0000A494, // 100644 in octal
		FileSize:     1024,
		Sha:          [20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		Path:         "README.md",
	}

	data := entry.bytes()

	fmt.Printf("Total size: %d bytes\n", len(data))
	fmt.Printf("Fixed part: %d bytes\n", 62)
	fmt.Printf("Path + null: %d bytes\n", len(entry.Path)+1)
	fmt.Printf("Padding: %d bytes\n", len(data)-62-len(entry.Path)-1)

	// Verify structure
	if data[62+len(entry.Path)] != 0 {
		panic("Missing null terminator!")
	}

	if len(data)%8 != 0 {
		panic("Not 8-byte aligned!")
	}

	fmt.Println("✓ Entry bytes are correct")
}
