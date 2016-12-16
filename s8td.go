package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

var port int
var uploadRoot string
var key string

func init() {
	flag.IntVar(&port, "port", 8080, "http listen port")
	flag.StringVar(&uploadRoot, "uploadRoot", "/tmp", "root path for uploads")
	flag.StringVar(&key, "key", "", "key for signing upload requests")
}

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/", getHandler)

	fmt.Printf("Listening on port %d\n", port)
	fmt.Printf("Upload root is %s\n", uploadRoot)

	portString := fmt.Sprintf(":%d", port)
	err := http.ListenAndServe(portString, nil)
	if err != nil {
		fmt.Printf("Unable to start HTTP server: %s", err)
		os.Exit(1)
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func checkSig(data, sig, key []byte) bool {
	mac := hmac.New(sha1.New, key)
	_, err := mac.Write(data)
	if err != nil {
		return false
	}
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(sig, expectedMAC)
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 2 {
		http.Error(w, "not found", 404)
		return
	}

	id := parts[1]
	filePath := path.Join(uploadRoot, id)

	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}

	defer func() {
		closeErr := file.Close()
		if closeErr != nil {
			fmt.Printf("Unable to close file %s", closeErr)
		}
	}()

	_, err = io.Copy(w, file)

	if err != nil {
		http.Error(w, "not found", 404)
		return
	}

}

func uploadHandler(w http.ResponseWriter, r *http.Request) {

	tsStr := r.FormValue("ts")
	sigStr := r.FormValue("sig")

	fmt.Printf("ts: %s sig: %s\n", tsStr, sigStr)

	sig, err := hex.DecodeString(sigStr)
	if err != nil {
		http.Error(w, "couldnt decode sig", 400)
		return
	}

	ts, err := strconv.Atoi(tsStr)
	if err != nil {
		http.Error(w, "ts might not be a number?", 400)
		return
	}

	if math.Abs(float64(time.Now().Unix()-int64(ts))) > 30 {
		fmt.Println("Err: Request too old")
		http.Error(w, "Request too old", 400)
		return
	}

	if !checkSig([]byte(tsStr), sig, []byte(key)) {
		fmt.Println("Err: sig no match")
		http.Error(w, "Sig no match", 400)
		return
	}

	fmt.Println("all ok")

	uploadedFile, _, err := r.FormFile("file")

	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	defer func() {
		upldErr := uploadedFile.Close()
		if upldErr != nil {
			fmt.Printf("Unable to close uploaded file %s", upldErr)
		}
	}()

	id := randString(8)
	filePath := path.Join(uploadRoot, id)

	out, err := os.Create(filePath)
	if err != nil {
		fmt.Fprintf(w, "Unable to create the file for writing. Check your write access privilege")
		return
	}

	defer func() {
		outErr := out.Close()
		if outErr != nil {
			fmt.Printf("Unable to close save destination for uplaoded file %s", outErr)
		}
	}()

	// write the content from POST to the file
	_, err = io.Copy(out, uploadedFile)
	if err != nil {
		fmt.Fprintln(w, err)
	}

	url := fmt.Sprintf("http://%s/%s", r.Host, id)
	fmt.Fprintf(w, url)
}
