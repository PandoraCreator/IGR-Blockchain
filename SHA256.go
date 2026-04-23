// [_SHA256 hashes_](https://en.wikipedia.org/wiki/SHA-2) are
// frequently used to compute short identities for binary
// or text blobs. For example, TLS/SSL certificates use SHA256
// to compute a certificate's signature. Here's how to compute
// SHA256 hashes in Go.

package main


import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

func main() {
	path := "a.pdf"
	// Replace example.pdf with your PDF file path.
	hash, err := HashPDF(path)
	if err != nil {
		fmt.Printf("failed to hash PDF: %v\n", err)
		return
	}

	fmt.Printf("PDF hash for %s: %s\n", path, hash)
}

// HashPDF computes the SHA256 hash of a PDF file's contents.
// It returns the hash as a lowercase hex string.
func HashPDF(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	// defining a new SHA256 hash object and copying the file's contents into it
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}


func writeHashToDB(hash string) error {
	// Placeholder for database connection and insertion logic
	// For example, you could use a SQL database and execute an INSERT statement here
	fmt.Printf("Writing hash to database: %s\n", hash)
	return nil	


}
