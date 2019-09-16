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
	"time"

	"github.com/kkty/scriproxy/pkg/scriproxy"
	"go.uber.org/zap"
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

	logger, err := zap.NewProduction()

	if err != nil {
		log.Fatal(err)
	}

	defer logger.Sync()

	times := make(map[*http.Request]time.Time)
	loggers := make(map[*http.Request]*zap.Logger)

	proxy := httputil.ReverseProxy{
		Director: func(req *http.Request) {
			logger := logger.With(
				zap.String("method", req.Method),
				zap.String("remote_addr", req.RemoteAddr),
				zap.String("original_user_agent", req.UserAgent()),
				zap.String("original_host", req.Host),
				zap.String("original_url_scheme", req.URL.Scheme),
				zap.String("original_url_host", req.URL.Host),
				zap.String("original_url_path", req.URL.Path),
				zap.String("original_url_query", req.URL.RawQuery),
			)

			requestRewriter.Rewrite(req)

			logger = logger.With(
				zap.String("host", req.Host),
				zap.String("url_scheme", req.URL.Scheme),
				zap.String("url_host", req.URL.Host),
				zap.String("url_path", req.URL.Path),
				zap.String("url_query", req.URL.RawQuery),
			)

			loggers[req] = logger
			times[req] = time.Now()
		},
		ModifyResponse: func(res *http.Response) error {
			loggers[res.Request].Info("received_response",
				zap.Int("status_code", res.StatusCode),
				zap.Duration("elapsed", time.Now().Sub(times[res.Request])))

			delete(times, res.Request)
			delete(loggers, res.Request)

			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, req *http.Request, err error) {
			logger.Error("error_from_proxy",
				zap.Duration("elapsed", time.Now().Sub(times[req])),
				zap.Error(err))

			delete(times, req)
			delete(loggers, req)
		},
	}

	port := os.Getenv("PORT")

	if port == "" {
		port = "80"
	}

	http.ListenAndServe(fmt.Sprintf(":%s", port), &proxy)
}
