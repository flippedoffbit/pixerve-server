# Pixerve Integration Tests & Usage Examples

This TypeScript package provides comprehensive integration tests for the Pixerve image processing server and serves as a practical guide for using Pixerve from TypeScript/JavaScript applications.

## Features

- 🧪 **Full Integration Testing**: Tests the complete Pixerve API workflow
- 📚 **Usage Examples**: Demonstrates how to integrate Pixerve into TS/JS backends
- 🔄 **Real HTTP Calls**: Tests actual API endpoints, not just internal functions
- 🚀 **Auto Server Management**: Automatically builds and starts Pixerve for testing

## Prerequisites

- Node.js 16+
- Go 1.19+
- ImageMagick (for image processing)

## Installation

```bash
# From the integration-tests directory
npm install
```

## Usage

### Running Integration Tests

```bash
# Run the full test suite
npm test

# Build only
npm run build

# Run in watch mode (for development)
npm run test:watch
```

### Using as a Code Example

The test file (`src/test.ts`) demonstrates:

1. **JWT Creation**: How to create properly signed JWTs for Pixerve
2. **File Upload**: How to upload images with job specifications
3. **API Queries**: How to check job status via success/failure endpoints
4. **Callback Handling**: How to receive processing completion callbacks

### Code Example

```typescript
import axios from 'axios';
import FormData from 'form-data';
import * as jose from 'jose';
import * as fs from 'fs';

// Create a JWT for image processing
const jobSpec = {
  completionCallback: 'https://your-app.com/webhook',
  priority: 0,
  keepOriginal: false,
  formats: {
    jpg: {
      settings: { quality: 80, speed: 1 },
      sizes: [[800, 600], [400, 300]]
    },
    webp: {
      settings: { quality: 85, speed: 2 },
      sizes: [[800], [400]]
    }
  },
  storageKeys: {
    s3: 'your-s3-key'
  },
  directHost: true,
  subDir: 'user-uploads'
};

const secret = new TextEncoder().encode('your-jwt-secret');
const jwt = await new jose.SignJWT({
  sub: 'job-id-123',
  job: jobSpec,
  iat: Math.floor(Date.now() / 1000),
  exp: Math.floor(Date.now() / 1000) + 3600
})
.setProtectedHeader({ alg: 'HS256' })
.sign(secret);

// Upload the image
const form = new FormData();
form.append('token', jwt);
form.append('file', fs.createReadStream('image.jpg'));

const response = await axios.post('http://localhost:8080/upload', form, {
  headers: form.getHeaders()
});

console.log('Upload successful:', response.data);
```

## API Endpoints Tested

- `POST /upload` - Image upload with JWT
- `GET /success?hash=...` - Check successful processing
- `GET /failures?hash=...` - Check failed processing
- `POST /callback` - Receive processing completion callbacks

## Configuration

The tests use these default settings:

- **Server URL**: `http://localhost:8080`
- **JWT Secret**: `test-secret-key-for-jwt-signing-at-least-32-bytes-long`
- **Test Image**: Auto-generated 1x1 PNG

## Test Structure

``` text
src/
├── test.ts          # Main integration test file
└── types.ts         # TypeScript type definitions (future)
```

## Development

```bash
# Install dependencies
npm install

# Build TypeScript
npm run build

# Run tests
npm test

# Run usage example
npm run example

# Clean build artifacts
npm run clean
```

## Code Examples

The tests include comprehensive error handling for:

- Server startup failures
- Network timeouts
- Invalid JWT tokens
- File upload errors
- API response validation

## Contributing

When adding new tests:

1. Follow the existing async/await patterns
2. Include proper error handling
3. Add TypeScript types for new API responses
4. Update this README with new examples

## License

MIT
