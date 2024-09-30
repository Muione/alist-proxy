package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alist-org/alist/v3/pkg/sign"
	"gopkg.in/yaml.v2"
)

type Link struct {
	Url    string      `json:"url"`
	Header http.Header `json:"header"`
}

type LinkResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    Link   `json:"data"`
}

type Json map[string]interface{}

type Result struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

var config struct {
	Port      int    `yaml:"port"`
	Https     bool   `yaml:"https"`
	CertFile  string `yaml:"certFile"`
	KeyFile   string `yaml:"keyFile"`
	Address   string `yaml:"address"`
	Token     string `yaml:"token"`
	UserAgent string `yaml:"userAgent"`
}

var (
	help       bool
	configFile string
	signer     sign.Sign
	HttpClient = &http.Client{}
)

func loadConfig(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return err
	}

	return nil
}

// createDefaultConfig creates a default config file with example values.
func createDefaultConfig(filename string) error {
	defaultConfig := `# Proxy port
port: 5243

# Use HTTPS (true/false)
https: false

# HTTPS certificate file (if https is true)
certFile: server.crt

# HTTPS key file (if https is true)
keyFile: server.key

# Alist server address
address: http://your-alist-server

# Alist server API token
token: alist-xxx

# User-Agent header to use
userAgent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36
`
	return os.WriteFile(filename, []byte(defaultConfig), 0644)
}

func errorResponse(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("content-type", "text/json")
	res, _ := json.Marshal(Result{Code: code, Msg: msg})
	_, _ = w.Write(res)
}

func downHandle(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Path
	signature := r.URL.Query().Get("sign")

	err := signer.Verify(filePath, signature)
	if err != nil {
		errorResponse(w, http.StatusUnauthorized, err.Error())
		logInfo("error", r, err.Error())
		return
	}

	link, err := getDownloadLink(filePath)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		logInfo("fail", r, err.Error())
		return
	}
	logInfo("info", r, "")

	err = proxyDownload(w, r, link)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		// logInfo("failed", r, err.Error())
		return
	}
}

// getDownloadLink retrieves the download link from the Alist server
func getDownloadLink(filePath string) (*Link, error) {
	data := Json{"path": filePath}
	dataByte, _ := json.Marshal(data)

	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/fs/link", config.Address), bytes.NewBuffer(dataByte))
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", config.Token)
	req.Header.Set("User-Agent", "Alist-Proxy")

	res, err := HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	dataByte, err = io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var resp LinkResp
	err = json.Unmarshal(dataByte, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Code != http.StatusOK {
		return nil, fmt.Errorf("alist server returned error: %s", resp.Message)
	}

	if !strings.HasPrefix(resp.Data.Url, "http") {
		resp.Data.Url = "http:" + resp.Data.Url
	}

	return &resp.Data, nil
}

// proxyDownload proxies the file download from the Alist server to the client
func proxyDownload(w http.ResponseWriter, r *http.Request, link *Link) error {
	req, _ := http.NewRequest(r.Method, link.Url, nil)
	req.Header.Set("User-Agent", config.UserAgent)

	if r.Header != nil {
		for k, v := range r.Header {
			req.Header[k] = v
		}
	}

	if link.Header != nil {
		for k, v := range link.Header {
			req.Header[k] = v
		}
	}

	var res *http.Response
	var err error

	for {
		res, err = HttpClient.Do(req)
		if err != nil {
			return err
		}
		if res.StatusCode < 300 || res.StatusCode >= 400 {
			break
		}
		location := res.Header.Get("Location")
		if location == "" {
			break
		}
		if strings.HasPrefix(location, config.Address+"/") {
			req, err = http.NewRequest(req.Method, location, req.Body)
			if err != nil {
				return err
			}
			downHandle(w, req)
			return nil

		} else {
			req, err = http.NewRequest(req.Method, location, req.Body)
			if err != nil {
				return err
			}
		}
	}

	defer res.Body.Close()

	res.Header.Del("Access-Control-Allow-Origin")
	res.Header.Del("set-cookie")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Add("Access-Control-Allow-Headers", "range")

	for h, v := range res.Header {
		w.Header()[h] = v
	}

	w.WriteHeader(res.StatusCode)

	_, err = io.Copy(w, res.Body)
	return err
}

// logInfo logs an info message with request details
func logInfo(logType string, r *http.Request, errMessage string) {
	fmt.Printf("[%s] %s - [%s] - [%s] - %s - %s - %s\n",
		logType,
		r.RemoteAddr,
		time.Now().Format("2006-01-02 15:04:05"),
		errMessage,
		r.Method,
		r.Proto,
		r.URL.Path,
	)
}

func init() {
	flag.BoolVar(&help, "h", false, "help")
	flag.BoolVar(&help, "help", false, "help")
	flag.StringVar(&configFile, "c", "config.yaml", "path to config.yaml")
	flag.Parse()
	err := loadConfig(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("load config error: %s\n", err.Error())
			os.Exit(1)
		}
		err := createDefaultConfig(configFile)
		if err != nil {
			fmt.Printf("Create config file error: %s\n", err.Error())
			os.Exit(1)
		}
		fmt.Println("Create config file success, please edit it and restart the proxy!")
		os.Exit(0)
	}
	signer = sign.NewHMACSign([]byte(config.Token))
}

func main() {
	if help {
		flag.Usage()
		return
	}
	addr := fmt.Sprintf(":%d", config.Port)
	fmt.Printf("listening on port: %s\n", addr)
	s := http.Server{
		Addr:    addr,
		Handler: http.HandlerFunc(downHandle),
	}

	var err error
	if !config.Https {
		err = s.ListenAndServe()
	} else {
		err = s.ListenAndServeTLS(config.CertFile, config.KeyFile)
	}
	if err != nil {
		fmt.Printf("failed to start: %s\n", err.Error())
	}
}
