package main

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: mygit <command> [<args>...]\n")
		os.Exit(1)
	}

	switch command := os.Args[1]; command {
	case "init":
		for _, dir := range []string{".git", ".git/objects", ".git/refs"} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
			}
		}

		headFileContents := []byte("ref: refs/heads/main\n")
		if err := os.WriteFile(".git/HEAD", headFileContents, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %s\n", err)
		}

		fmt.Println("Initialized git directory")
	case "cat-file": // git cat-file -p <object-SHA1>
		if len(os.Args) != 4 {
			fmt.Println("git cat-file -p <object-SHA1>")
			os.Exit(1)
		}

		flag := os.Args[2]

		if flag != "-p" {
			fmt.Println("only -p flag is supported")
			os.Exit(1)
		}

		objectHash := os.Args[3]
		if len(objectHash) != 40 {
			fmt.Println("len of object hash must be 40")
			os.Exit(1)
		}

		dir := objectHash[0:2]
		file := objectHash[2:]

		b, err := os.ReadFile(filepath.Join(".git/objects", dir, file))
		if err != nil {
			fmt.Printf("error on reading object file: %v", err)
			os.Exit(1)
		}

		blob, err := unzip(b)
		if err != nil {
			fmt.Printf("error on unzipping object file: %v", err)
			os.Exit(1)
		}

		content, err := parseBlobContent(blob)
		if err != nil {
			fmt.Printf("error on extracting blob file: %v", err)
			os.Exit(1)
		}

		fmt.Print(content)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
		os.Exit(1)
	}
}

func unzip(b []byte) ([]byte, error) {
	r := bytes.NewReader(b)
	zr, err := zlib.NewReader(r)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(nil)
	_, err = io.CopyBuffer(buf, zr, make([]byte, 512))
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func parseBlobContent(b []byte) (string, error) {
	i := bytes.IndexByte(b, 0)
	if i < 0 {
		return "", errors.New("cannot extract blob content")
	}

	return string(b[i+1:]), nil
}
