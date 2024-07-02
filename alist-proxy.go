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

var (
	help       bool
	configFile string
	s          sign.Sign
)

var config struct {
	Port     int    `yaml:"port"`
	Https    bool   `yaml:"https"`
	Help     bool   `yaml:"help"`
	CertFile string `yaml:"certFile"`
	KeyFile  string `yaml:"keyFile"`
	Address  string `yaml:"address"`
	Token    string `yaml:"token"`
}

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

func init() {
	flag.BoolVar(&help, "h", false, "help")
	flag.BoolVar(&help, "help", false, "help")
	flag.StringVar(&configFile, "c", "config.yaml", "path to config.yaml")
	flag.Parse()
	err := loadConfig(configFile)
	if err != nil {
		// 如果配置文件不存在，则在当前目录下创建一个默认配置文件
		if os.IsNotExist(err) {
			fmt.Println("config.yaml not found, create a default config file")
			defaultConfig := `# the proxy port
port: 5243
# use https protocol
https: false
# https cert file
certFile: server.crt
# https key file
keyFile: server.key
# alist server address
address: http://example.com
# alist server api token
token: alist-xxx
`
			err := os.WriteFile("config.yaml", []byte(defaultConfig), 0644)
			if err != nil {
				fmt.Printf("Create config file error: %s\n", err.Error())
				os.Exit(1)
			}
			fmt.Println("Create config file success, please edit it and restart the proxy!")
			os.Exit(0)
		}
		fmt.Printf("load config error: %s\n", err.Error())
		os.Exit(1)
	}
	s = sign.NewHMACSign([]byte(config.Token))
}

var HttpClient = &http.Client{}

type Json map[string]interface{}

type Result struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func errorResponse(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("content-type", "text/json")
	res, _ := json.Marshal(Result{Code: code, Msg: msg})
	w.WriteHeader(200)
	_, _ = w.Write(res)
}

func downHandle(w http.ResponseWriter, r *http.Request) {
	sign := r.URL.Query().Get("sign")
	filePath := r.URL.Path
	err := s.Verify(filePath, sign)
	if err != nil {
		// 签名验证失败，写入日志，格式为：
		// [error] host:port - [date] - error - method - proto - path
		fmt.Printf("[error] %s - [%s] - %s - %s - %s - %s\n",
			r.RemoteAddr,
			time.Now().Format("02/Jan/2006:15:04:05 +0800"),
			err.Error(),
			r.Method,
			r.Proto,
			r.URL.Path,
		)
		errorResponse(w, 401, err.Error())
		return
	}
	data := Json{
		"path": filePath,
	}
	dataByte, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/fs/link", config.Address), bytes.NewBuffer(dataByte))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", config.Token)
	res, err := HttpClient.Do(req)
	if err != nil {
		errorResponse(w, 500, err.Error())
		return
	}
	defer func() {
		_ = res.Body.Close()
	}()
	dataByte, err = io.ReadAll(res.Body)
	if err != nil {
		errorResponse(w, 500, err.Error())
		return
	}
	var resp LinkResp
	err = json.Unmarshal(dataByte, &resp)
	if err != nil {
		errorResponse(w, 500, err.Error())
		return
	}
	if resp.Code != 200 {
		errorResponse(w, resp.Code, resp.Message)
		return
	}
	if !strings.HasPrefix(resp.Data.Url, "http") {
		resp.Data.Url = "http:" + resp.Data.Url
	}
	// 请求链接成功，写入日志：
	// [error] host:port - [date] - error - method - proto - path
	fmt.Printf("[info] %s - [%s] - - %s - %s - %s\n",
		r.RemoteAddr,
		time.Now().Format("02/Jan/2006:15:04:05 +0800"),
		r.Method,
		r.Proto,
		r.URL.Path,
	)
	req2, _ := http.NewRequest(r.Method, resp.Data.Url, nil)
	for h, val := range r.Header {
		req2.Header[h] = val
	}
	for h, val := range resp.Data.Header {
		req2.Header[h] = val
	}
	res2, err := HttpClient.Do(req2)
	if err != nil {
		errorResponse(w, 500, err.Error())
		return
	}
	defer func() {
		_ = res2.Body.Close()
	}()
	res2.Header.Del("Access-Control-Allow-Origin")
	res2.Header.Del("set-cookie")
	for h, v := range res2.Header {
		w.Header()[h] = v
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Add("Access-Control-Allow-Headers", "range")
	w.WriteHeader(res2.StatusCode)
	_, err = io.Copy(w, res2.Body)
	if err != nil {
		errorResponse(w, 500, err.Error())
		return
	}
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
	if !config.Https {
		if err := s.ListenAndServe(); err != nil {
			fmt.Printf("failed to start: %s\n", err.Error())
		}
	} else {
		if err := s.ListenAndServeTLS(config.CertFile, config.KeyFile); err != nil {
			fmt.Printf("failed to start: %s\n", err.Error())
		}
	}
}
