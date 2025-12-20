# 1. Go Builder Stage
FROM golang:1.24-alpine AS go-builder
RUN apk add --no-cache gcc musl-dev git sqlite-dev
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download || (go mod init impossible-bot && go mod tidy)
COPY . .
RUN go build -ldflags="-s -w" -o bot .

# 2. Node.js Builder Stage
FROM node:20-alpine AS node-builder
RUN apk add --no-cache git 
WORKDIR /app
COPY package*.json ./
COPY lid-extractor.js ./
RUN npm install --production

# 3. Final Runtime Image
FROM alpine:latest
RUN apk add --no-cache \
    ca-certificates \
    sqlite-libs \
    nodejs \
    npm \
    && rm -rf /var/cache/apk/*
WORKDIR /app
COPY --from=go-builder /app/bot ./bot
COPY --from=node-builder /app/node_modules ./node_modules
COPY --from=node-builder /app/lid-extractor.js ./lid-extractor.js
COPY package.json ./package.json
COPY web ./web
COPY pic.png ./pic.png
RUN mkdir -p store logs
ENV PORT=8080
ENV NODE_ENV=production
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1
CMD ["./bot"]
