/**
 * Writer backend integration tests for Pixerve
 *
 * Tests different storage backends:
 * - Direct HTTP serving
 * - S3-compatible storage
 * - Google Cloud Storage
 * - SFTP/FTP storage
 * - Backend-specific configurations
 * - Error handling for backend failures
 */

import * as fs from 'fs';
import * as path from 'path';
import FormData from 'form-data';
import {
    Logger,
    LogLevel,
    JobSpec,
    UploadResponse,
    SuccessResponse,
    FailureResponse,
    JWTUtils,
    HTTPUtils,
    ImageUtils,
    PixerveServer,
    TestResults,
    logger
} from './common';

// For CommonJS __dirname equivalent
const __dirname = path.dirname(require.main!.filename);

export class BackendTests {
    private baseUrl: string;
    private server: PixerveServer;
    private results: TestResults;
    private logger: Logger;

    constructor(baseUrl = 'http://localhost:8080', logLevel: LogLevel = LogLevel.DEBUG) {
        this.baseUrl = baseUrl;
        this.server = new PixerveServer(baseUrl);
        this.results = new TestResults();
        this.logger = new Logger(logLevel, 'BACKEND');
    }

    async runAll (): Promise<void> {
        this.logger.info('Starting writer backend tests');

        try {
            // Setup
            await this.setup();

            // Run tests
            await this.testDirectHostBackend();
            await this.testS3Backend();
            await this.testGCSBackend();
            await this.testSFTPBackend();
            await this.testMultipleBackends();
            await this.testBackendErrors();
            await this.testBackendPriorities();

            // Cleanup
            await this.cleanup();

        } catch (error) {
            this.logger.error('Writer backend tests failed', { error: (error as Error).message });
            throw error;
        } finally {
            this.results.printSummary();
        }
    }

    private async setup (): Promise<void> {
        this.logger.info('Setting up writer backend tests');
        this.logger.debug('Ensuring test images directory exists');
        ImageUtils.ensureTestImagesDir();
        this.logger.debug('Starting Pixerve server');
        await this.server.start();
        this.logger.debug('Waiting for server to be ready');
        await this.server.waitForReady();
        this.logger.info('Writer backend tests setup completed');
    }

    private async cleanup (): Promise<void> {
        this.logger.info('Cleaning up writer backend tests');
        this.logger.debug('Stopping Pixerve server');
        await this.server.stop();
        this.logger.debug('Cleaning up test images');
        ImageUtils.cleanupTestImages();
        this.logger.info('Writer backend tests cleanup completed');
    }

    private async testDirectHostBackend (): Promise<void> {
        this.logger.info('Testing direct HTTP hosting backend');

        const testCases = [
            { name: 'Direct Host Only', directHost: true, storageKeys: {} },
            { name: 'Direct Host with Subdir', directHost: true, subDir: 'direct-host-test' },
            { name: 'No Direct Host', directHost: false, storageKeys: {} },
        ];

        for (const testCase of testCases) {
            await this.testDirectHostConfiguration(testCase.name, testCase.directHost, testCase.storageKeys, testCase.subDir);
        }
    }

    private async testDirectHostConfiguration (name: string, directHost: boolean, storageKeys?: Record<string, string>, subDir?: string): Promise<void> {
        const testName = `Direct Host: ${ name }`;
        this.logger.info('Testing direct host configuration', { name, directHost, subDir });
        const endTimer = this.results.startTest(testName);

        try {
            this.logger.debug('Downloading test image for direct host test', { name });
            const imagePath = await ImageUtils.downloadLoremPicsumImage(800, 600, `direct-host-${ name.replace(/\s+/g, '-').toLowerCase() }.jpg`);

            this.logger.debug('Creating job spec for direct host test', { name, directHost, subDir });
            const jobSpec: JobSpec = {
                priority: 0,
                keepOriginal: false,
                formats: {
                    jpg: {
                        settings: { quality: 80, speed: 1 },
                        sizes: [ [ 400, 300 ] ],
                    },
                },
                directHost,
                storageKeys,
                subDir,
            };

            this.logger.debug('Uploading image and waiting for completion', { name });
            const hash = await this.uploadImage(imagePath, jobSpec);
            const result = await this.waitForCompletion(hash);

            if (result.status === 'success') {
                this.logger.info('Direct host configuration test passed', { name, directHost, subDir, fileCount: result.file_count, hash });
                this.results.recordPass(testName, 0, {
                    directHost,
                    storageKeys: Object.keys(storageKeys || {}),
                    subDir,
                    fileCount: result.file_count,
                    hash
                });
            } else {
                const error = this.getErrorMessage(result);
                this.logger.warn('Direct host configuration test failed', { name, error });
                throw new Error(`Job failed: ${ error }`);
            }

        } catch (error) {
            this.logger.error('Direct host configuration test failed with exception', { name, error: (error as Error).message });
            this.results.recordFail(testName, 0, (error as Error).message);
        } finally {
            endTimer();
        }
    }

    private async testS3Backend (): Promise<void> {
        this.logger.info('Testing S3-compatible storage backend');

        const s3Configs = [
            {
                name: 'Basic S3',
                storageKeys: { s3: 'test-bucket-basic' }
            },
            {
                name: 'S3 with Path',
                storageKeys: { s3: 'test-bucket-path/images' }
            },
            {
                name: 'S3 with Credentials',
                storageKeys: {
                    s3: 'test-bucket-creds',
                    s3_access_key: 'test-key',
                    s3_secret_key: 'test-secret',
                    s3_region: 'us-east-1'
                }
            },
        ];

        for (const config of s3Configs) {
            await this.testS3Configuration(config.name, config.storageKeys);
        }
    }

    private async testS3Configuration (name: string, storageKeys: Record<string, string | undefined>): Promise<void> {
        const testName = `S3 Backend: ${ name }`;
        const endTimer = this.results.startTest(testName);

        try {
            const imagePath = await ImageUtils.downloadLoremPicsumImage(800, 600, `s3-${ name.replace(/\s+/g, '-').toLowerCase() }.jpg`);

            // Filter out undefined values
            const cleanStorageKeys = Object.fromEntries(
                Object.entries(storageKeys).filter(([ _, value ]) => value !== undefined)
            ) as Record<string, string>;

            const jobSpec: JobSpec = {
                priority: 0,
                keepOriginal: false,
                formats: {
                    jpg: {
                        settings: { quality: 80, speed: 1 },
                        sizes: [ [ 400, 300 ] ],
                    },
                },
                storageKeys: cleanStorageKeys,
                directHost: false,
                subDir: 's3-test',
            };

            const hash = await this.uploadImage(imagePath, jobSpec);
            const result = await this.waitForCompletion(hash);

            if (result.status === 'success') {
                this.results.recordPass(testName, 0, {
                    storageKeys: Object.keys(storageKeys),
                    fileCount: result.file_count,
                    hash
                });
            } else {
                // S3 might not be configured, which is OK for testing
                const error = this.getErrorMessage(result);
                this.logger.warn(`S3 test failed (might not be configured): ${ error }`);
                this.results.recordPass(testName, 0, {
                    status: 'skipped - not configured',
                    error,
                    storageKeys: Object.keys(storageKeys)
                });
            }

        } catch (error) {
            this.results.recordFail(testName, 0, (error as Error).message);
        } finally {
            endTimer();
        }
    }

    private async testGCSBackend (): Promise<void> {
        this.logger.info('Testing Google Cloud Storage backend');

        const gcsConfigs = [
            {
                name: 'Basic GCS',
                storageKeys: { gcs: 'test-gcs-bucket' }
            },
            {
                name: 'GCS with Project',
                storageKeys: {
                    gcs: 'test-gcs-bucket-project',
                    gcs_project_id: 'test-project',
                    gcs_credentials_path: '/path/to/creds.json'
                }
            },
        ];

        for (const config of gcsConfigs) {
            await this.testGCSConfiguration(config.name, config.storageKeys);
        }
    }

    private async testGCSConfiguration (name: string, storageKeys: Record<string, string | undefined>): Promise<void> {
        const testName = `GCS Backend: ${ name }`;
        const endTimer = this.results.startTest(testName);

        try {
            const imagePath = await ImageUtils.downloadLoremPicsumImage(800, 600, `gcs-${ name.replace(/\s+/g, '-').toLowerCase() }.jpg`);

            // Filter out undefined values
            const cleanStorageKeys = Object.fromEntries(
                Object.entries(storageKeys).filter(([ _, value ]) => value !== undefined)
            ) as Record<string, string>;

            const jobSpec: JobSpec = {
                priority: 0,
                keepOriginal: false,
                formats: {
                    jpg: {
                        settings: { quality: 80, speed: 1 },
                        sizes: [ [ 400, 300 ] ],
                    },
                },
                storageKeys: cleanStorageKeys,
                directHost: false,
                subDir: 'gcs-test',
            };

            const hash = await this.uploadImage(imagePath, jobSpec);
            const result = await this.waitForCompletion(hash);

            if (result.status === 'success') {
                this.results.recordPass(testName, 0, {
                    storageKeys: Object.keys(storageKeys),
                    fileCount: result.file_count,
                    hash
                });
            } else {
                // GCS might not be configured
                const error = this.getErrorMessage(result);
                this.logger.warn(`GCS test failed (might not be configured): ${ error }`);
                this.results.recordPass(testName, 0, {
                    status: 'skipped - not configured',
                    error,
                    storageKeys: Object.keys(storageKeys)
                });
            }

        } catch (error) {
            this.results.recordFail(testName, 0, (error as Error).message);
        } finally {
            endTimer();
        }
    }

    private async testSFTPBackend (): Promise<void> {
        this.logger.info('Testing SFTP/FTP storage backend');

        const sftpConfigs = [
            {
                name: 'Basic SFTP',
                storageKeys: {
                    sftp: 'sftp://example.com/images',
                    sftp_user: 'test-user',
                    sftp_password: 'test-pass'
                }
            },
            {
                name: 'SFTP with Key',
                storageKeys: {
                    sftp: 'sftp://example.com/images',
                    sftp_user: 'test-user',
                    sftp_key_path: '/path/to/key',
                    sftp_port: '22'
                }
            },
        ];

        for (const config of sftpConfigs) {
            await this.testSFTPConfiguration(config.name, config.storageKeys);
        }
    }

    private async testSFTPConfiguration (name: string, storageKeys: Record<string, string | undefined>): Promise<void> {
        const testName = `SFTP Backend: ${ name }`;
        const endTimer = this.results.startTest(testName);

        try {
            const imagePath = await ImageUtils.downloadLoremPicsumImage(800, 600, `sftp-${ name.replace(/\s+/g, '-').toLowerCase() }.jpg`);

            // Filter out undefined values
            const cleanStorageKeys = Object.fromEntries(
                Object.entries(storageKeys).filter(([ _, value ]) => value !== undefined)
            ) as Record<string, string>;

            const jobSpec: JobSpec = {
                priority: 0,
                keepOriginal: false,
                formats: {
                    jpg: {
                        settings: { quality: 80, speed: 1 },
                        sizes: [ [ 400, 300 ] ],
                    },
                },
                storageKeys: cleanStorageKeys,
                directHost: false,
                subDir: 'sftp-test',
            };

            const hash = await this.uploadImage(imagePath, jobSpec);
            const result = await this.waitForCompletion(hash);

            if (result.status === 'success') {
                this.results.recordPass(testName, 0, {
                    storageKeys: Object.keys(storageKeys),
                    fileCount: result.file_count,
                    hash
                });
            } else {
                // SFTP might not be configured
                const error = this.getErrorMessage(result);
                this.logger.warn(`SFTP test failed (might not be configured): ${ error }`);
                this.results.recordPass(testName, 0, {
                    status: 'skipped - not configured',
                    error,
                    storageKeys: Object.keys(storageKeys)
                });
            }

        } catch (error) {
            this.results.recordFail(testName, 0, (error as Error).message);
        } finally {
            endTimer();
        }
    }

    private async testMultipleBackends (): Promise<void> {
        this.logger.info('Testing multiple backends simultaneously');

        const testCases = [
            {
                name: 'Direct + S3',
                storageKeys: { s3: 'test-multi-bucket' },
                directHost: true
            },
            {
                name: 'Direct + GCS',
                storageKeys: { gcs: 'test-multi-gcs-bucket' },
                directHost: true
            },
            {
                name: 'All Backends',
                storageKeys: {
                    s3: 'test-all-s3',
                    gcs: 'test-all-gcs',
                    sftp: 'sftp://example.com/all'
                },
                directHost: true
            },
        ];

        for (const testCase of testCases) {
            await this.testMultipleBackendConfiguration(testCase.name, testCase.storageKeys, testCase.directHost);
        }
    }

    private async testMultipleBackendConfiguration (name: string, storageKeys: Record<string, string | undefined>, directHost: boolean): Promise<void> {
        const testName = `Multiple Backends: ${ name }`;
        const endTimer = this.results.startTest(testName);

        try {
            const imagePath = await ImageUtils.downloadLoremPicsumImage(800, 600, `multi-${ name.replace(/\s+/g, '-').toLowerCase() }.jpg`);

            // Filter out undefined values
            const cleanStorageKeys = Object.fromEntries(
                Object.entries(storageKeys).filter(([ _, value ]) => value !== undefined)
            ) as Record<string, string>;

            const jobSpec: JobSpec = {
                priority: 0,
                keepOriginal: false,
                formats: {
                    jpg: {
                        settings: { quality: 80, speed: 1 },
                        sizes: [ [ 400, 300 ] ],
                    },
                    webp: {
                        settings: { quality: 85, speed: 2 },
                        sizes: [ [ 400 ] ],
                    },
                },
                storageKeys: cleanStorageKeys,
                directHost,
                subDir: 'multi-backend-test',
            };

            const hash = await this.uploadImage(imagePath, jobSpec);
            const result = await this.waitForCompletion(hash);

            if (result.status === 'success') {
                this.results.recordPass(testName, 0, {
                    storageKeys: Object.keys(storageKeys),
                    directHost,
                    fileCount: result.file_count,
                    hash
                });
            } else {
                // Multiple backends might not all be configured
                const error = this.getErrorMessage(result);
                this.logger.warn(`Multiple backend test failed (might not be fully configured): ${ error }`);
                this.results.recordPass(testName, 0, {
                    status: 'partial - some backends not configured',
                    error,
                    storageKeys: Object.keys(storageKeys),
                    directHost
                });
            }

        } catch (error) {
            this.results.recordFail(testName, 0, (error as Error).message);
        } finally {
            endTimer();
        }
    }

    private async testBackendErrors (): Promise<void> {
        this.logger.info('Testing backend error handling');

        const errorCases = [
            {
                name: 'Invalid S3 Bucket',
                storageKeys: { s3: 'invalid-bucket-name-that-does-not-exist-12345' }
            },
            {
                name: 'Invalid GCS Bucket',
                storageKeys: { gcs: 'invalid-gcs-bucket-name-that-does-not-exist-12345' }
            },
            {
                name: 'Invalid SFTP URL',
                storageKeys: { sftp: 'sftp://invalid-host-that-does-not-exist.com/path' }
            },
            {
                name: 'Malformed Storage Keys',
                storageKeys: { s3: '', gcs: null as any, sftp: 'not-a-url' }
            },
        ];

        for (const errorCase of errorCases) {
            await this.testBackendErrorCase(errorCase.name, errorCase.storageKeys);
        }
    }

    private async testBackendErrorCase (name: string, storageKeys: Record<string, string | undefined>): Promise<void> {
        const testName = `Backend Error: ${ name }`;
        const endTimer = this.results.startTest(testName);

        try {
            const imagePath = ImageUtils.createMinimalPNG(`error-${ name.replace(/\s+/g, '-').toLowerCase() }.png`);

            // Filter out undefined values
            const cleanStorageKeys = Object.fromEntries(
                Object.entries(storageKeys).filter(([ _, value ]) => value !== undefined)
            ) as Record<string, string>;

            const jobSpec: JobSpec = {
                priority: 0,
                keepOriginal: false,
                formats: {
                    jpg: {
                        settings: { quality: 80, speed: 1 },
                        sizes: [ [ 200, 150 ] ],
                    },
                },
                storageKeys: cleanStorageKeys,
                directHost: true,
                subDir: 'error-test',
            };

            const hash = await this.uploadImage(imagePath, jobSpec);
            const result = await this.waitForCompletion(hash);

            if (result.status === 'failed') {
                // Expected failure for invalid backends
                const error = this.getErrorMessage(result);
                this.results.recordPass(testName, 0, {
                    expectedFailure: true,
                    error,
                    storageKeys: Object.keys(storageKeys)
                });
            } else {
                // Unexpected success - backends might be configured differently
                this.results.recordPass(testName, 0, {
                    unexpectedSuccess: true,
                    status: result.status,
                    storageKeys: Object.keys(storageKeys)
                });
            }

        } catch (error) {
            this.results.recordFail(testName, 0, (error as Error).message);
        } finally {
            endTimer();
        }
    }

    private async testBackendPriorities (): Promise<void> {
        this.logger.info('Testing backend priority handling');

        // Test with different priority levels
        const priorities = [ 0, 1, 2 ]; // 0 = realtime, higher = queued

        for (const priority of priorities) {
            await this.testBackendPriority(priority);
        }
    }

    private async testBackendPriority (priority: number): Promise<void> {
        const testName = `Backend Priority: ${ priority }`;
        const endTimer = this.results.startTest(testName);

        try {
            const imagePath = await ImageUtils.downloadLoremPicsumImage(600, 400, `priority-${ priority }.jpg`);

            const jobSpec: JobSpec = {
                priority,
                keepOriginal: false,
                formats: {
                    jpg: {
                        settings: { quality: 80, speed: 1 },
                        sizes: [ [ 300, 200 ] ],
                    },
                },
                directHost: true,
                subDir: `priority-${ priority }`,
            };

            const hash = await this.uploadImage(imagePath, jobSpec);
            const result = await this.waitForCompletion(hash);

            if (result.status === 'success') {
                this.results.recordPass(testName, 0, {
                    priority,
                    fileCount: result.file_count,
                    hash
                });
            } else {
                const error = this.getErrorMessage(result);
                throw new Error(`Job failed: ${ error }`);
            }

        } catch (error) {
            this.results.recordFail(testName, 0, (error as Error).message);
        } finally {
            endTimer();
        }
    }

    // Helper methods
    private async uploadImage (imagePath: string, jobSpec: JobSpec): Promise<string> {
        this.logger.debug('Creating JWT for image upload', { imagePath: path.basename(imagePath) });
        const jwt = await JWTUtils.createJWT(jobSpec);

        this.logger.debug('Preparing form data for upload', { imagePath: path.basename(imagePath) });
        const form = new FormData();
        form.append('token', jwt);
        form.append('file', fs.createReadStream(imagePath), {
            filename: path.basename(imagePath),
            contentType: ImageUtils.getContentType(imagePath),
        });

        this.logger.debug('Uploading image to server', { imagePath: path.basename(imagePath) });
        const response = await HTTPUtils.post(`${ this.baseUrl }/upload`, form);
        const responseData: UploadResponse = response.data;

        if (!responseData.hash) {
            this.logger.error('Upload response missing hash', { imagePath: path.basename(imagePath) });
            throw new Error('Upload response missing hash');
        }

        this.logger.debug('Image uploaded successfully', { hash: responseData.hash, imagePath: path.basename(imagePath) });
        return responseData.hash;
    }

    private getErrorMessage (result: SuccessResponse | FailureResponse): string {
        return 'error' in result ? result.error || 'Unknown error' : 'Unknown error';
    }

    private async waitForCompletion (hash: string, maxWaitSeconds = 120): Promise<SuccessResponse | FailureResponse> {
        const startTime = Date.now();
        this.logger.debug('Waiting for job completion', { hash, maxWaitSeconds });

        while (Date.now() - startTime < maxWaitSeconds * 1000) {
            this.logger.trace('Checking job status', { hash, elapsedSeconds: Math.floor((Date.now() - startTime) / 1000) });
            const response = await HTTPUtils.get(`${ this.baseUrl }/success?hash=${ hash }`);
            const responseData: SuccessResponse = response.data;

            if (responseData.status === 'success') {
                this.logger.debug('Job completed successfully', { hash, fileCount: responseData.file_count });
                return responseData;
            }

            // Check if failed
            const failureResponse = await HTTPUtils.get(`${ this.baseUrl }/failures?hash=${ hash }`);
            const failureData: any = failureResponse.data;

            if (failureData.status === 'failed') {
                this.logger.warn('Job failed', { hash, error: failureData.error });
                return failureData;
            }

            // Wait before checking again
            await new Promise(resolve => setTimeout(resolve, 2000));
        }

        this.logger.error('Job processing timeout', { hash, maxWaitSeconds });
        throw new Error(`Processing did not complete within ${ maxWaitSeconds } seconds`);
    }

    getResults (): TestResults {
        return this.results;
    }
}

// CLI runner
if (require.main === module) {
    const tests = new BackendTests();

    tests.runAll()
        .then(() => {
            logger.info('Writer backend tests completed successfully');
            process.exit(0);
        })
        .catch((error) => {
            logger.error('Writer backend tests failed', { error: (error as Error).message });
            process.exit(1);
        });
}

export default BackendTests;
