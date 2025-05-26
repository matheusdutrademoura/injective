
# Injective - Go Developer Home Assignment

This is my implementation of the **Go Developer Home Assignment** for the Injective team.  
It’s a live Bitcoin price streamer built with Go, SSE, and a sprinkle of brain-frying intensity.

## 🧠 What is this?

A backend service that:

- Periodically fetches the BTC/USD price using the [CoinDesk API](https://developers.coindesk.com/documentation/data-api/introduction)
- Buffers updates in a ring buffer with TTL expiration
- Broadcasts live prices to connected clients via **Server-Sent Events (SSE)**
- Serves a minimal frontend (`index.html`) that listens to `/stream`

Built with:

- Go (no frameworks)
- Vanilla HTML/JS for the frontend
- SSE (no WebSockets here!)
- Mutexes and channels — the Go way

> 🥵 Yes, this assignment made me sweat a little — but I'm proud of the result.  
> Hopefully it streams not only BTC prices, but also some good karma into my inbox. 😄

## Running Locally

### Prerequisites

- Go 1.20+
- A valid [CoinDesk API key](https://www.coindesk.com/coindesk-api)

### 1. Build

```bash
docker build -t injective . 
```

### 2. Run the server

```bash
docker run -p 8080:8080 injective
```

By default, the frontend will be available at:

```
http://localhost:8080/
```

And the SSE stream will be available at:

```
http://localhost:8080/stream
```

## 🧪 Running Tests

```bash
go test ./...
```
Covers:

- Ring buffer behavior
- SSE client management
- HTTP fetching with mocked APIs

## 🏎️ Race Condition Detection
Some tests are designed to check for race conditions in concurrent code (e.g., RingBuffer and ClientManager). To run all tests with the Go race detector enabled, use:

```bash
go test -race ./...
```

## 🛰 Example: Consume `/stream` with `curl`

You can subscribe to the live stream using any SSE-compatible client:

```bash
curl http://localhost:8080/stream
```

You can also pass a timestamp (in Unix seconds) to fetch missed updates:

```bash
curl "http://localhost:8080/stream?since=1716732900"
```

## 📦 Project Structure

```
.
├── cmd/injective        # Entry point
├── internal/            # Internal packages
│   ├── client           # SSE clients
│   ├── fetcher          # Price fetcher
│   ├── models           # Data models
│   ├── ringbuffer       # TTL-based circular buffer
│   └── server           # HTTP logic and orchestration
├── frontend/live.html   # Very minimalist UI
└── tests                # (Optional) unit tests live here
```

## 🏗️ Part 3 - Production Readiness

### Scaling to 10,000+ Users

- Run multiple instances of the SSE server behind a load balancer to handle many concurrent users.
- Use one dedicated fetcher to get BTC prices from the external API and publish updates to a central messaging system (e.g. Redis Pub/Sub).
- All SSE servers subscribe to this channel and broadcast updates to their clients.
- This avoids repeated API calls and keeps data consistent.
- Autoscale server instances based on demand.

### Reliability & Fault-Tolerance

- Add retries with backoff in the price fetcher when calling the external API.
- Handle client disconnects gracefully on the SSE servers.
- Run multiple replicas with health checks and auto-restart (e.g. Kubernetes) to avoid single points of failure.
- Use the centralized messaging layer to decouple components and limit cascading failures.

### Observability

- Implement structured logging for key events (connections, errors, broadcasts).
- Collect metrics (e.g. Prometheus) on client count, broadcast delay, and API errors.
- Use tracing to follow data flow if possible.
- Set alerts to catch errors or slowdowns early.