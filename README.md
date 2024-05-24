# code-push-go
Code-push-go cli used with [code-push-server-go](https://github.com/htdcx/code-push-server-go.git), only support react native

# Install
``` shell
git clone https://github.com/htdcx/code-push-go.git
cd code-push-go

#MacOS build GOOS:windows,darwin
CGO_ENABLED=0 GOOS=linux  GOARCH=amd64 go build -o code-push-go(.exe) main.go

#Windows build
set GOARCH=amd64
set GOOS=linux #windows,darwin
go build -o code-push-go(.exe) main.go
mv code-push-go(.exe) <You project>

#Linux
chmod +x code-push-go

#Login
./code-push-go login -u <userName> -p <password> -h <serverUrl>

```

# Use
``` shell
./code-push-go app create_app -n <AppName> -os <ios or android>
./code-push-go app create_deployment -n <AppName> -dn <DeploymentName>

#Update react native
./code-push-go create_bundle -t <TargetVersion> -n <AppName> -d <DeploymentName> -p <(*Optional) React native project default:./> --description  <(*Optional) Description default: ""/>

#More command
./code-push-go
```

## License
MIT License [Read](https://github.com/htdcx/code-push-go/blob/main/LICENSE)
