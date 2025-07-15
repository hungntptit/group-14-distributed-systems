# group-14-distributed-systems


chạy mỗi node với env variable


```bash
// bash
SELF_URL=http://localhost:8001 \
PORT=8001 \
PEERS=http://localhost:8001,http://localhost:8002,http://localhost:8003 \
go run main.go
```

```powershell
// powershell
$env:PORT = "8001"; `
$env:SELF_URL = "http://localhost:8001"; `
$env:PEERS = "http://localhost:8001,http://localhost:8002,http://localhost:8003"; `
go run main.go
```




hoặc tạo file `.env.1`


```bash
SELF_URL=http://localhost:8001
PORT=8001
PEERS=http://localhost:8001,http://localhost:8002,http://localhost:8003
```

rồi chạy 
```bash
ENV_FILE=.env.1 go run main.go
```

Test api

```curl
# PUT key - chưa implement
curl -X POST "http://localhost:8001/kv?key=hello&value=world"

# GET key - chưa implement
curl -X GET "http://localhost:8002/kv?key=hello"

# Health check
curl http://localhost:8003/health
```


## Các bước tiếp theo
 - Implement `forward.go` và `handler.go`
   - `forward.go`:
     - Implement `Forward()` để gửi HTTP request tới node khác (key-value put/get)
     - Implement `CopyResponse()` để copy response từ node được forward về client
   - `handler.go`:
     - Nhận HTTP request tại `/kv`
     - Sử dụng `ring.Ring.GetNodeForKey(key)` để xác định node cần xử lý
     - Nếu node là local → xử lý bằng `store`
     - Nếu node là node khác → gọi `forward.Forward()` để gửi đến node kia
 - Replication: 1 key lưu trên nhiều node
 - Peer discovery: dùng gossip để cập nhật danh sách node động
 - CLI tool: gõ lệnh put/get tiện hơn curl
 - DiskStore: lưu xuống file/disk
 - Quorum read/write: kiểm soát consistency

## Thống nhất format request response

```
API Endpoint: /kv

1. PUT (Ghi dữ liệu)
--------------------
Method: POST
URL: /kv?key=<key>&value=<value>

Ví dụ:
POST /kv?key=user123&value=Alice

Request body: (trống)

Response:
- 200 OK
{
  "status": "ok"
}

- 400 Bad Request (thiếu key hoặc value)
{
  "error": "missing key or value"
}


2. GET (Đọc dữ liệu)
--------------------
Method: GET
URL: /kv?key=<key>

Ví dụ:
GET /kv?key=user123

Response:
- 200 OK (nếu tồn tại)
{
  "key": "user123",
  "value": "Alice"
}

- 404 Not Found (nếu không có key)
{
  "error": "key not found"
}

- 400 Bad Request (thiếu key)
{
  "error": "missing key"
}


3. Dùng chung cho:
-------------------
- Client → Node
- Node → Node (forward)

Không có endpoint nội bộ riêng cho forward.
Giao tiếp giống y hệt: HTTP GET hoặc POST tới /kv.

```