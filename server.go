package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/kkty/scriproxy/pkg/scriproxy"
)

func readBytesFromFile(fileName string) ([]byte, error) {
	f, err := os.Open(fileName)

	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(f)
}

func main() {
	var scriptFileName, libraries string

	flag.StringVar(&scriptFileName, "script", "", "")
	flag.StringVar(&libraries, "libraries", "", "")
	flag.Parse()

	if scriptFileName == "" {
		log.Fatal("--script should be specified")
	}

	script, err := readBytesFromFile(scriptFileName)

	if err != nil {
		log.Fatal(err)
	}

	requestRewriter, err := scriproxy.NewRequestRewriter(
		script,
		strings.Split(libraries, ","),
	)

	if err != nil {
		log.Fatal(err)
	}

	proxy := httputil.ReverseProxy{
		Director: func(req *http.Request) {
			if err := requestRewriter.Rewrite(req); err != nil {
				log.Fatal(err)
			}
		},
	}

	port := os.Getenv("PORT")

	if port == "" {
		port = "80"
	}

	http.ListenAndServe(fmt.Sprintf(":%s", port), &proxy)
}
