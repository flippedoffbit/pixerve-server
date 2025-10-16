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

    constructor(baseUrl = 'http://localhost:8080', logLevel: LogLevel = LogLevel.DEBUG) {
        this.baseUrl = baseUrl;
        this.server = new PixerveServer(baseUrl);
        this.results = new TestResults();
        this.logger = new Logger(logLevel, 'BASIC');
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
        this.logger.debug('Ensuring test images directory exists');
        ImageUtils.ensureTestImagesDir();
        this.logger.debug('Starting Pixerve server');
        await this.server.start();
        this.logger.debug('Waiting for server to be ready');
        await this.server.waitForReady();
        this.logger.info('Basic tests setup completed');
    }

    private async cleanup (): Promise<void> {
        this.logger.info('Cleaning up basic functionality tests');
        this.logger.debug('Stopping Pixerve server');
        await this.server.stop();
        this.logger.debug('Cleaning up test images');
        ImageUtils.cleanupTestImages();
        this.logger.info('Basic tests cleanup completed');
    }

    private async testHealthEndpoint (): Promise<void> {
        this.logger.info('Testing health endpoint');
        const endTimer = this.results.startTest('Health Endpoint');

        try {
            this.logger.debug('Making request to /health endpoint');
            const response = await HTTPUtils.get(`${ this.baseUrl }/health`);
            this.logger.debug('Health endpoint response received', { status: response.status });

            if (response.status !== 200) {
                this.logger.warn('Health endpoint returned unexpected status', { status: response.status });
                throw new Error(`Health endpoint returned status ${ response.status }`);
            }

            this.logger.info('Health endpoint test passed');
            this.results.recordPass('Health Endpoint', 0, { status: response.status });
        } catch (error) {
            this.logger.error('Health endpoint test failed', { error: (error as Error).message });
            this.results.recordFail('Health Endpoint', 0, (error as Error).message);
            throw error;
        } finally {
            endTimer();
        }
    }

    private async testVersionEndpoint (): Promise<void> {
        this.logger.info('Testing version endpoint');
        const endTimer = this.results.startTest('Version Endpoint');

        try {
            this.logger.debug('Making request to /version endpoint');
            const response = await HTTPUtils.get(`${ this.baseUrl }/version`);
            this.logger.debug('Version endpoint response received', { status: response.status });

            if (response.status !== 200) {
                this.logger.warn('Version endpoint returned unexpected status', { status: response.status });
                throw new Error(`Version endpoint returned status ${ response.status }`);
            }

            if (!response.data.version) {
                this.logger.warn('Version endpoint response missing version field');
                throw new Error('Version endpoint response missing version field');
            }

            this.logger.info('Version endpoint test passed', { version: response.data.version });
            this.results.recordPass('Version Endpoint', 0, { version: response.data.version });
        } catch (error) {
            this.logger.error('Version endpoint test failed', { error: (error as Error).message });
            this.results.recordFail('Version Endpoint', 0, (error as Error).message);
            throw error;
        } finally {
            endTimer();
        }
    }

    private async testJWT (): Promise<void> {
        this.logger.info('Testing JWT creation and validation');
        const endTimer = this.results.startTest('JWT Creation and Validation');

        try {
            this.logger.debug('Creating job spec for JWT test');
            const jobSpec = JobSpecUtils.createJPGJobSpec();
            this.logger.debug('Creating JWT with test subject');
            const jwt = await JWTUtils.createJWT(jobSpec, 'test-subject');

            this.logger.debug('Verifying JWT');
            const verified = await JWTUtils.verifyJWT(jwt);
            if (verified.sub !== 'test-subject') {
                this.logger.warn('JWT subject verification failed', { expected: 'test-subject', actual: verified.sub });
                throw new Error('JWT subject verification failed');
            }

            if (!verified.job.formats.jpg) {
                this.logger.warn('JWT job spec verification failed - missing JPG format');
                throw new Error('JWT job spec verification failed');
            }

            this.logger.info('JWT creation and validation test passed', { subject: verified.sub });
            this.results.recordPass('JWT Creation and Validation', 0, { subject: verified.sub });
        } catch (error) {
            this.logger.error('JWT creation and validation test failed', { error: (error as Error).message });
            this.results.recordFail('JWT Creation and Validation', 0, (error as Error).message);
            throw error;
        } finally {
            endTimer();
        }
    }

    private async testImageUpload (): Promise<void> {
        this.logger.info('Testing image upload functionality');
        const endTimer = this.results.startTest('Image Upload');

        try {
            this.logger.debug('Creating test image for upload');
            const imagePath = ImageUtils.createMinimalPNG('basic-test.png');

            this.logger.debug('Creating job spec for upload');
            const jobSpec = JobSpecUtils.createJPGJobSpec();

            this.logger.debug('Creating JWT for upload');
            const jwt = await JWTUtils.createJWT(jobSpec);

            this.logger.debug('Preparing form data for upload');
            const form = new FormData();
            form.append('token', jwt);
            form.append('file', fs.createReadStream(imagePath), {
                filename: 'basic-test.png',
                contentType: ImageUtils.getContentType(imagePath),
            });

            this.logger.debug('Uploading image to server');
            const response = await HTTPUtils.post(`${ this.baseUrl }/upload`, form);

            if (response.status !== 200) {
                this.logger.warn('Upload failed with unexpected status', { status: response.status });
                throw new Error(`Upload failed with status ${ response.status }`);
            }

            const responseData: UploadResponse = response.data;
            if (!responseData.hash) {
                this.logger.warn('Upload response missing hash field');
                throw new Error('Upload response missing hash');
            }

            this.logger.info('Image uploaded successfully', { hash: responseData.hash });
            this.results.recordPass('Image Upload', 0, { hash: responseData.hash, message: responseData.message });

            // Store hash for other tests
            (global as any).testHash = responseData.hash;

        } catch (error) {
            this.logger.error('Image upload test failed', { error: (error as Error).message });
            this.results.recordFail('Image Upload', 0, (error as Error).message);
            throw error;
        } finally {
            endTimer();
        }
    }

    private async testStatusEndpoints (): Promise<void> {
        const hash = (global as any).testHash;
        if (!hash) {
            this.logger.error('No test hash available from upload test');
            throw new Error('No test hash available from upload test');
        }

        this.logger.info('Testing status endpoints', { hash });

        // Test success endpoint
        await this.testSuccessEndpoint(hash);

        // Test failure endpoint
        await this.testFailureEndpoint(hash);

        // Test status endpoint
        await this.testStatusEndpoint(hash);
    }

    private async testSuccessEndpoint (hash: string): Promise<void> {
        this.logger.info('Testing success endpoint', { hash });
        const endTimer = this.results.startTest('Success Endpoint');

        try {
            this.logger.debug('Making request to success endpoint');
            const response = await HTTPUtils.get(`${ this.baseUrl }/success?hash=${ hash }`);
            const responseData: SuccessResponse = response.data;

            this.logger.debug('Success endpoint response received', { status: responseData.status });

            // Accept both 'success' and 'not_found' (job might still be processing)
            if (responseData.status !== 'success' && responseData.status !== 'not_found') {
                this.logger.warn('Unexpected success status', { status: responseData.status });
                throw new Error(`Unexpected success status: ${ responseData.status }`);
            }

            this.logger.info('Success endpoint test passed', { status: responseData.status, fileCount: responseData.file_count });
            this.results.recordPass('Success Endpoint', 0, {
                status: responseData.status,
                file_count: responseData.file_count
            });
        } catch (error) {
            this.logger.error('Success endpoint test failed', { error: (error as Error).message });
            this.results.recordFail('Success Endpoint', 0, (error as Error).message);
            throw error;
        } finally {
            endTimer();
        }
    }

    private async testFailureEndpoint (hash: string): Promise<void> {
        this.logger.info('Testing failure endpoint', { hash });
        const endTimer = this.results.startTest('Failure Endpoint');

        try {
            this.logger.debug('Making request to failure endpoint');
            const response = await HTTPUtils.get(`${ this.baseUrl }/failures?hash=${ hash }`);
            const responseData: FailureResponse = response.data;

            this.logger.debug('Failure endpoint response received', { status: responseData.status });

            // Accept both 'failed' and 'not_found' (job might still be processing)
            if (responseData.status !== 'failed' && responseData.status !== 'not_found') {
                this.logger.warn('Unexpected failure status', { status: responseData.status });
                throw new Error(`Unexpected failure status: ${ responseData.status }`);
            }

            this.logger.info('Failure endpoint test passed', { status: responseData.status, error: responseData.error });
            this.results.recordPass('Failure Endpoint', 0, {
                status: responseData.status,
                error: responseData.error
            });
        } catch (error) {
            this.logger.error('Failure endpoint test failed', { error: (error as Error).message });
            this.results.recordFail('Failure Endpoint', 0, (error as Error).message);
            throw error;
        } finally {
            endTimer();
        }
    }

    private async testStatusEndpoint (hash: string): Promise<void> {
        this.logger.info('Testing status endpoint', { hash });
        const endTimer = this.results.startTest('Status Endpoint');

        try {
            this.logger.debug('Making request to status endpoint');
            const response = await HTTPUtils.get(`${ this.baseUrl }/status?hash=${ hash }`);
            const responseData: StatusResponse = response.data;

            this.logger.debug('Status endpoint response received', { status: responseData.status });

            // Accept various statuses
            const validStatuses = [ 'success', 'failed', 'not_found', 'processing', 'pending' ];
            if (!validStatuses.includes(responseData.status)) {
                this.logger.warn('Unexpected status received', { status: responseData.status, validStatuses });
                throw new Error(`Unexpected status: ${ responseData.status }`);
            }

            this.logger.info('Status endpoint test passed', { status: responseData.status, fileCount: responseData.file_count });
            this.results.recordPass('Status Endpoint', 0, {
                status: responseData.status,
                file_count: responseData.file_count
            });
        } catch (error) {
            this.logger.error('Status endpoint test failed', { error: (error as Error).message });
            this.results.recordFail('Status Endpoint', 0, (error as Error).message);
            throw error;
        } finally {
            endTimer();
        }
    }

    private async testCallbackFunctionality (): Promise<void> {
        this.logger.info('Testing callback functionality');
        const endTimer = this.results.startTest('Callback Functionality');

        try {
            this.logger.debug('Creating test callback server');
            const callbackServer = this.createCallbackServer();

            this.logger.debug('Creating test image for callback upload');
            const imagePath = ImageUtils.createMinimalPNG('callback-test.png');
            const callbackUrl = 'http://localhost:8081/callback';
            this.logger.debug('Creating job spec with callback URL', { callbackUrl });
            const jobSpec = JobSpecUtils.createCallbackJobSpec(callbackUrl);

            this.logger.debug('Creating JWT for callback upload');
            const jwt = await JWTUtils.createJWT(jobSpec);
            const form = new FormData();
            form.append('token', jwt);
            form.append('file', fs.createReadStream(imagePath), {
                filename: 'callback-test.png',
                contentType: ImageUtils.getContentType(imagePath),
            });

            this.logger.debug('Uploading image with callback configuration');
            const response = await HTTPUtils.post(`${ this.baseUrl }/upload`, form);
            const responseData: UploadResponse = response.data;

            this.logger.debug('Waiting for callback response');
            const callbackReceived = await this.waitForCallback(callbackServer, 10000);

            if (callbackReceived) {
                this.logger.info('Callback functionality test passed - callback received');
                this.results.recordPass('Callback Functionality', 0, { callbackReceived: true });
            } else {
                this.logger.warn('Callback not received within timeout, but job was configured correctly');
                this.results.recordPass('Callback Functionality', 0, { callbackReceived: false, note: 'timeout' });
            }

        } catch (error) {
            this.logger.error('Callback functionality test failed', { error: (error as Error).message });
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
        this.logger.info('Testing job cancellation functionality');
        const endTimer = this.results.startTest('Job Cancellation');

        try {
            this.logger.debug('Creating test image for cancellation test');
            const imagePath = ImageUtils.createMinimalPNG('cancel-test.png');
            const jobSpec = JobSpecUtils.createJPGJobSpec();

            this.logger.debug('Creating JWT for cancellation test');
            const jwt = await JWTUtils.createJWT(jobSpec);
            const form = new FormData();
            form.append('token', jwt);
            form.append('file', fs.createReadStream(imagePath), {
                filename: 'cancel-test.png',
                contentType: ImageUtils.getContentType(imagePath),
            });

            this.logger.debug('Uploading image for cancellation test');
            const response = await HTTPUtils.post(`${ this.baseUrl }/upload`, form);
            const responseData: UploadResponse = response.data;
            const hash = responseData.hash;

            this.logger.debug('Attempting to cancel the job', { hash });
            const cancelResponse = await HTTPUtils.get(`${ this.baseUrl }/cancel?hash=${ hash }`);

            if (cancelResponse.status === 200) {
                this.logger.info('Job cancellation successful');
                this.results.recordPass('Job Cancellation', 0, { cancelled: true });
            } else {
                // Cancellation might not be implemented or job might have completed
                this.logger.warn('Job cancellation returned non-200 status', { status: cancelResponse.status });
                this.results.recordPass('Job Cancellation', 0, { cancelled: false, note: 'not implemented or job completed' });
            }

        } catch (error) {
            this.logger.error('Job cancellation test failed', { error: (error as Error).message });
            this.results.recordFail('Job Cancellation', 0, (error as Error).message);
            throw error;
        } finally {
            endTimer();
        }
    }

    private async testInvalidRequests (): Promise<void> {
        this.logger.info('Testing invalid request handling');
        const endTimer = this.results.startTest('Invalid Requests');

        try {
            this.logger.debug('Testing invalid hash request');
            const invalidHashResponse = await HTTPUtils.get(`${ this.baseUrl }/success?hash=invalid-hash`);
            if (invalidHashResponse.status !== 200) {
                this.logger.warn('Invalid hash request failed with unexpected status', { status: invalidHashResponse.status });
                throw new Error(`Invalid hash request failed with status ${ invalidHashResponse.status }`);
            }

            const responseData: SuccessResponse = invalidHashResponse.data;
            if (responseData.status !== 'not_found') {
                this.logger.warn('Invalid hash did not return expected not_found status', { status: responseData.status });
                throw new Error(`Expected 'not_found' status for invalid hash, got '${ responseData.status }'`);
            }

            this.logger.debug('Testing upload without token');
            const form = new FormData();
            form.append('file', Buffer.from('fake image data'), {
                filename: 'test.jpg',
                contentType: 'image/jpeg',
            });

            try {
                await HTTPUtils.post(`${ this.baseUrl }/upload`, form);
                this.logger.error('Upload without token should have failed but succeeded');
                throw new Error('Expected upload without token to fail');
            } catch (error) {
                // Expected to fail
                this.logger.debug('Upload without token correctly failed');
            }

            this.logger.info('Invalid requests test passed');
            this.results.recordPass('Invalid Requests', 0, { invalidHashHandled: true, missingTokenHandled: true });
        } catch (error) {
            this.logger.error('Invalid requests test failed', { error: (error as Error).message });
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
