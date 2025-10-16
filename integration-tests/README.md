# Pixerve Integration Tests

Comprehensive integration test suite for the Pixerve image processing server, featuring Lorem Picsum image downloads, multiple codec testing, writer backend validation, and edge case handling.

## Features

- **Basic Functionality Tests**: Health endpoints, JWT validation, image uploads, status checking, callbacks, and job cancellation
- **Codec & Format Tests**: JPG, PNG, WebP, AVIF formats with various quality/speed settings and multi-format jobs
- **Writer Backend Tests**: Direct HTTP hosting, S3-compatible storage, Google Cloud Storage, SFTP/FTP storage
- **Edge Case Tests**: Large files, invalid formats, network errors, concurrent requests, malformed requests, corrupted files
- **Verbose Logging**: Comprehensive logging throughout all test suites with configurable log levels
- **Test Result Tracking**: Detailed pass/fail reporting with timing and error information

## Prerequisites

- Node.js 18+
- TypeScript 5.0+
- Go 1.19+ (for building Pixerve server)
- Access to Lorem Picsum API (for test images)

## Installation

```bash
cd integration-tests
npm install
npm run build
```

## Usage

### Run All Tests

```bash
npm test
```

### Run Specific Test Suites

```bash
# Basic functionality tests
npm run test:basic

# Codec and format tests
npm run test:codec

# Writer backend tests
npm run test:backend

# Edge case tests
npm run test:edge
```

### Advanced Usage

```bash
# Run with custom server URL
node dist/test-runner.js --url http://localhost:9090 all

# Enable verbose logging
node dist/test-runner.js --verbose all

# Output results as JSON
node dist/test-runner.js --json all

# Generate JUnit XML report
node dist/test-runner.js --junit results.xml all
```

## Test Configuration

### Environment Variables

- `PIXERVE_URL`: Pixerve server URL (default: `http://localhost:8080`)

### Test Images

Tests automatically download images from Lorem Picsum for realistic testing scenarios. Images are cached in the `test-images/` directory and cleaned up after each test run.

## Test Suites

### Basic Tests (`basic-tests.ts`)

Tests core Pixerve functionality:

- Health and version endpoints
- JWT token validation
- Image upload processing
- Job status monitoring
- Callback mechanisms
- Job cancellation
- Invalid request handling

### Codec Tests (`codec-tests.ts`)

Tests image processing capabilities:

- JPG format with quality settings (10-100)
- PNG format with speed settings (1-6)
- WebP format with quality/speed combinations
- AVIF format with various configurations
- Multi-format jobs
- Size variations and edge cases

### Backend Tests (`backend-tests.ts`)

Tests storage backend integrations:

- Direct HTTP hosting
- S3-compatible storage (AWS S3, MinIO, etc.)
- Google Cloud Storage
- SFTP/FTP storage
- Multiple backends simultaneously
- Backend error handling
- Backend priority settings

### Edge Case Tests (`edge-case-tests.ts`)

Tests error conditions and limits:

- Large file processing (5MB, 20MB, 50MB)
- Invalid file formats (text, corrupted JPEG, HTML, PDF)
- Network error handling
- Concurrent request processing (5, 10, 20 simultaneous uploads)
- Invalid JWT tokens (malformed, expired)
- Malformed HTTP requests
- Rate limiting scenarios
- Corrupted file handling
- Memory limit testing

## Test Results

### Console Output

Tests provide real-time feedback with:

- ✅ Pass indicators with test details
- ❌ Fail indicators with error messages
- ⏱️ Timing information
- 📊 Summary statistics

### JSON Output

Use `--json` flag for machine-readable results:

```json
[
  {
    "suite": "Basic Tests",
    "description": "Basic API functionality...",
    "duration": 1250,
    "results": {
      "results": [
        {
          "name": "Health Check",
          "passed": true,
          "duration": 45,
          "data": { "status": "ok" }
        }
      ]
    }
  }
]
```

### JUnit XML

Use `--junit results.xml` for CI/CD integration:

```xml
<testsuites>
  <testsuite name="Basic Tests" tests="10" failures="0" time="1.25">
    <testcase name="Health Check" classname="Basic Tests" time="0.045"/>
  </testsuite>
</testsuites>
```

## Development

### Project Structure

``` text
integration-tests/
├── src/
│   ├── common.ts          # Shared utilities and types
│   ├── basic-tests.ts     # Basic functionality tests
│   ├── codec-tests.ts     # Codec and format tests
│   ├── backend-tests.ts   # Writer backend tests
│   ├── edge-case-tests.ts # Edge case and error tests
│   └── test-runner.ts     # Main test runner CLI
├── test-images/           # Downloaded test images (auto-created)
├── dist/                  # Compiled JavaScript
├── package.json
├── tsconfig.json
└── README.md
```

### Adding New Tests

1. Create a new test file in `src/`
2. Implement a class extending the test pattern
3. Add the test suite to `test-runner.ts`
4. Update package.json scripts if needed

### Logging

Tests use a configurable logging system with levels:

- `ERROR`: Critical errors only
- `WARN`: Warnings and errors
- `INFO`: General information (default)
- `DEBUG`: Detailed debugging info
- `TRACE`: Very verbose tracing

Set log level in individual test classes or use `--verbose` flag.

## Troubleshooting

### Common Issues

1. **Server Connection Failed**
   - Ensure Pixerve server is running on the specified URL
   - Check firewall and network settings

2. **Image Download Failed**
   - Verify internet connection
   - Lorem Picsum API may be temporarily unavailable

3. **Backend Tests Skipped**
   - Storage backends (S3, GCS, SFTP) may not be configured
   - This is expected behavior - tests mark as "skipped"

4. **Memory/Performance Issues**
   - Large file tests may fail on systems with limited RAM
   - Reduce concurrent test limits if needed

### Debug Mode

Enable verbose logging for detailed troubleshooting:

```bash
node dist/test-runner.js --verbose all
```

## Code Examples

### Basic Image Upload

```typescript
import axios from 'axios';
import FormData from 'form-data';
import * as jose from 'jose';
import * as fs from 'fs';

// Create a JWT for image processing
const jobSpec = {
  priority: 0,
  keepOriginal: false,
  formats: {
    jpg: {
      settings: { quality: 80, speed: 1 },
      sizes: [[800, 600], [400, 300]]
    }
  },
  directHost: true
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

## Contributing

1. Follow the existing code patterns and TypeScript types
2. Add comprehensive error handling
3. Include both positive and negative test cases
4. Update documentation for new features
5. Ensure tests clean up after themselves

## License

MIT License - see LICENSE file for details.
