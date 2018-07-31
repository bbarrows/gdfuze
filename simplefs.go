// Copyright 2016 the Go-FUSE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// A Go mirror of libfuse's hello.c\

// Base Fuze FS From:
// https://github.com/hanwen/go-fuse.

// Using example Google Drive client code

package main

import (
	"syscall"
	"os/signal"
	"flag"
	"log"
	"fmt"
	"net/http"
	"os"	
	ioutil "io/ioutil"
	"encoding/json"
	fuse "github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
    "golang.org/x/net/context"
    oauth2 "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"	
	drive "google.golang.org/api/drive/v3"
)

type HelloFs struct {
	pathfs.FileSystem
}


var srv *drive.Service


var fileIDs map[string]string


// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
        tokenFile := "token.json"
        tok, err := tokenFromFile(tokenFile)
        if err != nil {
        	fmt.Println("No valid token.json file found");
        	os.Exit(1);
        }
        return config.Client(context.Background(), tok)
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
        f, err := os.Open(file)
        defer f.Close()
        if err != nil {
                return nil, err
        }
        tok := &oauth2.Token{}
        err = json.NewDecoder(f).Decode(tok)
        return tok, err
}











// For DirEntry I have three choices:
// S_IFDIR, S_IFREG, S_IFLNK
//
// I can OR permissions:
//(S_IFDIR | 0555)
//
//
// Scopes from:
// https://github.com/google/google-api-go-client/issues/218
//
// drive.DriveScope,
// drive.DriveReadonlyScope,
// drive.DriveAppdataScope,
// drive.DriveFileScope,
// drive.DriveMetadataScope,
// drive.DriveMetadataReadonlyScope,
// drive.DrivePhotosReadonlyScope,
//
// If modifying these scopes, delete your previously saved token.json.
//
func (me *HelloFs) GetAttr(name string, context *fuse.Context) (*fuse.Attr, fuse.Status) {
	switch name {
	case "file.txt":
		return &fuse.Attr{
			Mode: fuse.S_IFREG | 0644, Size: uint64(len(name)),
		}, fuse.OK
	case "":
		return &fuse.Attr{
			Mode: fuse.S_IFDIR | 0755,
		}, fuse.OK
	}
	return nil, fuse.ENOENT
}

func (me *HelloFs) OpenDir(name string, context *fuse.Context) (c []fuse.DirEntry, code fuse.Status) {
 	//uint64 inodeCounter = 0;

 	rootf, err := srv.Files.Get("root").Do()

 	fmt.Printf("Root File Dir %s", rootf.Id)
 	

 	fmt.Printf("Opening Dir %s", name)	

    r, err := srv.Files.List().PageSize(10).
            Fields("nextPageToken, files(id, name)").Do()
    if err != nil {
            log.Fatalf("Unable to retrieve files: %v", err)
    }

    var entries []fuse.DirEntry;

    fmt.Println("Files:")
    if len(r.Files) == 0 {
            fmt.Println("No files found.")
    } else {
            for c, i := range r.Files {
                    fmt.Printf("%s (%s)\n", i.Name, i.Id)
                    fmt.Printf("Count var c is: %d\n\n", c)

                    fileIDs[i.Name] = i.Id

                    var dirE = fuse.DirEntry{Name: i.Name, Mode: fuse.S_IFREG}
				    entries = append(entries, dirE)
				    //inodeCounter = inodeCounter + 1
            }
    }

    return entries, fuse.OK

	// if name == "" {
	// 	c = []fuse.DirEntry{{Name: "file.txt", Mode: fuse.S_IFREG}}
	// 	return c, fuse.OK
	// }
	// return nil, fuse.ENOENT
}

func (me *HelloFs) Open(name string, flags uint32, context *fuse.Context) (file nodefs.File, code fuse.Status) {
	
	// I need to get the file by the Id
	// https://github.com/google/google-api-go-client/blob/master/drive/v3/drive-gen.go#L4851
	// i.Id


	// To write a file:
	// https://gist.github.com/atotto/86fa30668473b41eeac7d750e5ad5f5c#file-google_drive_create_file_example-go-L126
	//
	// driveFile, err := srv.Files.Create(&drive.File{Name: filename}).Media(f).Do()
	//
	fmt.Printf("Opening file with name: %s\n", name)
	var fID = fileIDs[name]
	
	openFile, err := srv.Files.Get(fID).Fields("*").Do()
	if err != nil {
        // log.Fatalf("Unable to load file, Id: %s. Err: %v", fID, err)
		fmt.Printf("Unable to open %s, creating new file.", fID);
		if flags&fuse.O_ANYWRITE != 0 {
			return nil, fuse.EPERM
		}
		return nodefs.NewDataFile([]byte(name)), fuse.OK		
	}	

	var fileMime = openFile.MimeType;
	fmt.Printf("File mimeType: %s\n", fileMime)

	fileExportContent, err := srv.Files.Export(fID).FileMimeType("text/html").Do()
	if err != nil {
        log.Fatalf("Unable to export file, Id: %s. Err: %v", fID, err)
	}

	return &nodefs.WithFlags{
		File:      nodefs.NewDataFile([]byte(fileExportContent)),
		FuseFlags: fuse.FOPEN_DIRECT_IO,
	}, fuse.OK
	// Mime types to convert to:
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Basics_of_HTTP/MIME_types
	// APPLICATION: application/octet-stream, application/pkcs12, application/vnd.mspowerpoint, application/xhtml+xml, application/xml, application/pdf
	// TEXT: text/plain, text/html, text/css, text/javascript
	// IMAGE: image/gif, image/png, image/jpeg, image/bmp, image/webp
	// SUDIO: audio/midi, audio/mpeg, audio/webm, audio/ogg, audio/wav
	// VIDEO: video/webm, video/ogg


	// This returns a file resource
	// https://developers.google.com/drive/api/v3/reference/files#resource
	// Usefull looking attrs:
	// fileExtension
	// size
	// headRevisionId
	// fullFileExtension
	// originalFilename
	// name
	// description
	// kind 
	// id
	// mimeType
	//
	// https://github.com/google/google-api-go-client/issues/123
	//     file, _ := driveService.Files.Get(fileId).Do()
    // 	   fmt.Printf("name: '%s' modifiedTime: '%s'\n", file.Name, file.ModifiedTime)
	

	if name != "file.txt" {
		return nil, fuse.ENOENT
	}
	if flags&fuse.O_ANYWRITE != 0 {
		return nil, fuse.EPERM
	}
	return nodefs.NewDataFile([]byte(name)), fuse.OK
}

func main() {
	flag.Parse()
	if len(flag.Args()) < 1 {
		log.Fatal("Usage:\n  hello MOUNTPOINT")
	}

	fileIDs = make(map[string]string);

	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
	        log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
	        log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	fmt.Println("Created Google Drive Client.")

	srvFromClient, errFromClient := drive.New(client)
	if errFromClient != nil {
	        log.Fatalf("Unable to retrieve Drive client: %v", err)
	}

	fmt.Println("Created Google Drive Server.")

	srv = srvFromClient


	nfs := pathfs.NewPathNodeFs(&HelloFs{FileSystem: pathfs.NewDefaultFileSystem()}, nil)

	// NodeFS Docs
	// https://godoc.org/github.com/hanwen/go-fuse/fuse/nodefs
	//
	// To unmount:
	// https://godoc.org/github.com/hanwen/go-fuse/fuse/nodefs#FileSystemConnector.Unmount
	//
	server, _, err := nodefs.MountRoot(flag.Arg(0), nfs.Root(), nil)
	if err != nil {
		log.Fatalf("Mount fail: %v\n", err)
		server.Unmount();
	}
	fmt.Println("Root Mounted.")	

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		 syscall.SIGINT,
	    syscall.SIGKILL,
	    syscall.SIGTERM,
	    syscall.SIGQUIT)
	go func() {
	    s := <-sigc
		fmt.Printf("SIG Caught: %d\nUnmounting.", s)	    
	    server.Unmount();
	}()		

	server.Serve()
}
