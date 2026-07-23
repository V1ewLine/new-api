# 在运行容器中编译和启动项目

本文适用于已经进入运行容器、无法使用 `docker` 命令，并且项目源码位于
`/data/new-api` 的场景。

运行镜像通常只包含预编译的 `/new-api`，不包含 Bun 和 Go。因此，第一次编译
需要先安装构建工具。安装在容器系统目录中的工具可能会在容器重建后丢失。

## 1. 第一次编译

以下命令需要使用 `root` 用户执行。

### 1.1 安装基础依赖

```bash
apt-get update
apt-get install -y --no-install-recommends curl unzip
```

### 1.2 安装 Bun

```bash
curl -fsSL https://bun.com/install | bash

export BUN_INSTALL=/root/.bun
export PATH="$BUN_INSTALL/bin:$PATH"

bun --version
```

Bun 的 Linux 安装脚本需要 `unzip`。安装完成后，Bun 默认位于
`/root/.bun/bin`。

### 1.3 安装 Go

下面的命令安装 Go 1.26.5，并根据容器 CPU 架构选择对应的安装包：

```bash
case "$(uname -m)" in
  x86_64)
    GO_ARCH=amd64
    GO_SHA256=5c2c3b16caefa1d968a94c1daca04a7ca301a496d9b086e17ad77bb81393f053
    ;;
  aarch64|arm64)
    GO_ARCH=arm64
    GO_SHA256=fe4789e92b1f33358680864bbe8704289e7bb5fc207d80623c308935bd696d49
    ;;
  *)
    echo "不支持的架构：$(uname -m)"
    exit 1
    ;;
esac

wget -O /tmp/go.tar.gz \
  "https://go.dev/dl/go1.26.5.linux-${GO_ARCH}.tar.gz"

echo "${GO_SHA256}  /tmp/go.tar.gz" | sha256sum -c -

mkdir -p /opt/go1.26.5
tar -C /opt/go1.26.5 --strip-components=1 -xzf /tmp/go.tar.gz

export PATH="/opt/go1.26.5/bin:$PATH"

go version
```

### 1.4 编译前端

```bash
cd /data/new-api/web

bun install --frozen-lockfile

DISABLE_ESLINT_PLUGIN=true \
VITE_REACT_APP_VERSION="$(cat ../VERSION)" \
bun run build
```

前端构建产物会写入 `/data/new-api/web/dist`，后端编译时会将这些文件嵌入
最终二进制。

### 1.5 编译后端

```bash
cd /data/new-api

mkdir -p bin

CGO_ENABLED=0 \
GOEXPERIMENT=greenteagc \
go build \
  -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=$(cat VERSION)'" \
  -o ./bin/new-api-local \
  .

ls -lh ./bin/new-api-local
```

最终程序位于：

```text
/data/new-api/bin/new-api-local
```

## 2. 后续编译

只要容器没有被重建，Bun 和 Go 通常仍然存在。每次打开新的终端后，先恢复
环境变量：

```bash
export BUN_INSTALL=/root/.bun
export PATH="/opt/go1.26.5/bin:$BUN_INSTALL/bin:$PATH"

bun --version
go version
```

然后重新编译前端和后端：

```bash
cd /data/new-api/web

# package.json 或 bun.lock 没有变化时，这一步通常会很快完成。
bun install --frozen-lockfile

DISABLE_ESLINT_PLUGIN=true \
VITE_REACT_APP_VERSION="$(cat ../VERSION)" \
bun run build

cd /data/new-api

mkdir -p bin

CGO_ENABLED=0 \
GOEXPERIMENT=greenteagc \
go build \
  -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=$(cat VERSION)'" \
  -o ./bin/new-api-local \
  .

ls -lh ./bin/new-api-local
```

如果只修改了 Go 后端，并且不需要更新嵌入的前端资源，可以只执行后端编译
部分。

## 3. 编译后运行

### 3.1 检查当前进程

启动前先确认镜像自带的程序是否已经占用端口：

```bash
ps -p 1 -o pid,args
ps aux | grep '[n]ew-api'
```

如果没有其他程序占用 3000 端口，可以直接启动：

```bash
cd /data/new-api
exec ./bin/new-api-local --port 3000 --log-dir /data/logs
```

`exec` 会使用新程序替换当前 Shell，适合将当前 Shell 作为正式运行进程。

### 3.2 使用其他端口测试

如果镜像自带的 `/new-api` 已经占用 3000 端口，可以先使用 3001 端口测试：

```bash
cd /data/new-api
./bin/new-api-local --port 3001 --log-dir /data/logs
```

也可以在后台运行：

```bash
mkdir -p /data/logs

nohup /data/new-api/bin/new-api-local \
  --port 3001 \
  --log-dir /data/logs \
  >/data/logs/new-api-local.out 2>&1 &
```

查看启动日志：

```bash
tail -f /data/logs/new-api-local.out
```

### 3.3 PID 1 限制

如果 `ps -p 1 -o pid,args` 显示 PID 1 是镜像自带的 `/new-api`，不要直接终止
PID 1。PID 1 退出后，整个容器通常也会停止。

要让本地编译的程序正式监听 3000 端口，需要在 BitaHub 的实例启动配置中将
启动命令改为：

```bash
/data/new-api/bin/new-api-local --port 3000 --log-dir /data/logs
```

如果 BitaHub 只向外开放了 3000 端口，那么在 3001 端口启动的程序只能用于
容器内部测试，除非同时在 BitaHub 中开放或转发 3001 端口。

## 参考资料

- [Bun 安装文档](https://bun.com/docs/installation)
- [Go 官方下载页](https://go.dev/dl/)
