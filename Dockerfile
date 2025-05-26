# build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN go build -o injective ./cmd/injective

# final stage
FROM alpine:latest
WORKDIR /app

COPY --from=builder /app/injective .
COPY --from=builder /app/frontend ./frontend

EXPOSE 8080

# CoinDesk API Key and URL
ENV COINDESK_API_KEY=420fcfea38b151afe3f39c356bece6400ac97ea2ec2a4293e14392eaab15af7f
ENV COINDESK_API_URL=https://data-api.coindesk.com/index/cc/v1/latest/tick?market=ccix&instruments=BTC-USD&api_key=%s

CMD ["./injective"]