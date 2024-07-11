package opt

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"com.lc.go.codepush/client/constants"
	"com.lc.go.codepush/client/utils"
	"github.com/google/uuid"
	"github.com/liushuochen/gotable"
)

type App struct{}

type checkBundleReq struct {
	AppName    *string `json:"appName" binding:"required"`
	Deployment *string `json:"deployment" binding:"required"`
	Version    *string `json:"version" binding:"required"`
}

type checkBundleRep struct {
	AppName *string `json:"appName" binding:"required"`
	OS      *int    `json:"os" binding:"required"`
	Hash    *string `json:"hash"`
}

func (App) CreateBundle() {

	saveLoginInfo, err := utils.GetLoginfo()
	if err != nil {
		log.Println(err.Error())
		return
	}
	serverUrl := saveLoginInfo.ServerUrl
	token := saveLoginInfo.Token
	var targetVersion string
	var appName string
	var deployment string
	var rnDir string
	var description string
	var isMinifyDisabled bool

	flag.StringVar(&targetVersion, "t", "", "Target version")
	flag.StringVar(&appName, "n", "", "AppName")
	flag.StringVar(&deployment, "d", "", "DeploymentName")
	flag.StringVar(&rnDir, "p", "./", "React native project dir")
	flag.StringVar(&description, "description", "", "Description")
	flag.BoolVar(&isMinifyDisabled, "disable-minify", false, "Disable minify")
	flag.Parse()

	if targetVersion == "" || appName == "" || deployment == "" {
		fmt.Println("Usage: code-push-go create_bundle -t <TargetVersion> -n <AppName> -d <deployment> -p <*Optional React native project dir>  --description <*Optional Description>  --disable-minify (*Optional)")
		return
	}
	log.Println("Get app info...")
	Url, err := url.Parse(saveLoginInfo.ServerUrl + "/checkBundle")
	if err != nil {
		log.Panic("server url error :", err.Error())
	}

	checkBundleReq := checkBundleReq{
		AppName:    &appName,
		Deployment: &deployment,
		Version:    &targetVersion,
	}
	jsonByte, _ := json.Marshal(checkBundleReq)

	rep, err := utils.HttpPostToken[checkBundleRep](Url.String(), jsonByte, &saveLoginInfo.Token)
	if err != nil {
		fmt.Println(err)
		return
	}
	osName := "ios"
	if *rep.OS == 2 {
		osName = "android"
	}

	exist, _ := utils.PathExists(rnDir + "build")
	if exist {
		os.RemoveAll(rnDir + "build")
	}
	if err := os.MkdirAll(rnDir+"build/CodePush", os.ModePerm); err != nil {
		log.Panic("Create folder error :" + err.Error())
	}
	log.Println("Create bundle...")
	jsName := "main.jsbundle"
	if osName == "android" {
		jsName = "index.android.bundle"
	}

	minify := "true"
	if isMinifyDisabled {
		minify = "false"
	}

	cmd := exec.Command(
		"npx",
		"react-native",
		"bundle",
		"--assets-dest",
		rnDir+"build/CodePush",
		"--bundle-output",
		rnDir+"build/CodePush/"+jsName,
		"--dev",
		"false",
		"--entry-file",
		"index.js",
		"--platform",
		osName,
		"--minify",
		minify)
	cmd.Dir = rnDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("combined out:\n%s\n", string(out))
		log.Panic("cmd.Run() failed with ", err)
	}

	hash, error := getHash(rnDir + "build")
	if error != nil {
		log.Panic("hash error", error.Error())
	}
	if rep.Hash != nil && *rep.Hash != "" {
		if hash == *rep.Hash {
			fmt.Printf("Server bundle is new!")
			return
		}
	}
	uuidStr, _ := uuid.NewUUID()
	fileName := uuidStr.String() + ".zip"
	utils.Zip(rnDir+"build", fileName)
	os.RemoveAll(rnDir + "build")
	log.Println("Upload File...")

	if err != nil {
		log.Panic(err.Error())
	}
	Url, err = url.Parse(serverUrl + "/uploadBundle")
	if err != nil {
		log.Panic(err.Error())
	}
	req, err := newfileUploadRequest(Url.String(), nil, "file", fileName)
	if err != nil {
		log.Panic(err.Error())
	}
	req.Header.Set("token", token)
	client1 := &http.Client{}
	resp, err := client1.Do(req)
	if err != nil {
		log.Panic(err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Println("Upload fail")
		return
	}

	log.Println("Upload Success")
	log.Println("Create version...")

	Url, err = url.Parse(serverUrl + "/createBundle")
	if err != nil {
		log.Panic("Server url error :", err)
	}
	fileInfo, _ := os.Stat(fileName)
	size := fileInfo.Size()
	key := uuidStr.String() + ".zip"
	createBundleReq := createBundleReq{
		Version:     &targetVersion,
		DownloadUrl: &key,
		Size:        &size,
		AppName:     &appName,
		Deployment:  &deployment,
		Hash:        &hash,
		Description: &description,
	}
	jsonByte, _ = json.Marshal(createBundleReq)
	req, _ = http.NewRequest("POST", Url.String(), bytes.NewBuffer(jsonByte))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("token", token)
	client := &http.Client{}
	resp, err = client.Do(req)

	if err != nil {
		log.Panic("Create version error:", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		log.Println("Create version Success")
	} else {
		log.Panic("Create version error:", err.Error())
	}
	os.RemoveAll(fileName)
}

func getHash(path string) (string, error) {
	files, err := getAllFiles(path)
	if err != nil {
		return "", err
	}
	var fileHash []string
	for _, table1 := range files {
		fb, _ := os.ReadFile(table1)

		h := sha256.New()
		h.Write(fb)
		hash := h.Sum(nil)
		fileName := strings.TrimPrefix(table1, path)
		fileHash = append(fileHash, fileName+":"+fmt.Sprintf("%x", hash))
	}
	j, _ := json.Marshal(fileHash)
	jStr := strings.ReplaceAll(string(j), "\\/", "/")
	h := sha256.New()
	h.Write([]byte(jStr))
	hash := h.Sum(nil)
	return fmt.Sprintf("%x", hash), nil
}

func getAllFiles(dirPth string) (files []string, err error) {
	var dirs []string
	dir, err := os.ReadDir(dirPth)
	if err != nil {
		return nil, err
	}

	PthSep := string(os.PathSeparator)

	for _, fi := range dir {
		if fi.IsDir() {
			dirs = append(dirs, dirPth+PthSep+fi.Name())
			getAllFiles(dirPth + PthSep + fi.Name())
		} else {
			files = append(files, dirPth+PthSep+fi.Name())
		}
	}

	for _, table := range dirs {
		temp, _ := getAllFiles(table)
		for _, temp1 := range temp {
			files = append(files, temp1)
		}
	}

	return files, nil
}

func newfileUploadRequest(uri string, params map[string]string, paramName, path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, path)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}
	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("POST", uri, body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request, err
}

type createBundleReq struct {
	AppName     *string `json:"appName" binding:"required"`
	Deployment  *string `json:"deployment" binding:"required"`
	DownloadUrl *string `json:"downloadUrl" binding:"required"`
	Version     *string `json:"version" binding:"required"`
	Size        *int64  `json:"size" binding:"required"`
	Hash        *string `json:"hash" binding:"required"`
	Description *string `json:"description"`
}

func (a App) App(arge []string) {
	help := "Usage: code-push-go app <operate>\n" +
		"Commands:\n" +
		"	create_app\n" +
		"	create_deployment\n" +
		"	delete_app\n" +
		"	delete_deployment\n" +
		"	ls_app\n" +
		"	ls_deployment"
	if len(arge) < 2 {
		fmt.Println(help)
		return
	}
	command := arge[1]
	switch command {
	case "create_app":
		App{}.createApp()
	case "create_deployment":
		App{}.createDeployment()
	case "delete_app":
		App{}.deleteApp()
	case "delete_deployment":
		App{}.deleteDeployment()
	case "ls_app":
		App{}.lsApp()
	case "ls_deployment":
		App{}.lsDeployment()
	default:
		fmt.Println(help)
		return
	}

}

type createAppReq struct {
	AppName *string `json:"appName" binding:"required"`
	OS      *int    `json:"os" binding:"required"`
}

func (App) createApp() {
	saveLoginInfo, err := utils.GetLoginfo()
	if err != nil {
		log.Println(err.Error())
		return
	}

	var appName string
	var os string

	flag.StringVar(&appName, "n", "", "AppName")
	flag.StringVar(&os, "os", "", "AppName")

	flag.Parse()
	if appName == "" {
		fmt.Println("Usage: code-push-go app create_app -n <AppName> -os <ios or android>")
		return
	}
	Url, err := url.Parse(saveLoginInfo.ServerUrl + "/createApp")
	if err != nil {
		log.Panic("server url error :", err.Error())
	}

	osInt := 1
	if os == "android" {
		osInt = 2
	}
	createAppReq := createAppReq{
		AppName: &appName,
		OS:      &osInt,
	}
	jsonByte, _ := json.Marshal(createAppReq)

	reqStatus, err := utils.HttpPostToken[constants.RespStatus](Url.String(), jsonByte, &saveLoginInfo.Token)
	if err != nil {
		fmt.Println(err)
		return
	}
	if reqStatus.Success {
		fmt.Println("Create app " + appName + " success")
	}
}

type createDeploymentInfo struct {
	AppName        *string `json:"appName" binding:"required"`
	DeploymentName *string `json:"deploymentName" binding:"required"`
}
type deploymentInfoResp struct {
	Name *string `json:"name" binding:"required"`
	Key  *string `json:"key" binding:"required"`
}

func (App) createDeployment() {
	saveLoginInfo, err := utils.GetLoginfo()
	if err != nil {
		log.Println(err.Error())
		return
	}

	var deploymentName string
	var appName string
	flag.StringVar(&appName, "n", "", "AppName")

	flag.StringVar(&deploymentName, "dn", "", "DeploymentName")

	flag.Parse()
	if deploymentName == "" || appName == "" {
		fmt.Println("Usage: code-push-go app create_deployment -n <AppName> -dn <DeploymentName>")
		return
	}
	Url, err := url.Parse(saveLoginInfo.ServerUrl + "/createDeployment")
	if err != nil {
		log.Panic("server url error :", err.Error())
	}

	createDeploymentInfo := createDeploymentInfo{
		AppName:        &appName,
		DeploymentName: &deploymentName,
	}
	jsonByte, _ := json.Marshal(createDeploymentInfo)

	deploymentInfoResp, err := utils.HttpPostToken[deploymentInfoResp](Url.String(), jsonByte, &saveLoginInfo.Token)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Create deployment " + *deploymentInfoResp.Name + " success,Deployment key is " + *deploymentInfoResp.Key)
}
func (App) lsApp() {
	saveLoginInfo, err := utils.GetLoginfo()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	Url, err := url.Parse(saveLoginInfo.ServerUrl + "/lsApp")
	if err != nil {
		log.Panic("server url error :", err.Error())
	}

	apps, err := utils.HttpGetToken[[]string](Url.String(), &saveLoginInfo.Token)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	for _, v := range *apps {
		fmt.Println(v)
	}

}

type lsDeploymentReq struct {
	ShowKey *bool   `json:"k" binding:"required"`
	AppName *string `json:"appName" binding:"required"`
}

type lsDeploymentInfo struct {
	AppName     *string           `json:"appName"`
	Deployments *[]deploymentInfo `json:"deployments"`
}

type deploymentInfo struct {
	DeploymentName *string `json:"deploymentName"`
	AppVersion     string  `json:"appVersion"`
	Active         int     `json:"active"`
	Failed         int     `json:"failed"`
	Installed      int     `json:"installed"`
	DeploymentKey  string  `json:"deploymentKey"`
}

func (App) lsDeployment() {
	saveLoginInfo, err := utils.GetLoginfo()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	var appName string
	var showKey bool

	flag.StringVar(&appName, "n", "", "AppName")
	flag.BoolVar(&showKey, "k", false, "Show deployment key")
	flag.Parse()
	if appName == "" {
		fmt.Println("Usage: code-push-go app ls_deployment -n <AppName> -k (Show deployment key)")
		return
	}

	createDeploymentInfo := lsDeploymentReq{
		AppName: &appName,
		ShowKey: &showKey,
	}
	jsonByte, _ := json.Marshal(createDeploymentInfo)
	Url, err := url.Parse(saveLoginInfo.ServerUrl + "/lsDeployment")
	if err != nil {
		log.Panic("server url error :", err.Error())
	}
	rep, err := utils.HttpPostToken[lsDeploymentInfo](Url.String(), jsonByte, &saveLoginInfo.Token)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(*rep.AppName + " deployments:")
	titles := []string{"DeploymentName", "AppVersion", "Active", "Failed", "Installed"}
	if showKey {
		titles = append(titles, "DeploymentKey")
	}
	table, err := gotable.Create(titles...)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if rep.Deployments != nil {
		for _, v := range *rep.Deployments {
			columns := []string{*v.DeploymentName, v.AppVersion, strconv.Itoa(v.Active), strconv.Itoa(v.Failed), strconv.Itoa(v.Installed)}
			if showKey {
				columns = append(columns, v.DeploymentKey)
			}
			table.AddRow(columns)
		}
	}

	fmt.Println(table)
}

type deleteAppReq struct {
	AppName *string `json:"appName" binding:"required"`
}

func (App) deleteApp() {

	saveLoginInfo, err := utils.GetLoginfo()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	var appName string

	flag.StringVar(&appName, "n", "", "AppName")
	flag.Parse()
	if appName == "" {
		fmt.Println("Usage: code-push-go app delete_app -n <AppName>")
		return
	}

	deleteAppReq := deleteAppReq{
		AppName: &appName,
	}
	jsonByte, _ := json.Marshal(deleteAppReq)
	Url, err := url.Parse(saveLoginInfo.ServerUrl + "/delApp")
	if err != nil {
		log.Panic("server url error :", err.Error())
	}
	reqStatus, err := utils.HttpPostToken[constants.RespStatus](Url.String(), jsonByte, &saveLoginInfo.Token)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if reqStatus.Success {
		fmt.Println("Delete app " + appName + " success")
	}

}

type deleteDeploymentReq struct {
	AppName    *string `json:"appName" binding:"required"`
	Deployment *string `json:"deployment" binding:"required"`
}

func (App) deleteDeployment() {
	saveLoginInfo, err := utils.GetLoginfo()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	var deploymentName string
	var appName string
	flag.StringVar(&appName, "n", "", "AppName")

	flag.StringVar(&deploymentName, "dn", "", "DeploymentName")

	flag.Parse()
	if deploymentName == "" || appName == "" {
		fmt.Println("Usage: code-push-go app delete_deployment -n <AppName> -dn <DeploymentName>")
		return
	}
	deleteDeploymentReq := deleteDeploymentReq{
		AppName:    &appName,
		Deployment: &deploymentName,
	}

	jsonByte, _ := json.Marshal(deleteDeploymentReq)
	Url, err := url.Parse(saveLoginInfo.ServerUrl + "/delDeployment")
	if err != nil {
		log.Panic("server url error :", err.Error())
	}
	reqStatus, err := utils.HttpPostToken[constants.RespStatus](Url.String(), jsonByte, &saveLoginInfo.Token)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if reqStatus.Success {
		fmt.Println("Delete deployment " + deploymentName + " success")
	}
}

type rollbackReq struct {
	AppName    *string `json:"appName" binding:"required"`
	Deployment *string `json:"deployment" binding:"required"`
	Version    *string `json:"version" binding:"required"`
}

type rollbackRep struct {
	Success    *bool   `json:"success"`
	Version    *string `json:"version"`
	PackId     *int    `json:"packId"`
	Size       *int64  `json:"size"`
	Hash       *string `json:"hash"`
	CreateTime *int64  `json:"createTime"`
}

func (App) Rollback() {
	saveLoginInfo, err := utils.GetLoginfo()
	if err != nil {
		log.Println(err.Error())
		return
	}

	var targetVersion string
	var appName string
	var deployment string

	flag.StringVar(&targetVersion, "t", "", "Target version")
	flag.StringVar(&appName, "n", "", "AppName")
	flag.StringVar(&deployment, "d", "", "DeploymentName")
	flag.Parse()

	if appName == "" || deployment == "" || targetVersion == "" {
		fmt.Println("Usage: code-push-go rollback -n <AppName> -d <deployment> -t <TargetVersion>")
		return
	}

	rollbackReq := rollbackReq{
		AppName:    &appName,
		Deployment: &deployment,
		Version:    &targetVersion,
	}

	jsonByte, _ := json.Marshal(rollbackReq)
	Url, err := url.Parse(saveLoginInfo.ServerUrl + "/rollback")
	if err != nil {
		log.Panic("server url error :", err.Error())
	}
	reqStatus, err := utils.HttpPostToken[rollbackRep](Url.String(), jsonByte, &saveLoginInfo.Token)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if *reqStatus.Success {
		fmt.Println("Rollback " + deployment + " success,currut version:")
		titles := []string{"DeploymentName", "AppVersion", "PackId", "Size", "Hash", "CreateTime"}
		table, err := gotable.Create(titles...)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		var columns []string
		if reqStatus.PackId == nil {
			columns = []string{deployment, *reqStatus.Version}
		} else {
			columns = []string{deployment, *reqStatus.Version, strconv.Itoa(*reqStatus.PackId), strconv.FormatInt(*reqStatus.Size, 10), *reqStatus.Hash, time.Unix(0, *reqStatus.CreateTime*1e6).String()}
		}

		table.AddRow(columns)
		fmt.Println(table)

	}
}
