# Modular Rate-Limiting System

A production-grade, protocol-agnostic rate-limiting system built with Go and [GoFr](https://gofr.dev/). It supports both **HTTP** and **gRPC** protocols, uses a high-performance **Token Bucket** algorithm, and integrates with **Kafka** for event-driven workflows.

## 🚀 Features

- **Dual-Protocol Support**: Shared business logic accessible via REST (HTTP) and gRPC.
- **Advanced Rate Limiting**: Pluggable algorithm support (currently implements Lazy Refill Token Bucket).
- **Event-Driven**: Automatically publishes `user-created` events to Kafka.
- **Protocol Isolation**: Clean separation between transport layers and domain logic.
- **Interactive Docs**: Swagger UI (REST), AsyncAPI (Events), and gRPC Reflection.
- **Service Discovery**: gRPC Reflection enabled for easy discovery.

## 📂 Project Structure

```text
.
├── api/                # API Contracts
│   ├── openapi/        # REST Definitions (OpenAPI)
│   ├── asyncapi/       # Event Definitions (AsyncAPI)
│   └── proto/          # gRPC Definitions (Protobuf)
├── cmd/
│   └── server/         # Application Entry Point
├── internal/
│   ├── api/            # Protocol Adapters
│   │   ├── http/       # REST Handlers & Routers
│   │   └── grpc/       # gRPC Handlers & Interceptors
│   ├── domain/         # Domain Entities & Events
│   ├── service/        # Business Logic (Use Cases)
│   ├── store/          # Persistence (Postgres)
│   └── migrations/     # Database Migrations (User, Outbox)
├── pkg/                # Exportable Shared Libraries
│   └── ratelimit/      # Rate-Limit Library
├── static/             # Generated Swagger JSON
└── .data/              # Local Infrastructure Data (Ignored)
```

## 📜 API Contracts & Documentation

The system follows a **Contract-First** approach. All interfaces are defined before implementation:

- **REST (OpenAPI)**: [api/openapi/openapi.yaml](api/openapi/openapi.yaml)
- **Events (AsyncAPI)**: [api/asyncapi/asyncapi.yaml](api/asyncapi/asyncapi.yaml)
- **gRPC (Protobuf)**: [api/proto/user.proto](api/proto/user.proto)

> [!TIP]
> You can visualize the REST contracts using `make swagger`. For Events, copy [api/asyncapi/asyncapi.yaml](api/asyncapi/asyncapi.yaml) into [AsyncAPI Studio](https://studio.asyncapi.com/).

## 🛠️ Tech Stack

- **Framework**: GoFr (Opinionated Go Framework)
- **Database**: PostgreSQL
- **Messaging**: Apache Kafka
- **Documentation**: Swagger/OpenAPI 3.0
- **Communication**: Protobuf / gRPC

## ⚙️ Configuration

The system uses a `.env` file for all configurations. Key variables include:

| Variable | Description | Default |
|----------|-------------|---------|
| `APP_NAME` | Name of the service | `rate-limiter` |
| `PUBSUB_BACKEND` | Messaging backend | `kafka` |
| `PUBSUB_BROKER` | Kafka broker address | `localhost:9092` |
| `KAFKA_USER_TOPIC`| Kafka topic for user events | `system-design` |

## 🏁 Getting Started

### 1. Start Infrastructure
```bash
docker-compose -f docker-compose.local.yml up -d
```

### 2. Generate API Code & Docs
```bash
make generate   # gRPC code
make swagger    # REST docs (JSON sync)
```

### 3. Run the Server
```bash
go run cmd/server/main.go
```

## 🧪 Testing

### HTTP (REST)
```bash
# Create a User
curl -X POST http://localhost:8000/user \
     -H "Content-Type: application/json" \
     -d '{"username": "arjun", "email": "arjun@example.com"}'

# Get User Info (Rate limited)
curl -i http://localhost:8000/user/arjun
```

### gRPC
```bash
# Get User Info via gRPC
grpcurl -plaintext -d '{"user_id": "arjun"}' localhost:9000 user.UserService/GetUser
```

### Verify Kafka Events
You can use `kcat` or any Kafka client to listen for events:
```bash
kcat -b localhost:9092 -t system-design -C
```

## 📜 Architectural Patterns

1. **Functional Options**: Used in protocol routers (`WithRateLimiter`) for clean dependency injection.
2. **Lazy Refill**: The Token Bucket algorithm only updates state on request, ensuring $O(1)$ performance.
3. **Contract-First**: All APIs are defined in YAML/Proto before implementation.
4. **Clean Architecture**: Domain logic is completely independent of the protocol used to access it.
5. **Outbox Fallback**: If Kafka is unreachable, events are persisted to a DB `outbox` table for later reconciliation.
6. **Domain Events**: Events are defined as type-safe entities in the `domain` layer.
