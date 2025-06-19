# 📬 Message Queue API Specification

## 🧠 Control Plane APIs
Manage queues, permissions, quotas, and configurations.

### Queue Management

| Method   | Endpoint           | Description                         |
|----------|--------------------|-------------------------------------|
| `GET`    | `/queues`          | List all queues                     |
| `POST`   | `/queues`          | Create a new queue                  |
| `GET`    | `/queues/{id}`     | Get metadata/details of a queue     |
| `PUT`    | `/queues/{id}`     | Update queue configuration          |
| `DELETE` | `/queues/{id}`     | Delete a queue                      |


### Queue Operations

| Method | Endpoint                      | Description                        |
|--------|-------------------------------|------------------------------------|
| `GET`  | `/queues/{id}/stats`          | Get queue metrics/statistics       |
| `POST` | `/queues/{id}/purge`          | Purge all messages in the queue    |

### Access Control

| Method | Endpoint                            | Description                              |
|--------|-------------------------------------|------------------------------------------|
| `POST` | `/queues/{id}/permissions`          | Set access control policies (IAM)        |
| `GET`  | `/queues/{id}/permissions`          | List access control policies             |


### Quotas

| Method | Endpoint           | Description                         |
|--------|--------------------|-------------------------------------|
| `GET`  | `/quotas`          | View current quotas and usage       |
| `POST` | `/quotas`          | Set or update quotas                |

## 📦 Data Plane APIs
Used to push, pull, acknowledge, and inspect messages.

### Message Operations

| Method   | Endpoint                                    | Description                                 |
|----------|---------------------------------------------|---------------------------------------------|
| `POST`   | `/queues/{id}/messages`                     | Enqueue a message                           |
| `GET`    | `/queues/{id}/messages`                     | Receive messages (optionally with peek lock)|
| `DELETE` | `/queues/{id}/messages/{msg_id}`            | Acknowledge and delete a message            |
| `POST`   | `/queues/{id}/messages/{msg_id}/renew-lock` | Renew lock on a peeked message              |
| `POST`   | `/queues/{id}/messages/batch`               | Enqueue a batch of messages                 |

### Dead Letter Queue (DLQ)

| Method   | Endpoint                                | Description                           |
|----------|-----------------------------------------|---------------------------------------|
| `POST`   | `/queues/{id}/deadletter/{msg_id}`      | Move message to DLQ                   |
| `GET`    | `/queues/{id}/deadletter`               | List DLQ messages                     |
| `DELETE` | `/queues/{id}/deadletter/{msg_id}`      | Delete message from DLQ               |

### Replay

| Method | Endpoint                | Description                           |
|--------|-------------------------|---------------------------------------|
| `POST` | `/queues/{id}/replay`   | Replay messages from DLQ/archive      |

---
## 🩺 Health & Telemetry

| Method | Endpoint      | Description                        |
|--------|---------------|------------------------------------|
| `GET`  | `/healthz`    | Liveness probe                     |
| `GET`  | `/readyz`     | Readiness probe                    |
| `GET`  | `/metrics`    | Prometheus metrics endpoint        |
| `GET`  | `/traces`     | Trace view endpoint (optional)     |

---
---
### Purge Queue

| Method | POST |
|--------|------|
| Endpoint | `/queues/{id}/purge` |
| Description | Delete all messages in the queue |

#### Response (202 Accepted)

```json
{
  "status": "purging",
  "queueId": "my-queue"
}
```

---

## 📦 Data Plane APIs

### Enqueue Message

| Method | POST |
|--------|------|
| Endpoint | `/queues/{id}/messages` |
| Description | Enqueue a new message to the queue |

#### Request Headers

```http
Content-Type: application/json
Authorization: Bearer <token>
```

#### Request Body

```json
{
  "payload": {
    "eventType": "UserSignup",
    "data": {
      "userId": "abc123",
      "email": "user@example.com"
    }
  },
  "delaySeconds": 10,
  "attributes": {
    "correlationId": "req-879213"
  }
}
```

#### Response (202 Accepted)

```json
{
  "messageId": "msg-0023810",
  "status": "enqueued"
}
```

---

### Receive Message (Peek-Lock)

| Method | GET |
|--------|-----|
| Endpoint | `/queues/{id}/messages?maxMessages=1&visibilityTimeout=30s` |
| Description | Fetch message(s) with optional lock |

#### Response (200 OK)

```json
[
  {
    "messageId": "msg-0023810",
    "payload": {
      "eventType": "UserSignup",
      "data": {
        "userId": "abc123",
        "email": "user@example.com"
      }
    },
    "receivedAt": "2025-06-15T12:05:00Z",
    "lockToken": "lock-7491ab"
  }
]
```

---

### Acknowledge Message

| Method | DELETE |
|--------|--------|
| Endpoint | `/queues/{id}/messages/{msg_id}` |
| Description | Acknowledge and remove a message from the queue |

#### Request Headers

```http
Authorization: Bearer <token>
Lock-Token: lock-7491ab
```

#### Response (204 No Content)

No body.

---

### Renew Lock

| Method | POST |
|--------|------|
| Endpoint | `/queues/{id}/messages/{msg_id}/renew-lock` |
| Description | Renew the lock on a peeked message |

#### Request Headers

```http
Authorization: Bearer <token>
Lock-Token: lock-7491ab
```

#### Response (200 OK)

```json
{
  "messageId": "msg-0023810",
  "lockExpiration": "2025-06-15T12:06:00Z"
}
```

---

### Health & Metrics

#### Liveness Check

| Method | GET |
|--------|-----|
| Endpoint | `/healthz` |

#### Response (200 OK)

```text
OK
```

#### Prometheus Metrics

| Method | GET |
|--------|-----|
| Endpoint | `/metrics` |
| Content-Type | `text/plain; version=0.0.4` |

---

## 📝 Common Headers

| Header | Description |
|--------|-------------|
| `Authorization` | Bearer token or mTLS identity |
| `Content-Type` | Usually `application/json` |
| `Accept` | Usually `application/json` |
| `Lock-Token` | Used in message deletion and renewal |

---

## 🔐 Authentication

All control and data plane endpoints **require authentication**:
- Recommended: **Bearer token** in `Authorization` header
- Alternative: **mTLS** with client certificates
