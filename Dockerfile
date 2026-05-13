# Stage 1: 編譯階段 (使用 Golang 環境)
FROM golang:1.21-bookworm AS builder

# 設定作業系統環境
WORKDIR /build

# 設定 Go 模組下載快取目錄 (優化建置時間)
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org

# 複製依賴檔案並下載 (僅下載，不複製程式碼)
COPY go.mod go.sum* ./
RUN go mod download

# 複製程式碼
COPY . .

# 建置可執行檔
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/bin/server ./cmd

# Stage 2: 生產階段 (最小化映像)
FROM alpine:latest

# 安裝必要系統套件 (如需要 curl 用於 health check)
RUN apk --no-cache add ca-certificates tzdata

# 設定時區為台灣
ENV TZ=Asia/Taipei
RUN apk add --no-cache bash

# 設定工作目錄
WORKDIR /app

# 建立非 root 使用者 (安全最佳實踐)
RUN addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup

# 從 builder 階段複製可執行檔
COPY --from=builder --chown=appuser:appgroup /build/bin/server /app/server

# 複製配置文件 (如需要)
# COPY --chown=appuser:appgroup config.yaml /app/config.yaml

# 設定啟動使用者
USER appuser

# 暴露 HTTP 通訊埠
EXPOSE 8080

# 啟動命令
CMD ["./server"]
