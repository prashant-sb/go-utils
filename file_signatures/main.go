package main

// Tool iterates all files in given directory abd calculates
// given checksum.
//

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

var (
	dest = flag.String("dest", "/tmp", "root direcory for calculate file hashes")
	sign = flag.String("sign", "md5", "Hashing algorithm")
)

func checksumWorker(filePath string) error {
	var filehash func(filePath string) (string, error)

	switch *sign {

	case "crc":
		filehash = FileCrc32

	case "md5":
		filehash = FileMd5Sum

	case "sha256":
		filehash = FileSha256

	default:
		err := errors.New("Algorithm not supported.")
		return err
	}

	cs, err := filehash(filePath)
	if err != nil {
		fmt.Printf("Error: %v", err)
		return err
	}

	fmt.Printf("%s :: %s\n", filePath, cs)
	return nil
}

func walkWith(path string, info os.FileInfo, err error) error {
	if info.IsDir() {
		return nil
	}
	go checksumWorker(path)

	return nil
}

func main() {
	flag.Parse()

	err := filepath.Walk(*dest, walkWith)
	if err != nil {
		fmt.Printf("Error : %s", err.Error())
		return
	}
}
