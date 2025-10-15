/**
 * Common utilities for Pixerve integration tests
 *
 * This module provides shared functionality used across all test suites:
 * - JWT creation and validation
 * - HTTP client utilities
 * - Image downloading from Lorem Picsum
 * - Logging utilities
 * - Test server management
 * - File system utilities
 */

import * as fs from 'fs';
import * as path from 'path';
import * as http from 'http';
import { spawn, ChildProcess } from 'child_process';
import axios, { AxiosResponse } from 'axios';
import FormData from 'form-data';
import * as jose from 'jose';

// For CommonJS __dirname equivalent
const __dirname = path.dirname(require.main!.filename);

// Logging levels
export enum LogLevel {
    ERROR = 0,
    WARN = 1,
    INFO = 2,
    DEBUG = 3,
    TRACE = 4
}

// Types for Pixerve API
export interface JobSpec {
    completionCallback?: string;
    callbackHeaders?: Record<string, string>;
    priority: number;
    keepOriginal: boolean;
    formats: Record<string, FormatSpec>;
    storageKeys?: Record<string, string>;
    directHost?: boolean;
    subDir?: string;
}

export interface FormatSpec {
    settings: FormatSettings;
    sizes: number[][];
}

export interface FormatSettings {
    quality: number;
    speed: number;
}

export interface PixerveJWT {
    iss?: string;
    sub: string;
    iat: number;
    exp: number;
    job: JobSpec;
}

export interface SuccessResponse {
    hash: string;
    status: 'success' | 'not_found';
    timestamp?: string;
    file_count?: number;
    job_data?: any;
    message?: string;
}

export interface FailureResponse {
    hash: string;
    status: 'failed' | 'not_found';
    timestamp?: string;
    error?: string;
    job_data?: any;
    message?: string;
}

export interface UploadResponse {
    hash: string;
    message: string;
}

export interface StatusResponse {
    hash: string;
    status: 'success' | 'failed' | 'not_found';
    timestamp?: string;
    file_count?: number;
    error?: string;
    message?: string;
}

// Logger class for verbose logging
export class Logger {
    private level: LogLevel;
    private prefix: string;

    constructor(level: LogLevel = LogLevel.INFO, prefix = '') {
        this.level = level;
        this.prefix = prefix;
    }

    private shouldLog (level: LogLevel): boolean {
        return level <= this.level;
    }

    private formatMessage (level: string, message: string, data?: any): string {
        const timestamp = new Date().toISOString();
        const prefix = this.prefix ? `[${ this.prefix }] ` : '';
        const dataStr = data ? ` ${ JSON.stringify(data, null, 2) }` : '';
        return `${ timestamp } ${ level } ${ prefix }${ message }${ dataStr }`;
    }

    error (message: string, data?: any): void {
        if (this.shouldLog(LogLevel.ERROR)) {
            console.error(this.formatMessage('ERROR', message, data));
        }
    }

    warn (message: string, data?: any): void {
        if (this.shouldLog(LogLevel.WARN)) {
            console.warn(this.formatMessage('WARN', message, data));
        }
    }

    info (message: string, data?: any): void {
        if (this.shouldLog(LogLevel.INFO)) {
            console.log(this.formatMessage('INFO', message, data));
        }
    }

    debug (message: string, data?: any): void {
        if (this.shouldLog(LogLevel.DEBUG)) {
            console.log(this.formatMessage('DEBUG', message, data));
        }
    }

    trace (message: string, data?: any): void {
        if (this.shouldLog(LogLevel.TRACE)) {
            console.log(this.formatMessage('TRACE', message, data));
        }
    }
}

// Global logger instance
export const logger = new Logger(LogLevel.DEBUG, 'COMMON');

// JWT utilities
export class JWTUtils {
    private static readonly DEFAULT_SECRET = 'test-secret-key-for-jwt-signing-at-least-32-bytes-long';

    static async createJWT (jobSpec: JobSpec, subject = 'integration-test', expiresInSeconds = 3600, secret?: string): Promise<string> {
        const jwtSecret = secret || JWTUtils.DEFAULT_SECRET;
        const secretBytes = new TextEncoder().encode(jwtSecret);

        logger.debug('Creating JWT', { subject, expiresInSeconds, jobSpecKeys: Object.keys(jobSpec) });

        const jwt = await new jose.SignJWT({
            sub: subject,
            job: jobSpec,
            iat: Math.floor(Date.now() / 1000),
            exp: Math.floor(Date.now() / 1000) + expiresInSeconds,
        })
            .setProtectedHeader({ alg: 'HS256', typ: 'JWT' })
            .sign(secretBytes);

        logger.trace('JWT created successfully');
        return jwt;
    }

    static async verifyJWT (token: string, secret?: string): Promise<PixerveJWT> {
        const jwtSecret = secret || JWTUtils.DEFAULT_SECRET;
        const secretBytes = new TextEncoder().encode(jwtSecret);

        logger.debug('Verifying JWT');

        const { payload } = await jose.jwtVerify(token, secretBytes);
        const pixerveJWT = payload as unknown as PixerveJWT;

        logger.trace('JWT verified successfully', { sub: pixerveJWT.sub });
        return pixerveJWT;
    }
}

// HTTP client utilities
export class HTTPUtils {
    private static readonly DEFAULT_TIMEOUT = 30000;

    static async post (url: string, data: FormData, timeout = HTTPUtils.DEFAULT_TIMEOUT): Promise<AxiosResponse> {
        logger.debug('Making POST request', { url, timeout });

        const response = await axios.post(url, data, {
            headers: data.getHeaders(),
            timeout,
        });

        logger.trace('POST request completed', { status: response.status });
        return response;
    }

    static async get (url: string, timeout = HTTPUtils.DEFAULT_TIMEOUT): Promise<AxiosResponse> {
        logger.debug('Making GET request', { url, timeout });

        const response = await axios.get(url, { timeout });

        logger.trace('GET request completed', { status: response.status });
        return response;
    }

    static async waitForEndpoint (url: string, maxWaitSeconds = 30): Promise<void> {
        logger.info(`Waiting for endpoint to be ready: ${ url }`);

        for (let i = 0; i < maxWaitSeconds; i++) {
            try {
                await HTTPUtils.get(url, 1000);
                logger.info('Endpoint is ready');
                return;
            } catch (error) {
                logger.debug(`Endpoint not ready, retrying... (${ i + 1 }/${ maxWaitSeconds })`);
                await new Promise(resolve => setTimeout(resolve, 1000));
            }
        }

        throw new Error(`Endpoint ${ url } did not become ready within ${ maxWaitSeconds } seconds`);
    }
}

// Image utilities for downloading and managing test images
export class ImageUtils {
    private static readonly TEST_IMAGES_DIR = path.join(__dirname, '..', 'test-images');

    static getTestImagesDir (): string {
        return ImageUtils.TEST_IMAGES_DIR;
    }

    static ensureTestImagesDir (): void {
        if (!fs.existsSync(ImageUtils.TEST_IMAGES_DIR)) {
            fs.mkdirSync(ImageUtils.TEST_IMAGES_DIR, { recursive: true });
            logger.info('Created test images directory', { path: ImageUtils.TEST_IMAGES_DIR });
        }
    }

    static async downloadLoremPicsumImage (width: number, height: number, filename?: string): Promise<string> {
        ImageUtils.ensureTestImagesDir();

        const imageName = filename || `lorem-picsum-${ width }x${ height }-${ Date.now() }.jpg`;
        const imagePath = path.join(ImageUtils.TEST_IMAGES_DIR, imageName);

        const url = `https://picsum.photos/${ width }/${ height }`;
        logger.info('Downloading Lorem Picsum image', { url, width, height, savePath: imagePath });

        try {
            const response = await axios.get(url, {
                responseType: 'stream',
                timeout: 10000,
            });

            const writer = fs.createWriteStream(imagePath);
            response.data.pipe(writer);

            await new Promise<void>((resolve, reject) => {
                writer.on('finish', () => resolve());
                writer.on('error', reject);
            });

            logger.info('Image downloaded successfully', { size: fs.statSync(imagePath).size });
            return imagePath;
        } catch (error) {
            const err = error as Error;
            logger.error('Failed to download image', { url, error: err.message });
            throw error;
        }
    }

    static createMinimalPNG (filename = 'minimal.png'): string {
        ImageUtils.ensureTestImagesDir();

        const imagePath = path.join(ImageUtils.TEST_IMAGES_DIR, filename);

        // Create a minimal 1x1 PNG image (base64 encoded)
        const pngData = Buffer.from(
            'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==',
            'base64'
        );

        fs.writeFileSync(imagePath, pngData);
        logger.debug('Created minimal PNG image', { path: imagePath, size: pngData.length });

        return imagePath;
    }

    static createLargeTestImage (sizeMB: number, filename?: string): string {
        ImageUtils.ensureTestImagesDir();

        const imageName = filename || `large-test-${ sizeMB }mb-${ Date.now() }.png`;
        const imagePath = path.join(ImageUtils.TEST_IMAGES_DIR, imageName);

        // Create a large image by repeating the minimal PNG data
        const minimalPNG = Buffer.from(
            'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==',
            'base64'
        );

        const targetSize = sizeMB * 1024 * 1024;
        const repeats = Math.ceil(targetSize / minimalPNG.length);
        const largeData = Buffer.alloc(repeats * minimalPNG.length);

        for (let i = 0; i < repeats; i++) {
            minimalPNG.copy(largeData, i * minimalPNG.length);
        }

        fs.writeFileSync(imagePath, largeData);
        logger.info('Created large test image', { path: imagePath, size: largeData.length });

        return imagePath;
    }

    static getContentType (filePath: string): string {
        const ext = path.extname(filePath).toLowerCase();
        const types: Record<string, string> = {
            '.jpg': 'image/jpeg',
            '.jpeg': 'image/jpeg',
            '.png': 'image/png',
            '.webp': 'image/webp',
            '.avif': 'image/avif',
            '.gif': 'image/gif',
            '.bmp': 'image/bmp',
            '.tiff': 'image/tiff',
            '.svg': 'image/svg+xml',
        };

        return types[ ext ] || 'application/octet-stream';
    }

    static cleanupTestImages (): void {
        if (fs.existsSync(ImageUtils.TEST_IMAGES_DIR)) {
            const files = fs.readdirSync(ImageUtils.TEST_IMAGES_DIR);
            for (const file of files) {
                const filePath = path.join(ImageUtils.TEST_IMAGES_DIR, file);
                try {
                    fs.unlinkSync(filePath);
                    logger.debug('Cleaned up test image', { file: filePath });
                } catch (error) {
                    const err = error as Error;
                    logger.warn('Failed to cleanup test image', { file: filePath, error: err.message });
                }
            }
        }
    }
}

// Pixerve server management
export class PixerveServer {
    private process: ChildProcess | null = null;
    private baseUrl: string;
    private logger: Logger;

    constructor(baseUrl = 'http://localhost:8080') {
        this.baseUrl = baseUrl;
        this.logger = new Logger(LogLevel.INFO, 'SERVER');
    }

    async start (): Promise<void> {
        this.logger.info('Starting Pixerve server...');

        return new Promise((resolve, reject) => {
            // Build Pixerve first
            const buildProcess = spawn('go', [ 'build', '.' ], {
                cwd: path.join(__dirname, '..'),
                stdio: [ 'ignore', 'pipe', 'pipe' ]
            });

            buildProcess.on('close', (code) => {
                if (code !== 0) {
                    reject(new Error(`Go build failed with code ${ code }`));
                    return;
                }

                // Start the server
                this.process = spawn('./pixerve', [], {
                    cwd: path.join(__dirname, '..'),
                    stdio: [ 'ignore', 'pipe', 'pipe' ]
                });

                if (this.process.stdout) {
                    this.process.stdout.on('data', (data) => {
                        const output = data.toString();
                        this.logger.debug('Server stdout:', output.trim());
                        if (output.includes('Server started on :8080')) {
                            this.logger.info('Pixerve server started successfully');
                            resolve();
                        }
                    });
                }

                if (this.process.stderr) {
                    this.process.stderr.on('data', (data) => {
                        this.logger.debug('Server stderr:', data.toString().trim());
                    });
                }

                this.process.on('error', (error) => {
                    this.logger.error('Server process error', { error: error.message });
                    reject(error);
                });

                // Timeout after 15 seconds
                setTimeout(() => {
                    reject(new Error('Timeout waiting for Pixerve to start'));
                }, 15000);
            });

            buildProcess.on('error', reject);
        });
    }

    async waitForReady (): Promise<void> {
        await HTTPUtils.waitForEndpoint(`${ this.baseUrl }/health`);
    }

    async stop (): Promise<void> {
        if (this.process) {
            this.logger.info('Stopping Pixerve server...');
            this.process.kill('SIGTERM');

            await new Promise((resolve) => {
                this.process!.on('close', () => {
                    this.logger.info('Pixerve server stopped');
                    resolve(void 0);
                });
            });
        }
    }

    getBaseUrl (): string {
        return this.baseUrl;
    }
}

// Test result tracking
export class TestResults {
    private results: Array<{
        name: string;
        passed: boolean;
        duration: number;
        error?: string;
        data?: any;
    }> = [];

    private logger: Logger;

    constructor() {
        this.logger = new Logger(LogLevel.INFO, 'RESULTS');
    }

    startTest (name: string): () => void {
        const startTime = Date.now();
        this.logger.info(`Starting test: ${ name }`);

        return () => {
            const duration = Date.now() - startTime;
            this.logger.info(`Completed test: ${ name } (${ duration }ms)`);
        };
    }

    recordPass (name: string, duration: number, data?: any): void {
        this.results.push({ name, passed: true, duration, data });
        this.logger.info(`✅ Test passed: ${ name }`, { duration, data });
    }

    recordFail (name: string, duration: number, error: string, data?: any): void {
        this.results.push({ name, passed: false, duration, error, data });
        this.logger.error(`❌ Test failed: ${ name }`, { duration, error, data });
    }

    getSummary (): { total: number; passed: number; failed: number; duration: number; } {
        const total = this.results.length;
        const passed = this.results.filter(r => r.passed).length;
        const failed = total - passed;
        const duration = this.results.reduce((sum, r) => sum + r.duration, 0);

        return { total, passed, failed, duration };
    }

    printSummary (): void {
        const summary = this.getSummary();

        this.logger.info('Test Summary:', summary);

        if (summary.failed > 0) {
            this.logger.error('Failed tests:');
            this.results.filter(r => !r.passed).forEach(r => {
                this.logger.error(`  - ${ r.name }: ${ r.error }`);
            });
        }
    }

    getResults (): typeof this.results {
        return [ ...this.results ];
    }
}

// Utility functions for creating common job specs
export class JobSpecUtils {
    static createBasicJobSpec (formats: Record<string, FormatSpec> = {}): JobSpec {
        return {
            priority: 0,
            keepOriginal: false,
            formats: formats,
            directHost: true,
            subDir: 'integration-test',
        };
    }

    static createJPGJobSpec (quality = 80, sizes: number[][] = [ [ 800, 600 ], [ 400, 300 ] ]): JobSpec {
        return JobSpecUtils.createBasicJobSpec({
            jpg: {
                settings: { quality, speed: 1 },
                sizes,
            },
        });
    }

    static createWebPJobSpec (quality = 85, sizes: number[][] = [ [ 800 ], [ 400 ] ]): JobSpec {
        return JobSpecUtils.createBasicJobSpec({
            webp: {
                settings: { quality, speed: 2 },
                sizes,
            },
        });
    }

    static createMultiFormatJobSpec (): JobSpec {
        return JobSpecUtils.createBasicJobSpec({
            jpg: {
                settings: { quality: 80, speed: 1 },
                sizes: [ [ 800, 600 ], [ 400, 300 ] ],
            },
            webp: {
                settings: { quality: 85, speed: 2 },
                sizes: [ [ 800 ], [ 400 ] ],
            },
            png: {
                settings: { quality: 90, speed: 1 },
                sizes: [ [ 600, 400 ] ],
            },
        });
    }

    static createCallbackJobSpec (callbackUrl: string): JobSpec {
        const jobSpec = JobSpecUtils.createJPGJobSpec();
        jobSpec.completionCallback = callbackUrl;
        jobSpec.callbackHeaders = {
            'X-Test': 'integration-test',
            'Authorization': 'Bearer test-token',
        };
        return jobSpec;
    }
}

// Export utility functions
export { ImageUtils as default };
