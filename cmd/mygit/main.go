package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: mygit <command> [<args>...]\n")
		os.Exit(1)
	}

	switch command := os.Args[1]; command {
	case "init":
		err := Init()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
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

		content, err := readBlobContent(objectHash)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Print(content)
	case "hash-object":
		if len(os.Args) != 4 {
			fmt.Println("git hash-object -w <filename>")
			os.Exit(1)
		}

		flag := os.Args[2]
		if flag != "-w" {
			fmt.Println("only -w flag is supported")
			os.Exit(1)
		}

		filePath := os.Args[3]

		object, err := writeBlob(filePath)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Print(hexDump(object))
	case "ls-tree":
		if len(os.Args) != 4 {
			fmt.Println("git ls-tree --name-only <tree-SHA1>")
			os.Exit(1)
		}

		flag := os.Args[2]
		if flag != "--name-only" {
			fmt.Println("--name-only flag is required")
			os.Exit(1)
		}

		objectHash := os.Args[3]
		if len(objectHash) != 40 {
			fmt.Println("len of object hash must be 40")
			os.Exit(1)
		}

		entries, err := readTree(objectHash)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		for _, entry := range entries {
			fmt.Println(entry.name)
		}
	case "write-tree":
		if len(os.Args) != 2 {
			fmt.Println("git write-tree")
			os.Exit(1)
		}

		object, err := writeTree(".")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Print(hexDump(object))
	case "commit-tree":
		if len(os.Args) < 3 {
			fmt.Println("git commit-tree <tree-SHA1> -p <commit-SHA1> -m <message>")
			os.Exit(1)
		}

		parent := ""
		message := ""

		i := 3
		for i < len(os.Args)-1 {
			flag := os.Args[i]
			switch flag {
			case "-p":
				parent = os.Args[i+1]
			case "-m":
				message = os.Args[i+1]
			}
			i += 2
		}

		commit := Commit{
			treeObject:   os.Args[2],
			parentObject: parent,
			author: User{
				name:  "Scott Chacon",
				email: "schacon@gmail.com",
			},
			committer: User{
				name:  "Scott Chacon",
				email: "schacon@gmail.com",
			},
			message: message,
		}

		object, err := writeCommit(&commit)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Print(hexDump(object))
	case "clone":
		if len(os.Args) != 4 {
			fmt.Println("git clone <git-remote-repo> <dir>")
			os.Exit(1)
		}

		remoteRepo := os.Args[2]
		dir := os.Args[3]

		err := Clone(remoteRepo, dir)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
		os.Exit(1)
	}
}
