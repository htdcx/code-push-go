package main

import (
	"fmt"
	"os"

	"com.lc.go.codepush/client/opt"
)

func main() {
	fmt.Println("code-push-go V1.0.2")

	var args []string
	var notargs []string
	var in_flags bool = false
	for i := 0; i < len(os.Args); i++ {
		if os.Args[i][0] == '-' {
			in_flags = true
		}
		if i == 0 || in_flags {
			notargs = append(notargs, os.Args[i])
		} else {
			args = append(args, os.Args[i])
		}
	}
	os.Args = notargs
	help := "Usage: code-push-go <command>\n" +
		"Commands:\n" +
		"	login            Authenticate in order to begin managing your apps\n" +
		"	logout           Log out of the current session\n" +
		"	app              View and manage your CodePush apps\n" +
		"	create_bundle    Create react native hotfix bundle\n" +
		"	rollback         Rollback last dundle"

	var command string
	if len(args) <= 0 {
		fmt.Println(help)
		return
	}
	command = args[0]

	switch command {
	case "login":
		opt.User{}.Login()
	case "logout":
		opt.User{}.Logout()
	case "create_bundle":
		opt.App{}.CreateBundle()
	case "app":
		opt.App{}.App(args)
	case "rollback":
		opt.App{}.Rollback()
	default:
		fmt.Println(help)
		return
	}

}
