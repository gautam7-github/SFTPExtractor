package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type remoteFiles struct {
	Name    string
	Size    string
	ModTime string
}

func listFiles(sc sftp.Client, remoteDir string) (theFiles []remoteFiles, err error) {

	files, err := sc.ReadDir(remoteDir)
	if err != nil {
		return theFiles, fmt.Errorf("Unable to list remote dir: %v", err)
	}

	for _, f := range files {
		var name, modTime, size string

		name = f.Name()
		modTime = f.ModTime().Format("2006-01-02 15:04:05")
		size = fmt.Sprintf("%12d", f.Size())

		if f.IsDir() {
			name = name + "/"
			modTime = ""
			size = "PRE"
		}

		theFiles = append(theFiles, remoteFiles{
			Name:    name,
			Size:    size,
			ModTime: modTime,
		})
	}

	return theFiles, nil
}

func downloadFile(sc sftp.Client, remoteFile, localFile string) (err error) {

	log.Printf("Downloading [%s] to [%s] ...\n", remoteFile, localFile)
	// Note: SFTP To Go doesn't support O_RDWR mode
	srcFile, err := sc.OpenFile(remoteFile, (os.O_RDWR))
	if err != nil {
		return fmt.Errorf("unable to open remote file: %v", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(localFile)
	if err != nil {
		return fmt.Errorf("unable to open local file: %v", err)
	}
	defer dstFile.Close()

	bytes, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("unable to download remote file: %v", err)
	}
	log.Printf("%d bytes copied to %v", bytes, dstFile)

	return nil
}

func main() {

	host := ""
	username := ""
	password := ""

	port := ""

	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		// `InsecureIgnoreHostKey` is not recommended. Consider other more secured method for verifying host key
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Establishing connection
	conn, err := ssh.Dial("tcp", host+port, config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to dial: %v\n", err)
		os.Exit(1)
	}

	// `client` is the handler for performing operations on SFTP server
	client, err := sftp.NewClient(conn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create client: %v\n", err)
		os.Exit(1)
	}

	// Example for checking available storage space
	statVF, err := client.StatVFS("/")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fail to get filesystem info: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Space available: %d bytes\n", statVF.FreeSpace())
	// List files in the root directory .
	theFiles, err := listFiles(*client, "./")
	if err != nil {
		log.Fatalf("failed to list files in .: %v", err)
	}

	log.Printf("Found Files -")
	// Output each file name and size in bytes
	log.Printf("%19s %12s %s", "MOD TIME", "SIZE", "NAME")
	for _, theFile := range theFiles {
		log.Printf("%19s %12s %s", theFile.ModTime, theFile.Size, theFile.Name)
		err = downloadFile(*client, "./"+theFile.Name, theFile.Name)
		if err != nil {
			log.Fatalf("Could not download file; %v", err)
		}
	}
}
