package opt

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"syscall"

	"com.lc.go.codepush/client/constants"
	"com.lc.go.codepush/client/utils"
	"golang.org/x/term"
)

type User struct{}

type loginUser struct {
	UserName *string `json:"userName"`
	Password *string `json:"password"`
}

func (User) Login() {
	var userName string
	var password string
	var serverUrl string
	flag.StringVar(&userName, "u", "", "UserName")
	flag.StringVar(&password, "p", "", "Password")
	flag.StringVar(&serverUrl, "h", "", "ServerUrl")
	flag.Parse()
	if userName == "" || password == "" || serverUrl == "" {
		fmt.Println("Usage: code-push-go login -u <UserName> -p <Password> -h <ServerUrl>")
		return
	}
	passwordMd5 := utils.MD5(password)
	loginUser := loginUser{
		UserName: &userName,
		Password: &passwordMd5,
	}
	Url, err := url.Parse(serverUrl + "/login")
	if err != nil {
		log.Panic("server url error :", err.Error())
	}
	jsonByte, _ := json.Marshal(loginUser)

	respLogin, err := utils.HttpPost[respLogin](Url.String(), jsonByte)
	if err != nil {
		fmt.Println(err)
		return
	}
	saveLoginInfo := constants.SaveLoginInfo{
		Token:     respLogin.Token,
		ServerUrl: serverUrl,
	}
	bytes, _ := json.Marshal(saveLoginInfo)
	f, err := os.Create("./.code-push-go.json")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	_, err = f.Write(bytes)
	if err != nil {
		fmt.Println(err)
		return
	}
	log.Println("Login success")
}

type changePasswordReq struct {
	Password *string `json:"password" binding:"required"`
}

func (User) ChangePassword() {
	saveLoginInfo, err := utils.GetLoginfo()
	if err != nil {
		log.Println(err.Error())
		return
	}
	fmt.Print("Enter new password: ")
	bytepw, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		os.Exit(1)
	}
	pass := string(bytepw)
	fmt.Println()

	fmt.Print("Again enter password: ")
	bytepw, err = term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		os.Exit(1)
	}
	pass2 := string(bytepw)
	fmt.Println()

	if pass != pass2 {
		log.Panic("Passwords are inconsistent!")
	}
	passwordMd5 := utils.MD5(pass)

	checkBundleReq := changePasswordReq{
		Password: &passwordMd5,
	}
	Url, err := url.Parse(saveLoginInfo.ServerUrl + "/changePassword")
	if err != nil {
		log.Panic("server url error :", err.Error())
	}

	jsonByte, _ := json.Marshal(checkBundleReq)
	reqStatus, err := utils.HttpPostToken[constants.RespStatus](Url.String(), jsonByte, &saveLoginInfo.Token)
	if err != nil {
		fmt.Println(err)
		return
	}
	if reqStatus.Success {
		fmt.Println("Change password success")
	}
}

func (User) Logout() {
	os.Remove("./.code-push-go.json")
	fmt.Println("Logout success")
}

type respLogin struct {
	Token string `json:"token"`
}
