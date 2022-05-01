package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"html/template"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"
)

const MaxFileSize = 8 << 20 // 8MB
const FileKey = "upload"
const AuthTokenKey = "auth-token"

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Some error occured while loading .env file. Err: %s", err)
	}

	tmpl := template.Must(template.ParseFiles("index.html"))
	authToken, ok := os.LookupEnv("TOKEN")

	if !ok {
		fmt.Print("Auth token not set")
	}

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		err := tmpl.Execute(writer, struct{ AuthToken string }{authToken})

		if err != nil {
			return
		}
	})

	http.HandleFunc("/upload", func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case "POST":
			// 10MB max memory for the form
			if err := request.ParseMultipartForm(10 << 20); err != nil {
				http.Error(writer, err.Error(), http.StatusBadRequest)
				return
			}

			file, header, err := request.FormFile(FileKey)
			if err != nil {
				fmt.Printf("Error getting file with key %s: %s", FileKey, err.Error())
				http.Error(writer, err.Error(), http.StatusBadRequest)
				return
			}

			defer func(file multipart.File) {
				err := file.Close()
				if err != nil {
					fmt.Printf("Error closing file %+v", err)
				}
			}(file)

			if header.Size > MaxFileSize {
				http.Error(writer, fmt.Sprintf("The file %s is too large. Please upload an image less that %dMB",
					header.Filename, MaxFileSize>>20), http.StatusForbidden)
				return
			}

			buff := make([]byte, 512)
			_, err = file.Read(buff)
			if err != nil {
				fmt.Printf("Error reading buffer: %s", err.Error())
				http.Error(writer, fmt.Sprintf("An error occured"), http.StatusInternalServerError)
				return
			}

			filetype := http.DetectContentType(buff)
			if !strings.HasPrefix(filetype, "image/") {
				http.Error(writer, "File type is unsupported. Please upload an image", http.StatusForbidden)
			}

			if request.FormValue(AuthTokenKey) != authToken {
				http.Error(writer, "Invalid auth token", http.StatusForbidden)
			}

			// WRITE TO A TEMP FILE
			if err = ioutil.WriteFile(fmt.Sprintf("temp-%d", time.Now().Unix()),
				[]byte(fmt.Sprintf("File name: %s\nFile size: %d\nFile header: %s",
					header.Filename, header.Size, header.Header)), 0666); err != nil {
				fmt.Printf("Error writing to temp file %+v", err)
				return
			}

		default:
			fmt.Printf("%s request not handled", request.Method)
		}

		fmt.Fprintf(writer, "Upload Successful")
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
