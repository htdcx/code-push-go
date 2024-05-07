package utils

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"com.lc.go.codepush/client/constants"
)

type respErr struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Success bool   `json:"success"`
}

func HttpPost[T any](url string, body []byte) (*T, error) {
	return HttpPostToken[T](url, body, nil)
}
func HttpPostToken[T any](url string, body []byte, token *string) (*T, error) {
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if token != nil {
		req.Header.Set("token", *token)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 200 {
		var t T
		err := json.Unmarshal(respBody, &t)
		if err != nil {
			return nil, err
		}
		return &t, nil
	} else {
		var respErr respErr
		err := json.Unmarshal(respBody, &respErr)
		if err != nil {
			return nil, err
		}
		if respErr.Code == 1100 {
			if err := DelLoginfo(); err != nil {
				log.Panic(err.Error())
			}
		}
		return nil, errors.New(respErr.Msg)
	}
}

func HttpGet[T any](url string) (*T, error) {
	return HttpGetToken[T](url, nil)
}

func HttpGetToken[T any](url string, token *string) (*T, error) {
	req, _ := http.NewRequest("GET", url, nil)
	if token != nil {
		req.Header.Set("token", *token)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 200 {
		var t T
		err := json.Unmarshal(respBody, &t)
		if err != nil {
			return nil, err
		}
		return &t, nil
	} else {
		var respErr respErr
		err := json.Unmarshal(respBody, &respErr)
		if err != nil {
			return nil, err
		}
		if respErr.Code == 1100 {
			if err := DelLoginfo(); err != nil {
				log.Panic(err.Error())
			}
		}
		return nil, errors.New(respErr.Msg)
	}
}

func FileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	hash := md5.New()
	_, _ = io.Copy(hash, file)
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func Zip(src_dir string, zip_file_name string) {

	// 预防：旧文件无法覆盖
	os.RemoveAll(zip_file_name)

	// 创建：zip文件
	zipfile, _ := os.Create(zip_file_name)
	defer zipfile.Close()

	// 打开：zip文件
	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	nowSrc := strings.Replace(src_dir, "./", "", 1)
	// 遍历路径信息
	filepath.Walk(src_dir, func(path string, info os.FileInfo, _ error) error {

		// 如果是源路径，提前进行下一个遍历
		if path == src_dir {
			return nil
		}

		// 获取：文件头信息
		header, _ := zip.FileInfoHeader(info)
		header.Name = strings.TrimPrefix(path, nowSrc+`/`)

		// 判断：文件是不是文件夹
		if info.IsDir() {
			header.Name += `/`
		} else {
			// 设置：zip的文件压缩算法
			header.Method = zip.Deflate
		}

		// 创建：压缩包头部信息
		writer, _ := archive.CreateHeader(header)
		if !info.IsDir() {
			file, _ := os.Open(path)
			defer file.Close()
			io.Copy(writer, file)
		}
		return nil
	})
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func GetLoginfo() (*constants.SaveLoginInfo, error) {
	if b, _ := PathExists("./.code-push-go.json"); !b {
		return nil, errors.New("no login, usage: code-push-go login -u <UserName> -p <Password> -h <ServerUrl>")
	}
	info, err := os.ReadFile("./.code-push-go.json")
	if err != nil {
		return nil, errors.New("read config fail")
	}
	saveLoginInfo := constants.SaveLoginInfo{}
	err = json.Unmarshal(info, &saveLoginInfo)
	if err != nil {
		return nil, errors.New("read config fail")
	}
	return &saveLoginInfo, nil
}
func DelLoginfo() error {
	return os.Remove("./.code-push-go.json")
}
func MD5(str string) string {
	data := []byte(str)
	has := md5.Sum(data)
	md5str1 := fmt.Sprintf("%x", has)
	return md5str1
}
