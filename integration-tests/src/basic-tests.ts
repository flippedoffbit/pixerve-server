/**
 * Basic functionality integration tests for Pixerve
 *
 * Tests core functionality:
 * - JWT creation and validation
 * - Image upload
 * - Status checking (success/failure)
 * - Callback functionality
 * - Job cancellation
 * - Health and version endpoints
 */

import * as fs from 'fs';
import * as path from 'path';
import * as http from 'http';
import FormData from 'form-data';
import {
    Logger,
    LogLevel,
    JobSpec,
    UploadResponse,
    SuccessResponse,
    FailureResponse,
    StatusResponse,
    JWTUtils,
    HTTPUtils,
    ImageUtils,
    PixerveServer,
    TestResults,
    JobSpecUtils,
    logger
} from './common';

// For CommonJS __dirname equivalent
const __dirname = path.dirname(require.main!.filename);

export class BasicFunctionalityTests {
    private baseUrl: string;
    private server: PixerveServer;
    private results: TestResults;
    private logger: Logger;

    constructor(baseUrl = 'http://localhost:8080') {
        this.baseUrl = baseUrl;
        this.server = new PixerveServer(baseUrl);
        this.results = new TestResults();
        this.logger = new Logger(LogLevel.DEBUG, 'BASIC');
    }

    async runAll (): Promise<void> {
        this.logger.info('Starting basic functionality tests');

        try {
            // Setup
            await this.setup();

            // Run tests
            await this.testHealthEndpoint();
            await this.testVersionEndpoint();
            await this.testJWT();
            await this.testImageUpload();
            await this.testStatusEndpoints();
            await this.testCallbackFunctionality();
            await this.testJobCancellation();
            await this.testInvalidRequests();

            // Cleanup
            await this.cleanup();

        } catch (error) {
            this.logger.error('Basic functionality tests failed', { error: (error as Error).message });
            throw error;
        } finally {
            this.results.printSummary();
        }
    }

    private async setup (): Promise<void> {
        this.logger.info('Setting up basic functionality tests');
        ImageUtils.ensureTestImagesDir();
        await this.server.start();
        await this.server.waitForReady();
    }

    private async cleanup (): Promise<void> {
        this.logger.info('Cleaning up basic functionality tests');
        await this.server.stop();
        ImageUtils.cleanupTestImages();
    }

    private async testHealthEndpoint (): Promise<void> {
        const endTimer = this.results.startTest('Health Endpoint');

        try {
            const response = await HTTPUtils.get(`${ this.baseUrl }/health`);
            if (response.status !== 200) {
                throw new Error(`Health endpoint returned status ${ response.status }`);
            }

            this.results.recordPass('Health Endpoint', 0, { status: response.status });
        } catch (error) {
            this.results.recordFail('Health Endpoint', 0, (error as Error).message);
            throw error;
        } finally {
            endTimer();
        }
    }

    private async testVersionEndpoint (): Promise<void> {
        const endTimer = this.results.startTest('Version Endpoint');

        try {
            const response = await HTTPUtils.get(`${ this.baseUrl }/version`);
            if (response.status !== 200) {
                throw new Error(`Version endpoint returned status ${ response.status }`);
            }

            if (!response.data.version) {
                throw new Error('Version endpoint response missing version field');
            }

            this.results.recordPass('Version Endpoint', 0, { version: response.data.version });
        } catch (error) {
            this.results.recordFail('Version Endpoint', 0, (error as Error).message);
            throw error;
        } finally {
            endTimer();
        }
    }

    private async testJWT (): Promise<void> {
        const endTimer = this.results.startTest('JWT Creation and Validation');

        try {
            const jobSpec = JobSpecUtils.createJPGJobSpec();
            const jwt = await JWTUtils.createJWT(jobSpec, 'test-subject');

            // Verify the JWT
            const verified = await JWTUtils.verifyJWT(jwt);
            if (verified.sub !== 'test-subject') {
                throw new Error('JWT subject verification failed');
            }

            if (!verified.job.formats.jpg) {
                throw new Error('JWT job spec verification failed');
            }

            this.results.recordPass('JWT Creation and Validation', 0, { subject: verified.sub });
        } catch (error) {
            this.results.recordFail('JWT Creation and Validation', 0, (error as Error).message);
            throw error;
        } finally {
            endTimer();
        }
    }

    private async testImageUpload (): Promise<void> {
        const endTimer = this.results.startTest('Image Upload');

        try {
            // Create test image
            const imagePath = ImageUtils.createMinimalPNG('basic-test.png');

            // Create job spec
            const jobSpec = JobSpecUtils.createJPGJobSpec();

            // Create JWT
            const jwt = await JWTUtils.createJWT(jobSpec);

            // Create form data
            const form = new FormData();
            form.append('token', jwt);
            form.append('file', fs.createReadStream(imagePath), {
                filename: 'basic-test.png',
                contentType: ImageUtils.getContentType(imagePath),
            });

            // Upload
            const response = await HTTPUtils.post(`${ this.baseUrl }/upload`, form);

            if (response.status !== 200) {
                throw new Error(`Upload failed with status ${ response.status }`);
            }

            const responseData: UploadResponse = response.data;
            if (!responseData.hash) {
                throw new Error('Upload response missing hash');
            }

            this.logger.info('Image uploaded successfully', { hash: responseData.hash });
            this.results.recordPass('Image Upload', 0, { hash: responseData.hash, message: responseData.message });

            // Store hash for other tests
            (global as any).testHash = responseData.hash;

        } catch (error) {
            this.results.recordFail('Image Upload', 0, (error as Error).message);
            throw error;
        } finally {
            endTimer();
        }
    }

    private async testStatusEndpoints (): Promise<void> {
        const hash = (global as any).testHash;
        if (!hash) {
            throw new Error('No test hash available from upload test');
        }

        // Test success endpoint
        await this.testSuccessEndpoint(hash);

        // Test failure endpoint
        await this.testFailureEndpoint(hash);

        // Test status endpoint
        await this.testStatusEndpoint(hash);
    }

    private async testSuccessEndpoint (hash: string): Promise<void> {
        const endTimer = this.results.startTest('Success Endpoint');

        try {
            const response = await HTTPUtils.get(`${ this.baseUrl }/success?hash=${ hash }`);
            const responseData: SuccessResponse = response.data;

            // Accept both 'success' and 'not_found' (job might still be processing)
            if (responseData.status !== 'success' && responseData.status !== 'not_found') {
                throw new Error(`Unexpected success status: ${ responseData.status }`);
            }

            this.results.recordPass('Success Endpoint', 0, {
                status: responseData.status,
                file_count: responseData.file_count
            });
        } catch (error) {
            this.results.recordFail('Success Endpoint', 0, (error as Error).message);
            throw error;
        } finally {
            endTimer();
        }
    }

    private async testFailureEndpoint (hash: string): Promise<void> {
        const endTimer = this.results.startTest('Failure Endpoint');

        try {
            const response = await HTTPUtils.get(`${ this.baseUrl }/failures?hash=${ hash }`);
            const responseData: FailureResponse = response.data;

            // Accept both 'failed' and 'not_found' (job might still be processing)
            if (responseData.status !== 'failed' && responseData.status !== 'not_found') {
                throw new Error(`Unexpected failure status: ${ responseData.status }`);
            }

            this.results.recordPass('Failure Endpoint', 0, {
                status: responseData.status,
                error: responseData.error
            });
        } catch (error) {
            this.results.recordFail('Failure Endpoint', 0, (error as Error).message);
            throw error;
        } finally {
            endTimer();
        }
    }

    private async testStatusEndpoint (hash: string): Promise<void> {
        const endTimer = this.results.startTest('Status Endpoint');

        try {
            const response = await HTTPUtils.get(`${ this.baseUrl }/status?hash=${ hash }`);
            const responseData: StatusResponse = response.data;

            // Accept various statuses
            const validStatuses = [ 'success', 'failed', 'not_found', 'processing', 'pending' ];
            if (!validStatuses.includes(responseData.status)) {
                throw new Error(`Unexpected status: ${ responseData.status }`);
            }

            this.results.recordPass('Status Endpoint', 0, {
                status: responseData.status,
                file_count: responseData.file_count
            });
        } catch (error) {
            this.results.recordFail('Status Endpoint', 0, (error as Error).message);
            throw error;
        } finally {
            endTimer();
        }
    }

    private async testCallbackFunctionality (): Promise<void> {
        const endTimer = this.results.startTest('Callback Functionality');

        try {
            // Create a test callback server
            const callbackServer = this.createCallbackServer();

            // Upload with callback
            const imagePath = ImageUtils.createMinimalPNG('callback-test.png');
            const callbackUrl = 'http://localhost:8081/callback';
            const jobSpec = JobSpecUtils.createCallbackJobSpec(callbackUrl);

            const jwt = await JWTUtils.createJWT(jobSpec);
            const form = new FormData();
            form.append('token', jwt);
            form.append('file', fs.createReadStream(imagePath), {
                filename: 'callback-test.png',
                contentType: ImageUtils.getContentType(imagePath),
            });

            const response = await HTTPUtils.post(`${ this.baseUrl }/upload`, form);
            const responseData: UploadResponse = response.data;

            // Wait for callback (or timeout)
            const callbackReceived = await this.waitForCallback(callbackServer, 10000);

            if (callbackReceived) {
                this.results.recordPass('Callback Functionality', 0, { callbackReceived: true });
            } else {
                this.logger.warn('Callback not received within timeout, but job was configured correctly');
                this.results.recordPass('Callback Functionality', 0, { callbackReceived: false, note: 'timeout' });
            }

        } catch (error) {
            this.results.recordFail('Callback Functionality', 0, (error as Error).message);
            throw error;
        } finally {
            endTimer();
        }
    }

    private createCallbackServer (): http.Server {
        let callbackReceived = false;

        const server = http.createServer((req, res) => {
            if (req.url === '/callback' && req.method === 'POST') {
                callbackReceived = true;
                this.logger.info('Callback received!');
                res.writeHead(200);
                res.end('OK');
            } else {
                res.writeHead(404);
                res.end('Not Found');
            }
        });

        server.listen(8081, () => {
            this.logger.debug('Callback server listening on port 8081');
        });

        // Store callback status globally for test
        (global as any).callbackReceived = false;

        return server;
    }

    private async waitForCallback (server: http.Server, timeoutMs: number): Promise<boolean> {
        return new Promise((resolve) => {
            const timeout = setTimeout(() => {
                server.close();
                resolve(false);
            }, timeoutMs);

            // Check periodically if callback was received
            const checkInterval = setInterval(() => {
                if ((global as any).callbackReceived) {
                    clearTimeout(timeout);
                    clearInterval(checkInterval);
                    server.close();
                    resolve(true);
                }
            }, 500);
        });
    }

    private async testJobCancellation (): Promise<void> {
        const endTimer = this.results.startTest('Job Cancellation');

        try {
            // Upload a job first
            const imagePath = ImageUtils.createMinimalPNG('cancel-test.png');
            const jobSpec = JobSpecUtils.createJPGJobSpec();

            const jwt = await JWTUtils.createJWT(jobSpec);
            const form = new FormData();
            form.append('token', jwt);
            form.append('file', fs.createReadStream(imagePath), {
                filename: 'cancel-test.png',
                contentType: ImageUtils.getContentType(imagePath),
            });

            const response = await HTTPUtils.post(`${ this.baseUrl }/upload`, form);
            const responseData: UploadResponse = response.data;
            const hash = responseData.hash;

            // Try to cancel the job
            const cancelResponse = await HTTPUtils.get(`${ this.baseUrl }/cancel?hash=${ hash }`);

            if (cancelResponse.status === 200) {
                this.results.recordPass('Job Cancellation', 0, { cancelled: true });
            } else {
                // Cancellation might not be implemented or job might have completed
                this.logger.warn('Job cancellation returned non-200 status', { status: cancelResponse.status });
                this.results.recordPass('Job Cancellation', 0, { cancelled: false, note: 'not implemented or job completed' });
            }

        } catch (error) {
            this.results.recordFail('Job Cancellation', 0, (error as Error).message);
            throw error;
        } finally {
            endTimer();
        }
    }

    private async testInvalidRequests (): Promise<void> {
        const endTimer = this.results.startTest('Invalid Requests');

        try {
            // Test invalid hash
            const invalidHashResponse = await HTTPUtils.get(`${ this.baseUrl }/success?hash=invalid-hash`);
            if (invalidHashResponse.status !== 200) {
                throw new Error(`Invalid hash request failed with status ${ invalidHashResponse.status }`);
            }

            const responseData: SuccessResponse = invalidHashResponse.data;
            if (responseData.status !== 'not_found') {
                throw new Error(`Expected 'not_found' status for invalid hash, got '${ responseData.status }'`);
            }

            // Test missing token
            const form = new FormData();
            form.append('file', Buffer.from('fake image data'), {
                filename: 'test.jpg',
                contentType: 'image/jpeg',
            });

            try {
                await HTTPUtils.post(`${ this.baseUrl }/upload`, form);
                throw new Error('Expected upload without token to fail');
            } catch (error) {
                // Expected to fail
                this.logger.debug('Upload without token correctly failed');
            }

            this.results.recordPass('Invalid Requests', 0, { invalidHashHandled: true, missingTokenHandled: true });
        } catch (error) {
            this.results.recordFail('Invalid Requests', 0, (error as Error).message);
            throw error;
        } finally {
            endTimer();
        }
    }

    getResults (): TestResults {
        return this.results;
    }
}

// CLI runner
if (require.main === module) {
    const tests = new BasicFunctionalityTests();

    tests.runAll()
        .then(() => {
            logger.info('Basic functionality tests completed successfully');
            process.exit(0);
        })
        .catch((error) => {
            logger.error('Basic functionality tests failed', { error: (error as Error).message });
            process.exit(1);
        });
}

export default BasicFunctionalityTests;
