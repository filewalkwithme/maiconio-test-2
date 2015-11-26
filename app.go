package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"

	_ "github.com/maiconio/maiconio-test-2/deps/github.com/lib/pq"
	//for extracting service credentials from VCAP_SERVICES
	//"github.com/cloudfoundry-community/go-cfenv"
)

const (
	DEFAULT_PORT = "8080"
)

var db *sql.DB

//ElephantsqlCredentials represents the credentials for Elephantsql
type ElephantsqlCredentials struct {
	URI string `json:"uri"`
}

//Elephantsql represents the Elephantsql connection
type Elephantsql struct {
	Credendials ElephantsqlCredentials `json:"credentials"`
}

//VCAPServices represents the VCAP environment var
type VCAPServices struct {
	Elephantsql []Elephantsql `json:"elephantsql"`
}

var index = template.Must(template.ParseFiles(
	"templates/_base.html",
	"templates/index.html",
))

func indexHandler(w http.ResponseWriter, req *http.Request) {
	index.Execute(w, nil)
}

func helloworld(w http.ResponseWriter, req *http.Request) {
	index.Execute(w, nil)
}

func initAPP() {
	vcapJSON := os.Getenv("VCAP_SERVICES")

	if len(vcapJSON) == 0 {
		log.Fatalf("VCAP_SERVICES not defined")
	}

	var vcap VCAPServices
	err := json.Unmarshal([]byte(vcapJSON), &vcap)
	if err != nil {
		log.Fatalf("err: %v\n", err)
	}

	log.Printf("Connecting on %v\n", vcap.Elephantsql[0].Credendials.URI)
	tmpDB, err := sql.Open("postgres", vcap.Elephantsql[0].Credendials.URI)
	if err != nil {
		log.Fatal(err)
	}
	db = tmpDB

	//err = db.Ping()
	if err != nil {
		log.Fatalf("ping: %v\n", err)
	}
}

func uploadHandler(w http.ResponseWriter, req *http.Request) {
	uri := "https://gateway.watsonplatform.net/visual-recognition-beta/api/v1/tag/recognize"
	paramName := "img_File"

	err := req.ParseForm()
	if err != nil {
		fmt.Fprintf(w, "req.ParseForm(): %v\n", err)
		return
	}

	_, fileHeader, err := req.FormFile(paramName)
	if err != nil {
		fmt.Fprintf(w, "req.FormFile(\"img_File\"): %v\n", err)
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		fmt.Fprintf(w, "fileHeader.Open(): %v\n", err)
		return
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, fileHeader.Filename)
	if err != nil {
		fmt.Fprintf(w, "writer.CreateFormFile(paramName, fileHeader.Filename): %v\n", err)
		return
	}

	_, err = io.Copy(part, file)
	if err != nil {
		fmt.Fprintf(w, "io.Copy(part, file): %v\n", err)
		return
	}

	fileSave, err := fileHeader.Open()
	if err != nil {
		fmt.Fprintf(w, "[save]fileHeader.Open(): %v\n", err)
		return
	}

	fBuf, err := ioutil.ReadAll(fileSave)
	if err != nil {
		fmt.Fprintf(w, "ioutil.ReadAll(file): %v\n", err)
		return
	}

	err = ioutil.WriteFile("static/images/"+fileHeader.Filename, fBuf, 0600)
	if err != nil {
		fmt.Fprintf(w, "ioutil.WriteFile(\"photos/%v\"): %v\n", fileHeader.Filename, err)
		return
	}

	err = writer.Close()
	if err != nil {
		fmt.Fprintf(w, "writer.Close(): %v\n", err)
		return
	}

	request, err := http.NewRequest("POST", uri, body)
	if err != nil {
		fmt.Fprintf(w, "http.NewRequest(\"POST\", uri, body): %v\n", err)
		return
	}

	request.Header.Add("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		fmt.Fprintf(w, "client.Do(request): %v\n", err)
	} else {
		body := &bytes.Buffer{}
		_, err := body.ReadFrom(resp.Body)
		if err != nil {
			fmt.Fprintf(w, "body.ReadFrom(resp.Body): %v\n", err)
			return
		}

		err = resp.Body.Close()
		if err != nil {
			fmt.Fprintf(w, "resp.Body.Close(): %v\n", err)
			return
		}

		fmt.Fprintf(w, "%v\n", resp.StatusCode)
		fmt.Fprintf(w, "%v\n", resp.Header)

		fmt.Fprintf(w, "%v\n", body)
	}
}

func main() {
	initAPP()

	var port string
	if port = os.Getenv("PORT"); len(port) == 0 {
		port = DEFAULT_PORT
	}

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/upload", uploadHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Printf("Starting app on port %+v\n", port)
	http.ListenAndServe(":"+port, nil)
}
