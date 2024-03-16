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

func writeBlob(content []byte) (string, error) {
	blob := blobObject(content)
	hash := hashHex(blob)
	zippedBlob, err := zip(blob)
	if err != nil {
		return "", fmt.Errorf("error on zipping blob object: %v", err)
	}

	dir, file := splitDirFile(hash)
	if err := os.Mkdir(filepath.Join(".git/objects", dir), 0644); err != nil && !os.IsExist(err) {
		return "", fmt.Errorf("error on creating object dir: %v", err)
	}
	f, err := os.OpenFile(filepath.Join(".git/objects", dir, file), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return "", fmt.Errorf("error on opening object file: %v", err)
	}
	defer f.Close()

	_, err = io.CopyBuffer(f, bytes.NewReader(zippedBlob), make([]byte, 512))
	if err != nil {
		return "", fmt.Errorf("error on writing object file: %v", err)
	}

	return hash, nil
}

func readTree(object string) ([]*TreeEntry, error) {
	treeData, err := readObject(object)
	if err != nil {
		return nil, err
	}

	return parseTree(treeData)
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
	header := fmt.Sprintf("blob %d", len(b))
	headerBytes := append([]byte(header), 0)
	return append(headerBytes, b...)
}

type TreeEntry struct {
	hash []byte
	name string
	mode string
}

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

func hashHex(b []byte) string {
	h := sha1.Sum(b)
	// hex dump
	return fmt.Sprintf("%x", h)
}

func splitDirFile(hex string) (string, string) {
	return hex[:2], hex[2:]
}
