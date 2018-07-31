package main

import (
        "encoding/json"
        "fmt"
        "io/ioutil"
        "log"
        "net/http"
        "os"

        "golang.org/x/net/context"
        "golang.org/x/oauth2"
        "golang.org/x/oauth2/google"
        "google.golang.org/api/drive/v3"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
        tokenFile := "token.json"
        tok, err := tokenFromFile(tokenFile)
        if err != nil {
                tok = getTokenFromWeb(config)
                saveToken(tokenFile, tok)
        }
        return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
        authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
        fmt.Printf("Go to the following link in your browser then type the "+
                "authorization code: \n%v\n", authURL)

        var authCode string
        if _, err := fmt.Scan(&authCode); err != nil {
                log.Fatalf("Unable to read authorization code %v", err)
        }

        tok, err := config.Exchange(oauth2.NoContext, authCode)
        if err != nil {
                log.Fatalf("Unable to retrieve token from web %v", err)
        }
        return tok
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

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
        fmt.Printf("Saving credential file to: %s\n", path)
        f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
        defer f.Close()
        if err != nil {
                log.Fatalf("Unable to cache oauth token: %v", err)
        }
        json.NewEncoder(f).Encode(token)
}

func main() {
        b, err := ioutil.ReadFile("credentials.json")
        if err != nil {
                log.Fatalf("Unable to read client secret file: %v", err)
        }


        // Scopes from:
        // https://github.com/google/google-api-go-client/issues/218

        // drive.DriveScope,
        // drive.DriveReadonlyScope,
        // drive.DriveAppdataScope,
        // drive.DriveFileScope,
        // drive.DriveMetadataScope,
        // drive.DriveMetadataReadonlyScope,
        // drive.DrivePhotosReadonlyScope,

        // If modifying these scopes, delete your previously saved token.json.
        config, err := google.ConfigFromJSON(b, drive.DriveScope)
        if err != nil {
                log.Fatalf("Unable to parse client secret file to config: %v", err)
        }
        client := getClient(config)

        srv, err := drive.New(client)
        if err != nil {
                log.Fatalf("Unable to retrieve Drive client: %v", err)
        }

        r, err := srv.Files.List().PageSize(10).
                Fields("nextPageToken, files(id, name)").Do()
        if err != nil {
                log.Fatalf("Unable to retrieve files: %v", err)
        }
        fmt.Println("Files:")
        if len(r.Files) == 0 {
                fmt.Println("No files found.")
        } else {
                for _, i := range r.Files {
                        fmt.Printf("%s (%s)\n", i.Name, i.Id)
                }
        }
}

