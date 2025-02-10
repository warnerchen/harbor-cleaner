FROM golang:1.23.1-alpine

WORKDIR /app

COPY . .

RUN go build -o harbor-cleaner main.go \
    && mv harbor-cleaner /usr/local/bin/harbor-cleaner

ENTRYPOINT ["harbor-cleaner"]