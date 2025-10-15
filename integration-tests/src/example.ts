#!/usr/bin/env node

/**
 * Pixerve TypeScript Usage Example
 *
 * This script demonstrates how to use Pixerve from a TypeScript/JavaScript application.
 * It shows the complete workflow: JWT creation, file upload, and status checking.
 */

import axios, { AxiosResponse } from 'axios';
import FormData from 'form-data';
import * as jose from 'jose';
import * as fs from 'fs';
import * as path from 'path';

// Types (same as in Pixerve)
interface JobSpec {
    completionCallback?: string;
    callbackHeaders?: Record<string, string>;
    priority: number;
    keepOriginal: boolean;
    formats: Record<string, FormatSpec>;
    storageKeys?: Record<string, string>;
    directHost?: boolean;
    subDir?: string;
}

interface FormatSpec {
    settings: FormatSettings;
    sizes: number[][];
}

interface FormatSettings {
    quality: number;
    speed: number;
}

interface UploadResponse {
    hash: string;
    message: string;
}

interface StatusResponse {
    hash: string;
    status: 'success' | 'failed' | 'not_found';
    timestamp?: string;
    file_count?: number;
    error?: string;
    message?: string;
}

class PixerveClient {
    private baseUrl: string;
    private jwtSecret: string;

    constructor(baseUrl = 'http://localhost:8080', jwtSecret = 'your-jwt-secret-here') {
        this.baseUrl = baseUrl;
        this.jwtSecret = jwtSecret;
    }

    /**
     * Create a signed JWT for image processing
     */
    async createJobToken (jobSpec: JobSpec, subject = 'image-job', expiresInSeconds = 3600): Promise<string> {
        const secret = new TextEncoder().encode(this.jwtSecret);

        const jwt = await new jose.SignJWT({
            sub: subject,
            job: jobSpec,
            iat: Math.floor(Date.now() / 1000),
            exp: Math.floor(Date.now() / 1000) + expiresInSeconds,
        })
            .setProtectedHeader({ alg: 'HS256', typ: 'JWT' })
            .sign(secret);

        return jwt;
    }

    /**
     * Upload an image for processing
     */
    async uploadImage (imagePath: string, jobSpec: JobSpec): Promise<UploadResponse> {
        // Create JWT
        const jwt = await this.createJobToken(jobSpec);

        // Create form data
        const form = new FormData();
        form.append('token', jwt);
        form.append('file', fs.createReadStream(imagePath), {
            filename: path.basename(imagePath),
            contentType: this.getContentType(imagePath),
        });

        // Upload
        const response: AxiosResponse<UploadResponse> = await axios.post(
            `${ this.baseUrl }/upload`,
            form,
            {
                headers: form.getHeaders(),
                timeout: 30000,
            }
        );

        return response.data;
    }

    /**
     * Check processing status
     */
    async getStatus (hash: string, type: 'success' | 'failure' = 'success'): Promise<StatusResponse> {
        const endpoint = type === 'success' ? 'success' : 'failures';
        const response: AxiosResponse<StatusResponse> = await axios.get(
            `${ this.baseUrl }/${ endpoint }?hash=${ hash }`
        );

        return response.data;
    }

    /**
     * Wait for processing to complete
     */
    async waitForCompletion (hash: string, maxWaitSeconds = 60): Promise<StatusResponse> {
        const startTime = Date.now();

        while (Date.now() - startTime < maxWaitSeconds * 1000) {
            // Check success first
            const successStatus = await this.getStatus(hash, 'success');
            if (successStatus.status === 'success') {
                return successStatus;
            }

            // Check failure
            const failureStatus = await this.getStatus(hash, 'failure');
            if (failureStatus.status === 'failed') {
                return failureStatus;
            }

            // Wait before checking again
            await new Promise(resolve => setTimeout(resolve, 2000));
        }

        throw new Error(`Processing did not complete within ${ maxWaitSeconds } seconds`);
    }

    private getContentType (filePath: string): string {
        const ext = path.extname(filePath).toLowerCase();
        const types: Record<string, string> = {
            '.jpg': 'image/jpeg',
            '.jpeg': 'image/jpeg',
            '.png': 'image/png',
            '.webp': 'image/webp',
            '.avif': 'image/avif',
            '.gif': 'image/gif',
        };

        return types[ ext ] || 'application/octet-stream';
    }
}

// Example usage
async function example () {
    const client = new PixerveClient('http://localhost:8080', 'your-jwt-secret-here');

    // Define the processing job
    const jobSpec: JobSpec = {
        completionCallback: 'https://your-app.com/webhook',
        callbackHeaders: {
            'Authorization': 'Bearer your-webhook-token',
            'X-Custom-Header': 'custom-value',
        },
        priority: 0, // 0 = realtime, 1 = queued
        keepOriginal: false,
        formats: {
            // Convert to multiple JPG sizes
            jpg: {
                settings: { quality: 80, speed: 1 },
                sizes: [
                    [ 1920, 1080 ], // Large
                    [ 800, 600 ],    // Medium
                    [ 400, 300 ],    // Small
                ],
            },
            // Convert to WebP
            webp: {
                settings: { quality: 85, speed: 2 },
                sizes: [
                    [ 800 ],  // Square
                    [ 400 ],  // Smaller square
                ],
            },
        },
        storageKeys: {
            s3: 'your-s3-storage-key',
            // Add other storage backends as needed
        },
        directHost: true,  // Also serve via Pixerve HTTP
        subDir: 'user-uploads/123',  // Organize files
    };

    try {
        console.log('📤 Uploading image...');
        const uploadResult = await client.uploadImage('./example-image.jpg', jobSpec);
        console.log('✅ Upload successful:', uploadResult);

        console.log('⏳ Waiting for processing...');
        const finalStatus = await client.waitForCompletion(uploadResult.hash);
        console.log('🎉 Processing complete:', finalStatus);

        if (finalStatus.status === 'success') {
            console.log(`📊 Generated ${ finalStatus.file_count } files`);
        } else {
            console.error('❌ Processing failed:', finalStatus.error);
        }

    } catch (error) {
        console.error('💥 Error:', error);
    }
}

// Export for use as a module
export default PixerveClient;

// Run example if called directly
if (require.main === module) {
    example();
}