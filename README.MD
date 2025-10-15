# 🌌 Pixerve

**Pixerve** is a work-in-progress, open-source image ingestion and processing server built for modern web services.  
It aims to make image handling **predictable, transparent, and fast** — without locking anyone into closed APIs or external services.

---

## 🧭 Motivation

Every growing product ends up building the same thing over and over again:  
a simple, reliable place to upload images, convert them, resize them, and serve them.  

Pixerve is our attempt to stop rewriting that wheel — and do it properly this time:  
a small, efficient image worker you can host, shape, and extend however you like.

---

## 🌟 Current Features

### ✅ Core Functionality
- **JWT-Based Authentication**: Secure upload authorization with structured job specifications
- **Multi-Format Conversion**: Support for JPG, PNG, WebP, AVIF, and lossless copying
- **Multiple Storage Backends**: S3, Google Cloud Storage, SFTP, and direct HTTP serving
- **File-Based Queuing**: Crash-resistant job processing with in-memory pending management
- **Failure Tracking**: Persistent failure storage with query API for status checking
- **Structured Logging**: Custom logger with console and file output support

### ✅ API Endpoints
- `POST /upload` - Upload images with JWT authentication
- `GET /failures?hash=<sha256>` - Check processing status for uploaded files
- `GET /failures/list` - Admin endpoint for listing all failures

### ✅ Job Specification (JWT Payload)
```json
{
  "iss": "your-app",
  "sub": "user-123",
  "iat": 1640995200,
  "exp": 1641081600,
  "job": {
    "completionCallback": "https://your-app.com/webhook",
    "callbackHeaders": {"Authorization": "Bearer token"},
    "priority": 0,
    "keepOriginal": false,
    "formats": {
      "jpg": {
        "settings": {"quality": 80, "speed": 1},
        "sizes": [[800, 600], [400, 300]]
      },
      "webp": {
        "settings": {"quality": 85, "speed": 2},
        "sizes": [[800], [400]]
      }
    },
    "storageKeys": {
      "s3": "s3-credential-key",
      "gcs": "gcs-credential-key"
    },
    "directHost": true,
    "subDir": "tenant-123"
  }
}
```

---

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- ImageMagick (`magick` command) for JPG/PNG conversion
- `cwebp` for WebP conversion  
- `avifenc` for AVIF conversion

### Installation
```bash
git clone https://github.com/yourusername/pixerve.git
cd pixerve
go mod download
```

### Configuration
Set the JWT secret in `config/jwt_secret.go`:
```go
package config

const SHARED_JWT_SECRET = "your-256-bit-secret"
```

### Running
```bash
go run main.go
```

Server starts on `http://localhost:8080`

---

## 📖 Usage Guide

### 1. Upload Images

**Endpoint**: `POST /upload`

**Headers**:
```
Authorization: Bearer <jwt-token>
Content-Type: multipart/form-data
```

**Body**:
```
file: <image-file>
```

**Response** (immediate):
```json
{
  "hash": "sha256_of_uploaded_file",
  "expected_files": [
    "hash_original_600_800_.jpg",
    "hash_original_300_400_.jpg", 
    "hash_original_600_800_.webp",
    "hash_original.jpg"
  ]
}
```

### 2. Check Processing Status

**Endpoint**: `GET /failures?hash=<sha256>`

**Response** (success):
```json
{
  "hash": "sha256...",
  "status": "success",
  "message": "File processed successfully"
}
```

**Response** (failure):
```json
{
  "hash": "sha256...",
  "status": "failed",
  "timestamp": "2025-10-15T18:20:56Z",
  "error": "processing failed: invalid image format",
  "job_data": "{...}"
}
```

### 3. JWT Token Creation

Create JWT tokens with job specifications using your preferred JWT library:

```javascript
const jwt = require('jsonwebtoken');

const payload = {
  iss: 'my-app',
  sub: 'user-123',
  iat: Math.floor(Date.now() / 1000),
  exp: Math.floor(Date.now() / 1000) + 3600, // 1 hour
  job: {
    completionCallback: 'https://my-app.com/webhook',
    formats: {
      jpg: {
        settings: { quality: 80 },
        sizes: [[800, 600], [400, 300]]
      }
    },
    storageKeys: { s3: 'my-s3-key' },
    directHost: true
  }
};

const token = jwt.sign(payload, 'your-secret-key');
```

### 4. Storage Backend Setup

#### S3 Configuration
Store credentials in Pebble DB:
```bash
# Add S3 credentials
curl -X POST http://localhost:8080/admin/credentials \
  -d '{"key": "my-s3-key", "type": "s3", "data": {"access_key": "...", "secret_key": "...", "bucket": "..."}}'
```

#### Direct Serving
Files served at: `http://your-server/files/tenant-123/filename.jpg`

---

## 🧪 Testing

Run the test suite:
```bash
go test ./tests/...
```

Test coverage includes:
- JWT token parsing and validation
- Job processing logic
- Failure store operations
- Encoder registration
- Pending job management

---

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   HTTP Server   │────│  Job Processing  │────│  Storage        │
│                 │    │  - JWT Parsing   │    │  Backends       │
│ • /upload       │    │  - Conversions   │    │  - S3           │
│ • /failures     │    │  - File Naming   │    │  - GCS          │
└─────────────────┘    └──────────────────┘    │  - SFTP         │
                                               │  - Direct       │
┌─────────────────┐    ┌──────────────────┐    └─────────────────┘
│  Queue System   │────│  Failure Store   │
│                 │    │  - Pebble DB     │
│ • File-based    │    │  - Query API     │
│ • In-memory     │    │  - Crash recovery│
└─────────────────┘    └──────────────────┘
```

---

## 📋 What Remains To Do

### 🔄 High Priority
- **Callback Implementation**: HTTP callbacks for job completion
- **Credential Management API**: REST endpoints for managing storage credentials
- **Rate Limiting**: Prevent abuse and manage resource usage
- **Metrics & Monitoring**: Processing stats, queue depth, failure rates

### 🔄 Medium Priority  
- **Batch Uploads**: Support multiple files in single request
- **Progress Tracking**: Real-time processing status updates
- **Image Optimization**: Auto-orientation, EXIF stripping, format selection
- **CDN Integration**: CloudFront, Cloudflare integration

### 🔄 Low Priority
- **Video Support**: Extend to video transcoding
- **Webhook Retries**: Failed callback retry logic
- **Admin Dashboard**: Web UI for monitoring and management
- **Plugin System**: Custom encoders and storage backends

---

## 🚀 Enhancement Suggestions

### Performance
- **Worker Pool**: Multiple concurrent workers for high throughput
- **GPU Acceleration**: Hardware-accelerated encoding where available
- **Caching Layer**: Redis for frequently accessed images
- **CDN Integration**: Automatic CDN invalidation and distribution

### Reliability
- **Dead Letter Queue**: Handle persistently failing jobs
- **Circuit Breakers**: Fail fast for unhealthy storage backends
- **Health Checks**: Storage backend availability monitoring
- **Graceful Shutdown**: Complete in-flight jobs before shutdown

### Security
- **Request Signing**: Additional request authentication beyond JWT
- **Rate Limiting**: Per-user and per-IP limits
- **Content Validation**: Image malware scanning and type validation
- **Audit Logging**: Complete request/response logging for compliance

### Developer Experience
- **OpenAPI Spec**: Complete API documentation
- **SDKs**: Client libraries for popular languages
- **Docker Images**: Pre-built containers with all dependencies
- **Configuration UI**: Web interface for server configuration

### Enterprise Features
- **Multi-Tenancy**: Complete tenant isolation
- **SLA Management**: Guaranteed processing times
- **Custom Workflows**: Pluggable processing pipelines
- **Analytics**: Usage patterns and performance insights

---

## 🧭 Philosophy

Pixerve doesn't chase feature lists or buzzwords.  
Its purpose is **clarity and control** — the kind of tool you can actually understand, trust, and build around.  
If it ends up helping others the same way it helps us, then it's already a success.

---

## 🪶 License

TBD — open source, permissive, and community-friendly once the first release lands.

---

## 🗣️ Follow Along

Pixerve's journey is open.  
Discussions, design notes, and ideas will all live here — transparently, as they evolve.  

Stay tuned, and if you like where this is going, star the repo or drop by with ideas.  

> _"Managed Image Processing — one pixel at a time."_
