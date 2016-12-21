package main

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
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
var keyFile string
var keys map[string][]byte

func init() {
	flag.IntVar(&port, "port", 8080, "http listen port")
	flag.StringVar(&uploadRoot, "uploadRoot", "/tmp", "root path for uploads")
	flag.StringVar(&keyFile, "keyFile", "", "Path to file with id:key pairs")
}

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	loadKeys()

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

func loadKeys() {

	file, err := os.Open(keyFile)
	if err != nil {
		fmt.Print("Unable to load keys, file missing")
		os.Exit(2)
	}

	tmp := map[string][]byte{}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		parts := strings.Split(":", scanner.Text())
		if len(parts) != 2 {
			fmt.Print("Unable to load keys, bad file format")
			os.Exit(2)
		}
		tmp[parts[0]] = []byte(parts[1])
	}

	if err := scanner.Err(); err != nil {
		fmt.Print("Unable to load keys, err reading file: ", err)
	}

	keys = tmp
}

func lookupKey(id string) ([]byte, error) {
	k, ok := keys[id]
	if !ok {
		return nil, fmt.Errorf("ID: '%s' not found", id)
	}

	return k, nil
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func checkSig(data string, sig []byte, key []byte) bool {
	mac := hmac.New(sha1.New, key)
	_, err := io.WriteString(mac, data)
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

	id := r.FormValue("id")
	tsStr := r.FormValue("ts")
	sigStr := r.FormValue("sig")

	fmt.Printf("id: %s ts: %s sig: %s\n", id, tsStr, sigStr)

	sig, err := hex.DecodeString(sigStr)
	if err != nil {
		http.Error(w, "couldnt decode sig", 400)
		return
	}

	ts, err := strconv.ParseInt(tsStr, 10, 0)
	if err != nil {
		http.Error(w, "ts might not be a number?", 400)
		return
	}

	key, err := lookupKey(id)
	if err != nil {
		http.Error(w, "Err: id not found", 400)
		return
	}

	if !validateTimestamp(ts) {
		fmt.Println("Err: Request too old")
		http.Error(w, "Request too old", 400)
		return
	}

	if !checkSig(tsStr, sig, key) {
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

	fid := randString(8)
	filePath := path.Join(uploadRoot, fid)

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

	url := fmt.Sprintf("http://%s/%s", r.Host, fid)
	fmt.Fprintf(w, url)
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	if x == 0 {
		return 0
	}
	return x
}

func validateTimestamp(checkTs int64) bool {
	t := time.Now().Unix()
	tolerance := int64(30)

	d := abs(t - checkTs)

	if d > tolerance {
		return false
	} else {
		return true
	}
}
