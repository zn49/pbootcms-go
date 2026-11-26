FROM golang:1.27-alpine AS build
WORKDIR /src
ENV GOTOOLCHAIN=auto \
	GOPROXY=https://goproxy.cn,direct \
	GOSUMDB=sum.golang.google.cn
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/pbootcms-go ./cmd/pbootcms-go

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata su-exec \
	&& addgroup -S -g 10001 gboot \
	&& adduser -S -u 10001 -G gboot -H -D gboot \
	&& mkdir -p /app/site \
	&& chown -R gboot:gboot /app
WORKDIR /app

COPY --from=build /out/pbootcms-go /app/pbootcms-go
COPY docker/config.docker.json /app/config.json
COPY docker/entrypoint.sh /app/entrypoint.sh
# site tree (templates/static/skin); writable dirs created/fixed at start
COPY site/ /app/site/
RUN chmod 755 /app/pbootcms-go /app/entrypoint.sh \
	&& chmod 600 /app/config.json \
	&& rm -rf /app/site/data/cache /app/site/data/session /app/site/data/log \
	&& find /app/site/data -maxdepth 1 -type f \( -name '*.db' -o -name '*.sqlite' -o -name 'install.lock' -o -name '.app_salt' -o -name 'gboot-config.json' \) -delete \
	&& mkdir -p /app/site/data /app/site/static/upload /app/site/static/backup \
	&& chown -R gboot:gboot /app/site/data /app/site/static/upload /app/site/static/backup \
	&& chmod 750 /app/site/data /app/site/static/upload /app/site/static/backup \
	&& find /app/site -type d -exec chmod 755 {} \; \
	&& find /app/site -type f -exec chmod 644 {} \; \
	&& chmod 750 /app/site/data /app/site/static/upload /app/site/static/backup

ENV GBOOT_CONFIG=/app/site/data/gboot-config.json \
	TZ=Asia/Shanghai
EXPOSE 8080
# entrypoint starts as root only to chown volumes, then su-exec → gboot (uid 10001)
ENTRYPOINT ["/app/entrypoint.sh"]
