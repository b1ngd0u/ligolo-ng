export GO111MODULE=on

VERSION=$(shell date +"%Y.%m.%d")

BUILD=$(shell git rev-parse HEAD)
BASEDIR=./dist

LDFLAGS=-ldflags "-s -w -X main.build=${BUILD} -buildid=${BUILD}"
GCFLAGS=-gcflags=all=-trimpath=$(shell echo ${HOME})
ASMFLAGS=-asmflags=all=-trimpath=$(shell echo ${HOME})

GOFILES=`go list -buildvcs=false ./...`
GOFILESNOTEST=`go list -buildvcs=false ./... | grep -v test`

# Make Directory to store executables
$(shell mkdir -p ${BASEDIR})

# goreleaser build --config .goreleaser.yml --rm-dist --skip-validate
all: linux linux-arm64 windows windows-arm64
	@chmod +x dist/*

mac: lint
	@env CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} -o ${BASEDIR}/ligolo-ng-proxy-darwin_amd64 cmd/proxy/main.go
	@env CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} -o ${BASEDIR}/ligolo-ng-agent-darwin_amd64 cmd/agent/main.go

linux: lint
	@env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} -o ${BASEDIR}/ligolo-ng-proxy-linux_amd64 cmd/proxy/main.go
	@env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} -o ${BASEDIR}/ligolo-ng-agent-linux_amd64 cmd/agent/main.go

linux-arm64: lint
	@env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} -o ${BASEDIR}/ligolo-ng-proxy-linux_arm64 cmd/proxy/main.go
	@env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} -o ${BASEDIR}/ligolo-ng-agent-linux_arm64 cmd/agent/main.go

windows: lint
	@env CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} -o ${BASEDIR}/ligolo-ng-proxy-windows_amd64.exe cmd/proxy/main.go
	@env CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} -o ${BASEDIR}/ligolo-ng-agent-windows_amd64.exe cmd/agent/main.go

windows-arm64: lint
	@env CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -trimpath ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} -o ${BASEDIR}/ligolo-ng-proxy-windows_arm64.exe cmd/proxy/main.go
	@env CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -trimpath ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} -o ${BASEDIR}/ligolo-ng-agent-windows_arm64.exe cmd/agent/main.go

tidy:
	@go mod tidy

update: tidy
	@go get -v -d ./...
	@go get -u all

dep: ## Get the dependencies
	@go install github.com/goreleaser/goreleaser
	@go install github.com/securego/gosec/v2/cmd/gosec@latest

lint: ## Lint the files
	@env CGO_ENABLED=0 go fmt ${GOFILES}
	@env CGO_ENABLED=0 go vet ${GOFILESNOTEST}

security:
	@gosec -tests ./...

release:
	@goreleaser release --config .github/goreleaser.yml

clean:
	@rm -rf ${BASEDIR}

terminal_proxy:
	go run cmd/proxy/main.go -selfcert

terminal_agent:
	go run cmd/agent/main.go -connect localhost:11601 -ignore-cert

# Optional Tailscale feature-omission build tags synced from github.com/tailscale/tailcat v0.7.0 build-tags.txt
# Reduces binary size (~16%) without affecting tailcat data-plane functionality.
TAILCAT_TAGS=ts_omit_ace,ts_omit_acme,ts_omit_advertiseexitnode,ts_omit_advertiseroutes,ts_omit_appconnectors,ts_omit_aws,ts_omit_bird,ts_omit_cachenetmap,ts_omit_captiveportal,ts_omit_capture,ts_omit_cliconndiag,ts_omit_clientmetrics,ts_omit_clientupdate,ts_omit_cloud,ts_omit_colorable,ts_omit_completion,ts_omit_completion_scripts,ts_omit_conn25,ts_omit_connreject,ts_omit_debug,ts_omit_debugeventbus,ts_omit_debugportmapper,ts_omit_desktop_sessions,ts_omit_dns,ts_omit_dnsresolvecache,ts_omit_doctor,ts_omit_drive,ts_omit_favorites,ts_omit_flashappliance,ts_omit_health,ts_omit_hujsonconf,ts_omit_identityfederation,ts_omit_ipnbus,ts_omit_iptables,ts_omit_kube,ts_omit_linkspeed,ts_omit_linuxdnsfight,ts_omit_listenrawdisco,ts_omit_logtail,ts_omit_netlog,ts_omit_networkmanager,ts_omit_oauthkey,ts_omit_osrouter,ts_omit_outboundproxy,ts_omit_peerapiclient,ts_omit_peerapiserver,ts_omit_portlist,ts_omit_portmapper,ts_omit_posture,ts_omit_qrcodes,ts_omit_relayserver,ts_omit_remoteconfig,ts_omit_resolved,ts_omit_routecheck,ts_omit_runtimemetrics,ts_omit_sdnotify,ts_omit_serve,ts_omit_serviceclientprefs,ts_omit_synology,ts_omit_syslog,ts_omit_syspolicy,ts_omit_systray,ts_omit_taildrop,ts_omit_tailnetlock,ts_omit_tap,ts_omit_tpm,ts_omit_tundevstats,ts_omit_unixsocketidentity,ts_omit_useexitnode,ts_omit_useproxy,ts_omit_usermetrics,ts_omit_useroutes,ts_omit_wakeonlan,ts_omit_webbrowser,ts_omit_webclient

linux-tailcat-slim: lint
	@env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags "${TAILCAT_TAGS}" -trimpath ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} -o ${BASEDIR}/ligolo-ng-proxy-linux_amd64 cmd/proxy/main.go
	@env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags "${TAILCAT_TAGS}" -trimpath ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} -o ${BASEDIR}/ligolo-ng-agent-linux_amd64 cmd/agent/main.go

.PHONY: all linux linux-arm64 windows windows-arm64 linux-tailcat-slim tidy update dep lint security release clean terminal
