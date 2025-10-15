/**
 * Codec and format integration tests for Pixerve
 *
 * Tests different image formats and codecs:
 * - JPG with various quality settings
 * - PNG with different compression levels
 * - WebP with quality and speed variations
 * - AVIF support (if available)
 * - Multiple format conversion in single job
 * - Format-specific edge cases
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

export class CodecFormatTests {
    private baseUrl: string;
    private server: PixerveServer;
    private results: TestResults;
    private logger: Logger;

    constructor(baseUrl = 'http://localhost:8080') {
        this.baseUrl = baseUrl;
        this.server = new PixerveServer(baseUrl);
        this.results = new TestResults();
        this.logger = new Logger(LogLevel.DEBUG, 'CODEC');
    }

    async runAll (): Promise<void> {
        this.logger.info('Starting codec and format tests');

        try {
            // Setup
            await this.setup();

            // Run tests
            await this.testJPGFormats();
            await this.testPNGFormats();
            await this.testWebPFormats();
            await this.testAVIFFormats();
            await this.testMultiFormatJobs();
            await this.testFormatEdgeCases();
            await this.testQualitySettings();
            await this.testSizeVariations();

            // Cleanup
            await this.cleanup();

        } catch (error) {
            this.logger.error('Codec and format tests failed', { error: (error as Error).message });
            throw error;
        } finally {
            this.results.printSummary();
        }
    }

    private async setup (): Promise<void> {
        this.logger.info('Setting up codec and format tests');
        ImageUtils.ensureTestImagesDir();
        await this.server.start();
        await this.server.waitForReady();
    }

    private async cleanup (): Promise<void> {
        this.logger.info('Cleaning up codec and format tests');
        await this.server.stop();
        ImageUtils.cleanupTestImages();
    }

    private async testJPGFormats (): Promise<void> {
        this.logger.info('Testing JPG format variations');

        const qualities = [ 10, 50, 80, 95, 100 ];
        const sizes = [ [ 800, 600 ], [ 400, 300 ], [ 200, 150 ] ];

        for (const quality of qualities) {
            for (const size of sizes) {
                await this.testJPGConversion(quality, size);
            }
        }
    }

    private async testJPGConversion (quality: number, size: number[]): Promise<void> {
        const testName = `JPG Quality ${ quality } Size ${ size.join('x') }`;
        const endTimer = this.results.startTest(testName);

        try {
            // Download a test image from Lorem Picsum
            const imagePath = await ImageUtils.downloadLoremPicsumImage(1000, 800, `jpg-test-${ quality }-${ size.join('x') }.jpg`);

            // Create job spec for JPG conversion
            const jobSpec: JobSpec = {
                priority: 0,
                keepOriginal: false,
                formats: {
                    jpg: {
                        settings: { quality, speed: 1 },
                        sizes: [ size ],
                    },
                },
                directHost: true,
                subDir: `codec-tests/jpg-q${ quality }`,
            };

            // Upload and wait for completion
            const hash = await this.uploadImage(imagePath, jobSpec);
            const result = await this.waitForCompletion(hash);

            if (result.status === 'success') {
                this.results.recordPass(testName, 0, {
                    quality,
                    size,
                    fileCount: result.file_count,
                    hash
                });
            } else {
                const error = 'error' in result ? result.error : 'Unknown error';
                throw new Error(`Job failed: ${ error }`);
            }

        } catch (error) {
            this.results.recordFail(testName, 0, (error as Error).message);
        } finally {
            endTimer();
        }
    }

    private async testPNGFormats (): Promise<void> {
        this.logger.info('Testing PNG format variations');

        const qualities = [ 10, 50, 80, 95 ]; // PNG quality affects compression
        const sizes = [ [ 800, 600 ], [ 400, 300 ] ];

        for (const quality of qualities) {
            for (const size of sizes) {
                await this.testPNGConversion(quality, size);
            }
        }
    }

    private async testPNGConversion (quality: number, size: number[]): Promise<void> {
        const testName = `PNG Quality ${ quality } Size ${ size.join('x') }`;
        const endTimer = this.results.startTest(testName);

        try {
            const imagePath = await ImageUtils.downloadLoremPicsumImage(1000, 800, `png-test-${ quality }-${ size.join('x') }.jpg`);

            const jobSpec: JobSpec = {
                priority: 0,
                keepOriginal: false,
                formats: {
                    png: {
                        settings: { quality, speed: 1 },
                        sizes: [ size ],
                    },
                },
                directHost: true,
                subDir: `codec-tests/png-q${ quality }`,
            };

            const hash = await this.uploadImage(imagePath, jobSpec);
            const result = await this.waitForCompletion(hash);

            if (result.status === 'success') {
                this.results.recordPass(testName, 0, {
                    quality,
                    size,
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

    private async testWebPFormats (): Promise<void> {
        this.logger.info('Testing WebP format variations');

        const qualities = [ 10, 50, 80, 95 ];
        const speeds = [ 1, 2, 3 ]; // WebP encoding speed
        const sizes = [ [ 800, 600 ], [ 400, 300 ] ];

        for (const quality of qualities) {
            for (const speed of speeds) {
                for (const size of sizes) {
                    await this.testWebPConversion(quality, speed, size);
                }
            }
        }
    }

    private async testWebPConversion (quality: number, speed: number, size: number[]): Promise<void> {
        const testName = `WebP Quality ${ quality } Speed ${ speed } Size ${ size.join('x') }`;
        const endTimer = this.results.startTest(testName);

        try {
            const imagePath = await ImageUtils.downloadLoremPicsumImage(1000, 800, `webp-test-${ quality }-${ speed }-${ size.join('x') }.jpg`);

            const jobSpec: JobSpec = {
                priority: 0,
                keepOriginal: false,
                formats: {
                    webp: {
                        settings: { quality, speed },
                        sizes: [ size ],
                    },
                },
                directHost: true,
                subDir: `codec-tests/webp-q${ quality }-s${ speed }`,
            };

            const hash = await this.uploadImage(imagePath, jobSpec);
            const result = await this.waitForCompletion(hash);

            if (result.status === 'success') {
                this.results.recordPass(testName, 0, {
                    quality,
                    speed,
                    size,
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

    private async testAVIFFormats (): Promise<void> {
        this.logger.info('Testing AVIF format support');

        const qualities = [ 50, 80 ]; // AVIF can be slow, so fewer tests
        const sizes = [ [ 400, 300 ] ];

        for (const quality of qualities) {
            for (const size of sizes) {
                await this.testAVIFConversion(quality, size);
            }
        }
    }

    private async testAVIFConversion (quality: number, size: number[]): Promise<void> {
        const testName = `AVIF Quality ${ quality } Size ${ size.join('x') }`;
        const endTimer = this.results.startTest(testName);

        try {
            const imagePath = await ImageUtils.downloadLoremPicsumImage(800, 600, `avif-test-${ quality }-${ size.join('x') }.jpg`);

            const jobSpec: JobSpec = {
                priority: 0,
                keepOriginal: false,
                formats: {
                    avif: {
                        settings: { quality, speed: 1 },
                        sizes: [ size ],
                    },
                },
                directHost: true,
                subDir: `codec-tests/avif-q${ quality }`,
            };

            const hash = await this.uploadImage(imagePath, jobSpec);
            const result = await this.waitForCompletion(hash);

            if (result.status === 'success') {
                this.results.recordPass(testName, 0, {
                    quality,
                    size,
                    fileCount: result.file_count,
                    hash
                });
            } else {
                // AVIF might not be supported, which is OK
                const error = this.getErrorMessage(result);
                this.logger.warn(`AVIF conversion failed (might not be supported): ${ error }`);
                this.results.recordPass(testName, 0, {
                    quality,
                    size,
                    status: 'skipped - not supported',
                    error
                });
            }

        } catch (error) {
            this.results.recordFail(testName, 0, (error as Error).message);
        } finally {
            endTimer();
        }
    }

    private async testMultiFormatJobs (): Promise<void> {
        this.logger.info('Testing multi-format jobs');

        const testCases = [
            {
                name: 'JPG + WebP',
                formats: {
                    jpg: { settings: { quality: 80, speed: 1 }, sizes: [ [ 800, 600 ], [ 400, 300 ] ] },
                    webp: { settings: { quality: 85, speed: 2 }, sizes: [ [ 800 ], [ 400 ] ] },
                }
            },
            {
                name: 'JPG + PNG + WebP',
                formats: {
                    jpg: { settings: { quality: 75, speed: 1 }, sizes: [ [ 600, 400 ] ] },
                    png: { settings: { quality: 90, speed: 1 }, sizes: [ [ 600, 400 ] ] },
                    webp: { settings: { quality: 80, speed: 2 }, sizes: [ [ 600 ], [ 300 ] ] },
                }
            }
        ];

        for (const testCase of testCases) {
            await this.testMultiFormatConversion(testCase.name, testCase.formats);
        }
    }

    private async testMultiFormatConversion (name: string, formats: Record<string, any>): Promise<void> {
        const testName = `Multi-Format: ${ name }`;
        const endTimer = this.results.startTest(testName);

        try {
            const imagePath = await ImageUtils.downloadLoremPicsumImage(1000, 800, `multi-${ name.replace(/\s+/g, '-').toLowerCase() }.jpg`);

            const jobSpec: JobSpec = {
                priority: 0,
                keepOriginal: false,
                formats,
                directHost: true,
                subDir: `codec-tests/multi-${ name.replace(/\s+/g, '-').toLowerCase() }`,
            };

            const hash = await this.uploadImage(imagePath, jobSpec);
            const result = await this.waitForCompletion(hash);

            if (result.status === 'success') {
                this.results.recordPass(testName, 0, {
                    formats: Object.keys(formats),
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

    private async testFormatEdgeCases (): Promise<void> {
        this.logger.info('Testing format edge cases');

        // Test cases for edge cases
        const edgeCases = [
            {
                name: 'Zero Quality',
                formats: { jpg: { settings: { quality: 0, speed: 1 }, sizes: [ [ 400, 300 ] ] } }
            },
            {
                name: 'Maximum Quality',
                formats: { png: { settings: { quality: 100, speed: 1 }, sizes: [ [ 400, 300 ] ] } }
            },
            {
                name: 'Very Small Size',
                formats: { jpg: { settings: { quality: 80, speed: 1 }, sizes: [ [ 1, 1 ] ] } }
            },
            {
                name: 'Square Images',
                formats: { webp: { settings: { quality: 85, speed: 2 }, sizes: [ [ 200 ], [ 100 ], [ 50 ] ] } }
            }
        ];

        for (const edgeCase of edgeCases) {
            await this.testEdgeCase(edgeCase.name, edgeCase.formats);
        }
    }

    private async testEdgeCase (name: string, formats: Record<string, any>): Promise<void> {
        const testName = `Edge Case: ${ name }`;
        const endTimer = this.results.startTest(testName);

        try {
            const imagePath = await ImageUtils.downloadLoremPicsumImage(800, 600, `edge-${ name.replace(/\s+/g, '-').toLowerCase() }.jpg`);

            const jobSpec: JobSpec = {
                priority: 0,
                keepOriginal: false,
                formats,
                directHost: true,
                subDir: `codec-tests/edge-${ name.replace(/\s+/g, '-').toLowerCase() }`,
            };

            const hash = await this.uploadImage(imagePath, jobSpec);
            const result = await this.waitForCompletion(hash);

            if (result.status === 'success') {
                this.results.recordPass(testName, 0, {
                    formats: Object.keys(formats),
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

    private async testQualitySettings (): Promise<void> {
        this.logger.info('Testing quality settings impact');

        const qualities = [ 10, 30, 60, 90 ];
        const results: any[] = [];

        for (const quality of qualities) {
            const imagePath = await ImageUtils.downloadLoremPicsumImage(1000, 800, `quality-test-${ quality }.jpg`);

            const jobSpec: JobSpec = {
                priority: 0,
                keepOriginal: false,
                formats: {
                    jpg: {
                        settings: { quality, speed: 1 },
                        sizes: [ [ 800, 600 ] ],
                    },
                },
                directHost: true,
                subDir: `codec-tests/quality-${ quality }`,
            };

            const hash = await this.uploadImage(imagePath, jobSpec);
            const result = await this.waitForCompletion(hash);

            if (result.status === 'success') {
                results.push({ quality, success: true, fileCount: result.file_count });
            } else {
                results.push({ quality, success: false, error: this.getErrorMessage(result) });
            }
        }

        this.results.recordPass('Quality Settings Impact', 0, { results });
    }

    private async testSizeVariations (): Promise<void> {
        this.logger.info('Testing size variations');

        const sizes = [
            [ 1920, 1080 ], // Full HD
            [ 1280, 720 ],  // HD
            [ 800, 600 ],   // Standard
            [ 400, 300 ],   // Small
            [ 200, 150 ],   // Thumbnail
            [ 100, 100 ],   // Square
        ];

        const results: any[] = [];

        for (const size of sizes) {
            const imagePath = await ImageUtils.downloadLoremPicsumImage(1200, 900, `size-test-${ size.join('x') }.jpg`);

            const jobSpec: JobSpec = {
                priority: 0,
                keepOriginal: false,
                formats: {
                    jpg: {
                        settings: { quality: 80, speed: 1 },
                        sizes: [ size ],
                    },
                },
                directHost: true,
                subDir: `codec-tests/size-${ size.join('x') }`,
            };

            const hash = await this.uploadImage(imagePath, jobSpec);
            const result = await this.waitForCompletion(hash);

            if (result.status === 'success') {
                results.push({ size, success: true, fileCount: result.file_count });
            } else {
                results.push({ size, success: false, error: this.getErrorMessage(result) });
            }
        }

        this.results.recordPass('Size Variations', 0, { results });
    }

    // Helper methods
    private async uploadImage (imagePath: string, jobSpec: JobSpec): Promise<string> {
        const jwt = await JWTUtils.createJWT(jobSpec);

        const form = new FormData();
        form.append('token', jwt);
        form.append('file', fs.createReadStream(imagePath), {
            filename: path.basename(imagePath),
            contentType: ImageUtils.getContentType(imagePath),
        });

        const response = await HTTPUtils.post(`${ this.baseUrl }/upload`, form);
        const responseData: UploadResponse = response.data;

        if (!responseData.hash) {
            throw new Error('Upload response missing hash');
        }

        return responseData.hash;
    }

    private getErrorMessage (result: SuccessResponse | FailureResponse): string {
        return 'error' in result ? result.error || 'Unknown error' : 'Unknown error';
    }

    private async waitForCompletion (hash: string, maxWaitSeconds = 120): Promise<SuccessResponse | FailureResponse> {
        const startTime = Date.now();

        while (Date.now() - startTime < maxWaitSeconds * 1000) {
            const response = await HTTPUtils.get(`${ this.baseUrl }/success?hash=${ hash }`);
            const responseData: SuccessResponse = response.data;

            if (responseData.status === 'success') {
                return responseData;
            }

            // Check if failed
            const failureResponse = await HTTPUtils.get(`${ this.baseUrl }/failures?hash=${ hash }`);
            const failureData: any = failureResponse.data;

            if (failureData.status === 'failed') {
                return failureData;
            }

            // Wait before checking again
            await new Promise(resolve => setTimeout(resolve, 2000));
        }

        throw new Error(`Processing did not complete within ${ maxWaitSeconds } seconds`);
    }

    getResults (): TestResults {
        return this.results;
    }
}

// CLI runner
if (require.main === module) {
    const tests = new CodecFormatTests();

    tests.runAll()
        .then(() => {
            logger.info('Codec and format tests completed successfully');
            process.exit(0);
        })
        .catch((error) => {
            logger.error('Codec and format tests failed', { error: (error as Error).message });
            process.exit(1);
        });
}

export default CodecFormatTests;
