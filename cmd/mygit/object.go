package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

func readObject(object string) ([]byte, error) {
	dir, file := splitDirFile(object)
	b, err := os.ReadFile(filepath.Join(".git/objects", dir, file))
	if err != nil {
		return nil, fmt.Errorf("error on reading object file: %v", err)
	}

	objectData, err := unzip(b)
	if err != nil {
		return nil, fmt.Errorf("error on unzipping object file: %v", err)
	}

	return objectData, nil
}

func writeObject(objectData []byte) ([]byte, error) {
	zippedData, err := zip(objectData)
	if err != nil {
		return nil, fmt.Errorf("error on zipping blob object: %v", err)
	}

	object := hash(objectData)
	dir, file := splitDirFile(hexDump(object))
	if err := os.Mkdir(filepath.Join(".git/objects", dir), 0644); err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("error on creating object dir: %v", err)
	}
	f, err := os.OpenFile(filepath.Join(".git/objects", dir, file), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("error on opening object file: %v", err)
	}
	defer f.Close()

	_, err = io.CopyBuffer(f, bytes.NewReader(zippedData), make([]byte, 512))
	if err != nil {
		return nil, fmt.Errorf("error on writing object file: %v", err)
	}

	return object, nil
}

func readBlobContent(object string) (string, error) {
	blob, err := readObject(object)
	if err != nil {
		return "", err
	}

	content, err := parseBlobContent(blob)
	if err != nil {
		return "", fmt.Errorf("error on extracting blob file: %v", err)
	}

	return content, nil
}

func writeBlob(filePath string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("error on reading file: %v", err)
		os.Exit(1)
	}

	blob := blobObject(content)
	return writeObject(blob)
}

func readTree(object string) ([]*TreeEntry, error) {
	treeData, err := readObject(object)
	if err != nil {
		return nil, err
	}

	return parseTree(treeData)
}

func writeTree(dir string) ([]byte, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	treeEntires := make([]*TreeEntry, 0, len(dirEntries))

	for _, dirEntry := range dirEntries {
		if filepath.Join(dir, dirEntry.Name()) == ".git" || !dirEntry.IsDir() {
			continue
		}

		object, err := writeTree(filepath.Join(dir, dirEntry.Name()))
		if err != nil {
			return nil, err
		}
		treeEntires = append(treeEntires, &TreeEntry{
			name: dirEntry.Name(),
			hash: object,
			mode: TreeEntryModeTree,
		})
	}

	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}

		object, err := writeBlob(filepath.Join(dir, dirEntry.Name()))
		if err != nil {
			return nil, err
		}
		treeEntires = append(treeEntires, &TreeEntry{
			name: dirEntry.Name(),
			hash: object,
			mode: TreeEntryModeBlob,
		})
	}

	sort.SliceStable(treeEntires, func(i, j int) bool {
		return treeEntires[i].name < treeEntires[j].name
	})

	buf := make([]byte, 0)
	for _, entry := range treeEntires {
		buf = append(buf, entry.Bytes()...)
	}

	header := fmt.Sprintf("tree %d\x00", len(buf))

	treeData := append([]byte(header), buf...)
	return writeObject(treeData)
}

func writeCommit(commit *Commit) ([]byte, error) {
	objectData := commit.Bytes()
	return writeObject(objectData)
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

func zip(b []byte) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	w := zlib.NewWriter(buf)
	_, err := w.Write(b)
	if err != nil {
		return nil, err
	}

	w.Close()
	return buf.Bytes(), nil
}

func parseBlobContent(b []byte) (string, error) {
	// blob <size>\x00<content>
	i := bytes.IndexByte(b, 0)
	if i < 0 {
		return "", errors.New("cannot extract blob content")
	}

	return string(b[i+1:]), nil
}

func blobObject(b []byte) []byte {
	header := fmt.Sprintf("blob %d\x00", len(b))
	return append([]byte(header), b...)
}

type TreeEntry struct {
	hash []byte
	name string
	mode string
}

func (t *TreeEntry) Bytes() []byte {
	buf := bytes.NewBuffer(nil)
	buf.Write([]byte(t.mode))
	buf.Write([]byte{'\x20'})
	buf.Write([]byte(t.name))
	buf.Write([]byte{'\x00'})
	buf.Write([]byte(t.hash))
	return buf.Bytes()
}

const (
	TreeEntryModeBlob = "100644"
	TreeEntryModeTree = "40000"
)

func parseTree(b []byte) ([]*TreeEntry, error) {
	offset := bytes.IndexByte(b, 0) + 1
	entries := make([]*TreeEntry, 0)
	for offset < len(b) {
		entry, skipN, err := parseTreeEntry(b[offset:])
		if err != nil {
			return nil, fmt.Errorf("cannot parse tree: %v", err)
		}
		entries = append(entries, entry)
		offset += skipN
	}
	return entries, nil
}

func parseTreeEntry(b []byte) (*TreeEntry, int, error) {
	i := bytes.IndexByte(b, 0)
	if i < 0 {
		return nil, 0, errors.New("cannot parse tree entry")
	}
	mode, name, found := bytes.Cut(b[:i], []byte(" "))
	if !found {
		return nil, 0, errors.New("cannot parse tree entry")
	}
	hash := b[i+1 : i+21]
	entry := &TreeEntry{
		hash: hash,
		name: string(name),
		mode: string(mode),
	}
	return entry, i + 21, nil
}

type Commit struct {
	treeObject   string
	parentObject string
	author       User
	committer    User
	message      string
}

func (c *Commit) Bytes() []byte {
	buf := bytes.NewBuffer(nil)
	buf.WriteString(fmt.Sprintf("tree %s\n", c.treeObject))
	if c.parentObject != "" {
		buf.WriteString(fmt.Sprintf("parent %s\n", c.parentObject))
	}
	buf.WriteString(fmt.Sprintf("author %s %s\n", c.author.name, c.author.email))
	buf.WriteString(fmt.Sprintf("committer %s %s\n", c.committer.name, c.committer.email))
	buf.WriteString(fmt.Sprintf("\n%s\n", c.message))

	header := fmt.Sprintf("commit %d\x00", buf.Len())

	return append([]byte(header), buf.Bytes()...)
}

type User struct {
	name  string
	email string
}

func hash(b []byte) []byte {
	h := sha1.Sum(b)
	return h[:]
}

func hexDump(b []byte) string {
	return fmt.Sprintf("%x", b)
}

func splitDirFile(hex string) (string, string) {
	return hex[:2], hex[2:]
}
