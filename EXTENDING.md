# 🚀 Extending Pixerve: Adding Converter & Writer Backends

This guide explains how to extend Pixerve's functionality by adding new **converter backends** (image encoders) and **writer backends** (storage systems). Pixerve's modular architecture makes it easy to add support for new image formats and storage providers without modifying core code.

## 📋 Table of Contents

- [Architecture Overview](#architecture-overview)
- [Adding Converter Backends](#adding-converter-backends)
  - [Local Command Encoders](#local-command-encoders)
  - [Remote GRPC Encoders](#remote-grpc-encoders)
  - [Custom Logic Encoders](#custom-logic-encoders)
- [Adding Writer Backends](#adding-writer-backends)
- [JWT Specification](#jwt-specification)
- [Registration & Initialization](#registration--initialization)
- [Testing Your Extensions](#testing-your-extensions)
- [Best Practices](#best-practices)
- [Examples Repository](#examples-repository)

## 🏗️ Architecture Overview

Pixerve uses a **plugin-based architecture** with clean abstractions:

### Converter Backends (Encoders)

```go
// encoder/encoder.go
type EncodeFunc func(ctx context.Context, input, output string, opts EncodeOptions) error

type EncodeOptions struct {
    Width, Height int  // Target dimensions
    Quality       int  // 1-100 quality setting
    Speed         int  // Encoder speed/efficiency tradeoff
}
```

- **Registry**: `map[string]EncodeFunc` maps format names to encoder functions
- **Generic Interface**: Works for local commands, remote APIs, or custom logic
- **Context Support**: Built-in cancellation and timeout handling

### Writer Backends (Storage)

```go
// writerBackends/executor.go
func WriteImage(ctx context.Context, accessInfo map[string]string, reader io.Reader, backendType string) error
```

- **Switch-Based Dispatch**: Single function routes to specific backend implementations
- **Generic Interface**: `io.Reader` input, `map[string]string` credentials/config
- **Extensible**: Add new cases to the switch statement

## 🎨 Adding Converter Backends

### Local Command Encoders

Local encoders use command-line tools installed on the system.

#### Example: Adding HEIC Support

**1. Create encoder file: `encoder/heic.go`**

```go
package encoder

import (
    "context"
    "fmt"
    "os/exec"
    "strconv"
    "pixerve/logger"
)

func encodeHEIC(ctx context.Context, input, output string, opts EncodeOptions) error {
    // Check if ImageMagick is available
    if _, err := exec.LookPath("magick"); err != nil {
        return fmt.Errorf("ImageMagick not found: %w", err)
    }

    // Build command arguments
    args := []string{
        "magick",
        "convert",
        input,
        "-quality", strconv.Itoa(opts.Quality),
    }

    // Add resizing if specified
    if opts.Width > 0 && opts.Height > 0 {
        args = append(args, "-resize", fmt.Sprintf("%dx%d", opts.Width, opts.Height))
    }

    args = append(args, output)

    // Execute command
    cmd := exec.CommandContext(ctx, "magick", args...)
    output, err := cmd.CombinedOutput()
    if err != nil {
        logger.Errorf("HEIC encoding failed: %s", string(output))
        return fmt.Errorf("HEIC encoding failed: %w", err)
    }

    logger.Debugf("HEIC encoding completed: %s -> %s", input, output)
    return nil
}
```

**2. Register in init function: `encoder/heic.go`**

```go
func init() {
    Register("heic", "magick", encodeHEIC)
}
```

**3. Update imports in `encoder/encoder.go`**

```go
import (
    // ... existing imports
    _ "pixerve/encoder"  // registers all encoders
)
```

#### Example: Adding JPEG XL Support

**`encoder/jxl.go`**

```go
package encoder

import (
    "context"
    "fmt"
    "os/exec"
    "strconv"
    "pixerve/logger"
)

func encodeJXL(ctx context.Context, input, output string, opts EncodeOptions) error {
    // cjxl (Cloudinary's JPEG XL encoder)
    args := []string{
        "cjxl",
        input,
        output,
        "--quality", strconv.Itoa(opts.Quality),
    }

    // JPEG XL specific options
    if opts.Speed > 0 {
        args = append(args, "--speed", strconv.Itoa(opts.Speed))
    }

    cmd := exec.CommandContext(ctx, "cjxl", args...)
    if output, err := cmd.CombinedOutput(); err != nil {
        logger.Errorf("JXL encoding failed: %s", string(output))
        return fmt.Errorf("JXL encoding failed: %w", err)
    }

    return nil
}

func init() {
    Register("jxl", "cjxl", encodeJXL)
}
```

### Remote GRPC Encoders

Remote encoders offload processing to specialized services via GRPC.

#### Example: GPU-Accelerated WebP Encoder

##### 1. Define protobuf (optional but recommended)

```protobuf
// proto/gpu_webp.proto
service GPUWebPService {
    rpc EncodeWebP(EncodeRequest) returns (EncodeResponse);
}

message EncodeRequest {
    bytes image_data = 1;
    int32 quality = 2;
    int32 width = 3;
    int32 height = 4;
    int32 speed = 5;
}

message EncodeResponse {
    bytes encoded_data = 1;
    string error_message = 2;
}
```

##### 2. Create encoder: `encoder/gpu_webp.go`

```go
package encoder

import (
    "context"
    "fmt"
    "os"
    "io/ioutil"
    "google.golang.org/grpc"
    pb "pixerve/proto/gpu_webp"  // generated from protobuf
    "pixerve/logger"
)

var gpuWebpClient pb.GPUWebPServiceClient

func initGPUWebpClient() {
    // Initialize GRPC client (could read from env vars)
    conn, err := grpc.Dial("gpu-service:50051", grpc.WithInsecure())
    if err != nil {
        logger.Warnf("GPU WebP service unavailable: %v", err)
        return
    }
    gpuWebpClient = pb.NewGPUWebPServiceClient(conn)
    Register("gpu-webp", "grpc-client", encodeGPUWebP)
}

func encodeGPUWebP(ctx context.Context, input, output string, opts EncodeOptions) error {
    if gpuWebpClient == nil {
        return fmt.Errorf("GPU WebP service not available")
    }

    // Read input file
    inputData, err := ioutil.ReadFile(input)
    if err != nil {
        return fmt.Errorf("failed to read input: %w", err)
    }

    // Make GRPC call
    resp, err := gpuWebpClient.EncodeWebP(ctx, &pb.EncodeRequest{
        ImageData: inputData,
        Quality:   int32(opts.Quality),
        Width:     int32(opts.Width),
        Height:    int32(opts.Height),
        Speed:     int32(opts.Speed),
    })
    if err != nil {
        return fmt.Errorf("GPU WebP encoding failed: %w", err)
    }

    if resp.ErrorMessage != "" {
        return fmt.Errorf("GPU WebP service error: %s", resp.ErrorMessage)
    }

    // Write output
    if err := ioutil.WriteFile(output, resp.EncodedData, 0644); err != nil {
        return fmt.Errorf("failed to write output: %w", err)
    }

    logger.Debugf("GPU WebP encoding completed: %s -> %s", input, output)
    return nil
}

func init() {
    initGPUWebpClient()
}
```

#### Example: Cloud AI Image Enhancement

**`encoder/ai_enhance.go`**

```go
package encoder

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "pixerve/logger"
)

type AIEnhanceRequest struct {
    ImageData string `json:"image_data"`  // base64 encoded
    Model     string `json:"model"`       // "upscale", "denoise", "colorize"
    Strength  int    `json:"strength"`    // 1-100
}

type AIEnhanceResponse struct {
    EnhancedImage string `json:"enhanced_image"`
    Error         string `json:"error,omitempty"`
}

func encodeAIEnhance(ctx context.Context, input, output string, opts EncodeOptions) error {
    // Read and encode input
    inputData, err := ioutil.ReadFile(input)
    if err != nil {
        return err
    }

    // Prepare request
    req := AIEnhanceRequest{
        ImageData: base64.StdEncoding.EncodeToString(inputData),
        Model:     "upscale",  // could be configurable via opts
        Strength:  opts.Quality,
    }

    jsonData, _ := json.Marshal(req)

    // HTTP call to AI service
    httpReq, err := http.NewRequestWithContext(ctx, "POST",
        "https://ai-enhance-service.com/api/enhance", bytes.NewBuffer(jsonData))
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+os.Getenv("AI_SERVICE_TOKEN"))

    resp, err := http.DefaultClient.Do(httpReq)
    if err != nil {
        return fmt.Errorf("AI enhancement failed: %w", err)
    }
    defer resp.Body.Close()

    var aiResp AIEnhanceResponse
    if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
        return fmt.Errorf("failed to decode AI response: %w", err)
    }

    if aiResp.Error != "" {
        return fmt.Errorf("AI service error: %s", aiResp.Error)
    }

    // Decode and save result
    enhancedData, err := base64.StdEncoding.DecodeString(aiResp.EnhancedImage)
    if err != nil {
        return fmt.Errorf("failed to decode enhanced image: %w", err)
    }

    return ioutil.WriteFile(output, enhancedData, 0644)
}

func init() {
    Register("ai-enhance", "http-client", encodeAIEnhance)
}
```

### Custom Logic Encoders

For encoders that need custom processing logic.

#### Example: Progressive JPEG with Custom Compression

**`encoder/progressive_jpeg.go`**

```go
package encoder

import (
    "context"
    "fmt"
    "image"
    "image/jpeg"
    "os"
    "pixerve/logger"

    "github.com/nfnt/resize"
)

func encodeProgressiveJPEG(ctx context.Context, input, output string, opts EncodeOptions) error {
    // Open input file
    file, err := os.Open(input)
    if err != nil {
        return fmt.Errorf("failed to open input: %w", err)
    }
    defer file.Close()

    // Decode image
    img, _, err := image.Decode(file)
    if err != nil {
        return fmt.Errorf("failed to decode image: %w", err)
    }

    // Resize if needed
    if opts.Width > 0 && opts.Height > 0 {
        img = resize.Resize(uint(opts.Width), uint(opts.Height), img, resize.Lanczos3)
    }

    // Create output file
    outFile, err := os.Create(output)
    if err != nil {
        return fmt.Errorf("failed to create output: %w", err)
    }
    defer outFile.Close()

    // Custom JPEG options
    encoder := &jpeg.Encoder{
        Quality: opts.Quality,
        Options: &jpeg.Options{
            Quality: opts.Quality,
        },
    }

    // Encode with progressive flag (Go 1.19+)
    if err := encoder.Encode(outFile, img); err != nil {
        return fmt.Errorf("JPEG encoding failed: %w", err)
    }

    logger.Debugf("Progressive JPEG encoding completed: %s -> %s", input, output)
    return nil
}

func init() {
    Register("progressive-jpeg", "go-native", encodeProgressiveJPEG)
}
```

## 💾 Adding Writer Backends

Writer backends handle storing processed images to different storage systems.

### Example: Adding Cloudflare R2 Support

**1. Add case to executor: `writerBackends/executor.go`**

```go
case "cloudflare-r2":
    err := UploadToCloudflareR2(ctx, accessInfo, reader)
    if err != nil {
        return fmt.Errorf("failed to upload to Cloudflare R2: %w", err)
    }
```

**2. Implement uploader: `writerBackends/cloudflare_r2.go`**

```go
package writerbackends

import (
    "context"
    "fmt"
    "io"
    "mime"
    "path/filepath"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/credentials"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
    "github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func UploadToCloudflareR2(ctx context.Context, accessInfo map[string]string, reader io.Reader) error {
    // Cloudflare R2 uses S3-compatible API
    accountId := accessInfo["account_id"]
    accessKey := accessInfo["access_key_id"]
    secretKey := accessInfo["secret_access_key"]
    bucket := accessInfo["bucket"]
    region := accessInfo["region"]  // usually "auto" for R2
    key := accessInfo["key"]        // object key/path

    if accountId == "" || accessKey == "" || secretKey == "" || bucket == "" || key == "" {
        return fmt.Errorf("missing required Cloudflare R2 credentials")
    }

    // Create S3-compatible session
    sess, err := session.NewSession(&aws.Config{
        Region:      aws.String(region),
        Endpoint:    aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountId)),
        Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
        S3ForcePathStyle: aws.Bool(true),
    })
    if err != nil {
        return fmt.Errorf("failed to create R2 session: %w", err)
    }

    // Determine content type
    contentType := mime.TypeByExtension(filepath.Ext(key))
    if contentType == "" {
        contentType = "application/octet-stream"
    }

    // Upload
    uploader := s3manager.NewUploader(sess)
    _, err = uploader.UploadWithContext(ctx, &s3manager.UploadInput{
        Bucket:      aws.String(bucket),
        Key:         aws.String(key),
        Body:        reader,
        ContentType: aws.String(contentType),
        ACL:         aws.String("private"), // or "public-read" based on needs
    })

    if err != nil {
        return fmt.Errorf("R2 upload failed: %w", err)
    }

    return nil
}
```

### Example: Adding Backblaze B2 Support

**`writerBackends/backblaze_b2.go`**

```go
package writerbackends

import (
    "context"
    "crypto/sha1"
    "fmt"
    "io"
    "net/http"
    "encoding/json"
    "bytes"
    "pixerve/logger"
)

type B2AuthResponse struct {
    AccountID      string `json:"accountId"`
    AuthorizationToken string `json:"authorizationToken"`
    APIURL         string `json:"apiUrl"`
    DownloadURL    string `json:"downloadUrl"`
}

type B2UploadResponse struct {
    FileID   string `json:"fileId"`
    FileName string `json:"fileName"`
    UploadTimestamp int64 `json:"uploadTimestamp"`
}

func UploadToBackblazeB2(ctx context.Context, accessInfo map[string]string, reader io.Reader) error {
    appKeyId := accessInfo["application_key_id"]
    appKey := accessInfo["application_key"]
    bucketId := accessInfo["bucket_id"]
    fileName := accessInfo["file_name"]

    if appKeyId == "" || appKey == "" || bucketId == "" || fileName == "" {
        return fmt.Errorf("missing required Backblaze B2 credentials")
    }

    // Authenticate
    authResp, err := authenticateB2(appKeyId, appKey)
    if err != nil {
        return fmt.Errorf("B2 authentication failed: %w", err)
    }

    // Get upload URL
    uploadURL, uploadAuth, err := getB2UploadURL(authResp, bucketId)
    if err != nil {
        return fmt.Errorf("failed to get B2 upload URL: %w", err)
    }

    // Read all data for SHA1 calculation
    data, err := io.ReadAll(reader)
    if err != nil {
        return fmt.Errorf("failed to read data: %w", err)
    }

    // Calculate SHA1
    sha1Hash := fmt.Sprintf("%x", sha1.Sum(data))

    // Upload
    req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, bytes.NewReader(data))
    if err != nil {
        return err
    }

    req.Header.Set("Authorization", uploadAuth)
    req.Header.Set("X-Bz-File-Name", fileName)
    req.Header.Set("Content-Type", "b2/x-auto")  // auto-detect
    req.Header.Set("X-Bz-Content-Sha1", sha1Hash)
    req.Header.Set("X-Bz-Info-src_last_modified_millis", "0")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return fmt.Errorf("B2 upload request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return fmt.Errorf("B2 upload failed with status: %d", resp.StatusCode)
    }

    var uploadResp B2UploadResponse
    if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
        return fmt.Errorf("failed to decode B2 response: %w", err)
    }

    logger.Debugf("B2 upload successful: %s (ID: %s)", fileName, uploadResp.FileID)
    return nil
}

func authenticateB2(keyID, key string) (*B2AuthResponse, error) {
    auth := keyID + ":" + key
    encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))

    req, err := http.NewRequest("GET", "https://api.backblazeb2.com/b2api/v2/b2_authorize_account", nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Authorization", "Basic "+encodedAuth)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var authResp B2AuthResponse
    if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
        return nil, err
    }

    return &authResp, nil
}

func getB2UploadURL(authResp *B2AuthResponse, bucketID string) (string, string, error) {
    // Implementation for getting upload URL...
    // (Would make another API call to get_upload_url)
    return "", "", nil
}
```

## 📝 JWT Specification

### Basic JWT Structure

```json
{
  "iss": "pixerve-client",
  "sub": "user_access_key_12345",
  "iat": 1640995200,
  "exp": 1640998800,
  "job": {
    "completionCallback": "https://api.example.com/webhook",
    "priority": 1,
    "keepOriginal": false,
    "formats": {
      "local-jpeg": {
        "settings": {"quality": 80, "speed": 3},
        "sizes": [[800, 600], [400, 300]]
      },
      "remote-avif": {
        "settings": {"quality": 90, "speed": 2},
        "sizes": [[1920, 1080]]
      },
      "gpu-webp": {
        "settings": {"quality": 85, "speed": 1},
        "sizes": [[1200, 800]]
      }
    },
    "storageKeys": {
      "s3": "storage_key_abc123",
      "cloudflare-r2": "storage_key_def456"
    }
  }
}
```

### Advanced JWT with Multiple Backends

```json
{
  "job": {
    "formats": {
      "heic": {
        "settings": {"quality": 85},
        "sizes": [[2000, 1500]]
      },
      "jxl": {
        "settings": {"quality": 90, "speed": 2},
        "sizes": [[1500, 1125]]
      },
      "ai-enhance": {
        "settings": {"quality": 95},
        "sizes": [[1000, 750]]
      }
    },
    "storageKeys": {
      "backblaze-b2": "b2_key_xyz789",
      "gcp": "gcp_key_abc123"
    },
    "directHost": true,
    "subDir": "user_uploads/2024"
  }
}
```

### Storage Key Configuration

Storage keys are created via `/register` endpoint and referenced in JWT:

```bash
# Register S3 credentials
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{
    "access_key_id": "AKIA...",
    "secret_access_key": "...",
    "region": "us-east-1",
    "bucket": "my-images"
  }'
# Returns: {"access_key": "storage_key_abc123"}

# Register Cloudflare R2
curl -X POST http://localhost:8080/register \
  -d '{
    "account_id": "1234567890",
    "access_key_id": "abc123...",
    "secret_access_key": "...",
    "bucket": "images"
  }'
# Returns: {"access_key": "storage_key_def456"}
```

## 🔧 Registration & Initialization

### Encoder Registration

Encoders are automatically registered via `init()` functions:

```go
// encoder/custom.go
package encoder

func init() {
    Register("custom-format", "required-command", encodeCustom)
    // Registers only if "required-command" is available in PATH
}
```

### Backend Registration

Writer backends are registered by adding cases to the switch:

```go
// writerBackends/executor.go
func WriteImage(ctx context.Context, accessInfo map[string]string, reader io.Reader, backendType string) error {
    switch backendType {
    case "s3":
        // ...
    case "gcs":
        // ...
    case "your-new-backend":
        return UploadToYourBackend(ctx, accessInfo, reader)
    default:
        return fmt.Errorf("unknown backend: %s", backendType)
    }
}
```

### Environment Variables

```bash
# For remote services
export GPU_WEBP_SERVICE_URL="grpc://gpu-service:50051"
export AI_ENHANCE_API_KEY="sk-..."

# For storage backends
export CLOUDFLARE_ACCOUNT_ID="123456"
export BACKBLAZE_KEY_ID="abc123"
```

## 🧪 Testing Your Extensions

### Unit Testing Encoders

```go
// encoder/heic_test.go
package encoder

import (
    "context"
    "os"
    "testing"
)

func TestEncodeHEIC(t *testing.T) {
    // Create test input
    input := createTestJPEG(t, 100, 100)
    defer os.Remove(input)

    output := "/tmp/test.heic"
    defer os.Remove(output)

    opts := EncodeOptions{
        Width: 50, Height: 50,
        Quality: 80,
    }

    err := encodeHEIC(context.Background(), input, output, opts)
    if err != nil {
        t.Fatalf("HEIC encoding failed: %v", err)
    }

    // Verify output exists and has reasonable size
    stat, err := os.Stat(output)
    if err != nil {
        t.Fatal("Output file not created")
    }
    if stat.Size() < 100 {  // Very basic check
        t.Error("Output file suspiciously small")
    }
}
```

### Integration Testing Backends

```go
// writerBackends/cloudflare_r2_test.go
package writerbackends

import (
    "bytes"
    "context"
    "os"
    "testing"
)

func TestUploadToCloudflareR2(t *testing.T) {
    if os.Getenv("CLOUDFLARE_R2_TEST") != "true" {
        t.Skip("Skipping Cloudflare R2 integration test")
    }

    accessInfo := map[string]string{
        "account_id": os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
        "access_key_id": os.Getenv("CLOUDFLARE_ACCESS_KEY"),
        "secret_access_key": os.Getenv("CLOUDFLARE_SECRET_KEY"),
        "bucket": os.Getenv("CLOUDFLARE_BUCKET"),
        "region": "auto",
        "key": "test-image.jpg",
    }

    testData := []byte("fake image data")
    reader := bytes.NewReader(testData)

    err := UploadToCloudflareR2(context.Background(), accessInfo, reader)
    if err != nil {
        t.Fatalf("Cloudflare R2 upload failed: %v", err)
    }
}
```

### End-to-End Testing

```bash
# 1. Start Pixerve
go run main.go

# 2. Register credentials
curl -X POST http://localhost:8080/register \
  -d '{"access_key_id": "...", "secret_access_key": "...", "bucket": "test"}'

# 3. Create test JWT
# (Use your JWT creation utility)

# 4. Upload test image
curl -X POST http://localhost:8080/upload \
  -H "Authorization: Bearer <jwt>" \
  -F "file=@test.jpg"

# 5. Check results
curl http://localhost:8080/status?hash=<returned-hash>
```

## 📋 Best Practices

### Encoder Development

1. **Error Handling**: Always check for command availability in `init()`
2. **Context Support**: Use `exec.CommandContext()` for cancellation
3. **Logging**: Log at appropriate levels (Debug for success, Error for failures)
4. **Resource Cleanup**: Close files and clean up temporary resources
5. **Validation**: Validate input parameters and file formats

### Backend Development

1. **Credential Security**: Never log credentials or sensitive data
2. **Timeout Handling**: Respect context deadlines for network operations
3. **Retry Logic**: Implement exponential backoff for transient failures
4. **Content Types**: Set appropriate MIME types for uploaded files
5. **Error Messages**: Provide clear, actionable error messages

### General Guidelines

1. **Naming Conventions**: Use descriptive names like `gpu-webp`, `cloudflare-r2`
2. **Documentation**: Document required environment variables and setup steps
3. **Dependencies**: Keep external dependencies minimal and well-maintained
4. **Testing**: Include both unit tests and integration tests
5. **Monitoring**: Add metrics and logging for performance monitoring

## 📚 Examples Repository

For complete working examples, see the [Pixerve Extensions Repository](https://github.com/example/pixerve-extensions):

- `encoders/heic/` - HEIC encoding with ImageMagick
- `encoders/jxl/` - JPEG XL with cjxl
- `encoders/gpu_webp/` - GRPC-based GPU WebP encoding
- `backends/cloudflare_r2/` - Cloudflare R2 storage
- `backends/backblaze_b2/` - Backblaze B2 storage
- `backends/akamai/` - Akamai CDN integration

## 🎯 Next Steps

1. **Choose Your Extension**: Decide whether to add a converter or storage backend
2. **Study Existing Code**: Look at similar implementations for patterns
3. **Implement Incrementally**: Start with basic functionality, add features iteratively
4. **Test Thoroughly**: Include unit tests, integration tests, and manual testing
5. **Document**: Update this guide with your new extension
6. **Contribute**: Consider upstreaming valuable extensions

---

**Need Help?** Check the [Pixerve GitHub Issues](https://github.com/example/pixerve/issues) or join our community Discord for extension development discussions.
