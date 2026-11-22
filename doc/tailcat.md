# Tailcat Transport (`tailcat://`) 使用與進版銜接指南

## 使用方式

### 1. Reverse 模式（Proxy 監聽、Agent 主動連線）
無須 `-selfcert`、`-certfile`、`-ignore-cert` 或公網 Port Forwarding，直接使用 Tailcat 原生 WireGuard + 256-bit PSK 加密。

```bash
# Proxy 端（預設虛擬埠 11601，可改為 tailcat://:443）
./proxy -laddr tailcat://
# 輸出：Listening tailcat on tailcat://tc...
# （於互動式 CLI 可隨時輸入 tailcat_addr 重新查詢位址）

# Agent 端
./agent -connect tailcat://tc...
```

### 2. Bind 模式（Agent 監聽、Proxy 主動連線）
```bash
# Agent 端
./agent -bind tailcat://
# 輸出：Listening tailcat on tailcat://tc...

# Proxy CLI 端
ligolo-ng » connect_agent --ip tailcat://tc...
```

### 3. 進階參數
- `-tailcat-derpmap <URL>`：指定自訂 DERP Map JSON URL（例如自架 `derper`）。
- `-tailcat-region <ID>`：指定監聽端偏好的 DERP Region ID（`0` 為自動依延遲挑選最近節點）。
- `-tailcat-derp-only`：強制所有 Tailcat 流量皆經由 DERP 中繼，不進行直連 UDP 升級（Proxy 與 Agent 皆支援，等同 `TS_DEBUG_ALWAYS_USE_DERP=true`）。
- `-v`：開啟 debug 模式時會一併輸出 `[tailcat]` 內部連線與 NAT 穿透追蹤日誌。

> **自動 TUN 路由迴圈防護與低 MTU 穩定性**：
> - Agent 啟用 `tailcat://` 時會自動追蹤自身的 `magicsock` 本地 UDP 端點並拒絕經由 Ligolo 轉發 UDP 至該端點，避免 Proxy 將 Agent 所在內網網段加入 `ligolo` TUN 路由時，觸發 `magicsock` 誤將 WireGuard 通道切進 TUN 造成的自循環斷線；同時保留經由真實 LAN/WAN 升級高速直連 P2P 的能力。
> - 為避免在低 MTU 路徑（MTU < 1440，如 PPPoE、Tailscale、VPN 或巢狀隧道）觸發 Linux UDP GSO 黑洞卡死（`tailscale/tailcat#168`），`pkg/tailcatutils` 預設會套用 `TS_DEBUG_DISABLE_UDP_GSO=true`；若在確定 MTU 1500 的環境下欲手動啟用 UDP GSO，可顯式設定 `TS_DEBUG_DISABLE_UDP_GSO=false`。

### 4. 縮小執行檔體積（選用）
```bash
make linux-tailcat-slim
```

---

## 未來進版銜接 SOP

### 一、`ligolo-ng` 上游進版
本功能實作於獨立分支 `feat/tailcat`，並將絕大多數程式碼隔離於新增檔案（`pkg/tailcatutils/`、`*_tailcat.go`、`cmd/proxy/app/tailcat.go`），僅在 5 個上游既有檔案保留最小掛鉤點：
- `pkg/utils/ligolourl.go`（`IsValid()`）
- `pkg/controller/controller.go`（`ControllerConfig`、`ListenAndServe()`）
- `cmd/proxy/main.go`（CLI flags 與 `ControllerConfig`）
- `cmd/proxy/app/app.go`（`registerTailcatCommands()` 與 `connect_agent` 分流）
- `cmd/agent/main.go`（CLI flags 與 `IsTailcat()` 分流）
- `pkg/agent/handler.go`（5 行 UDP 過濾掛鉤，實作於獨立檔案 `pkg/agent/udp_filter.go`）
- `pkg/proxy/netstack/tun/wireguard_ep.go`（因 `tailscale.com` 透過 MVS 帶入新版 `golang.zx2c4.com/wireguard`，適配非 Linux 平台的 batched `Read`/`Write` API；若上游未來自行升級 wireguard，rebase 時直接採用上游即可）

上游進版同步流程：
```bash
git fetch upstream
git checkout feat/tailcat
git rebase upstream/master
go mod tidy
go test ./...
go vet ./...
```

### 二、`github.com/tailscale/tailcat` 進版（例如 `v0.7.0` → `v0.8.0+`）
全專案**僅 `pkg/tailcatutils/`** 匯入 `github.com/tailscale/tailcat` 與 `tailscale.com/...`，其餘模組均只依賴 Go 標準庫 `net.Listener` / `net.Conn`。

升級 `tailcat` 流程：
1. 確認目標版本 `tailcat` 的 `go.mod` 所需最低 Go 版本，以及共用相依套件（如 `golang.zx2c4.com/wintun`、`golang.zx2c4.com/wireguard`、`github.com/coder/websocket`）是否相容。
2. 更新版本：`go get github.com/tailscale/tailcat@<new-version> && go mod tidy`。
3. 僅需檢視並調整單一適配層 [pkg/tailcatutils/tailcat.go](file:///home/kali/_Neon/ligolo-ng/pkg/tailcatutils/tailcat.go)（例如上游 API 調整 `AllowedClients` → `AllowClient`、或欲啟用新版 `Hybrid` 後量子握手 / `HTTPS_PROXY` 支援）。
4. 若使用 `make linux-tailcat-slim`，同步更新 `Makefile` 中 `TAILCAT_TAGS`（對照新版 `tailcat` 儲存庫的 `build-tags.txt`）。
5. 驗證測試與多平台交叉編譯：
   ```bash
   go test -v ./pkg/utils/... ./pkg/tailcatutils/... ./pkg/controller/...
   CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./cmd/proxy ./cmd/agent
   CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./cmd/proxy ./cmd/agent
   ```
