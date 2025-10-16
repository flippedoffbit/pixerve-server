#!/usr/bin/env node

/**
 * Pixerve Integration Test Runner
 *
 * Runs all integration test suites:
 * - Basic functionality tests
 * - Codec and format tests
 * - Writer backend tests
 * - Edge case tests
 *
 * Usage:
 *   npm test                    # Run all tests
 *   npm run test:basic         # Run only basic tests
 *   npm run test:codec         # Run only codec tests
 *   npm run test:backend       # Run only backend tests
 *   npm run test:edge          # Run only edge case tests
 *   node dist/test-runner.js --help  # Show help
 */

import * as fs from 'fs';
import * as path from 'path';
import { Command } from 'commander';
import { Logger, LogLevel, logger } from './common';
import BasicTests from './basic-tests';
import CodecTests from './codec-tests';
import BackendTests from './backend-tests';
import EdgeCaseTests from './edge-case-tests';

interface TestSuite {
    name: string;
    description: string;
    run: () => Promise<void>;
    getResults: () => any;
}

class TestRunner {
    private logger: Logger;
    private program: Command;
    private baseUrl: string;
    private logLevel: LogLevel;

    constructor() {
        this.logger = new Logger(LogLevel.INFO, 'RUNNER');
        this.program = new Command();
        this.baseUrl = process.env.PIXERVE_URL || 'http://localhost:8080';
        this.logLevel = LogLevel.INFO;

        this.setupCLI();
    }

    private setupCLI (): void {
        this.program
            .name('pixerve-test-runner')
            .description('Run Pixerve integration tests')
            .version('1.0.0')
            .option('-u, --url <url>', 'Pixerve server URL', this.baseUrl)
            .option('-v, --verbose', 'Enable verbose logging (DEBUG level)')
            .option('-l, --log-level <level>', 'Set log level (ERROR, WARN, INFO, DEBUG, TRACE)', 'INFO')
            .option('--no-color', 'Disable colored output')
            .option('--json', 'Output results as JSON')
            .option('--junit <file>', 'Output results in JUnit XML format');

        // Set log level based on options
        this.program.on('option:verbose', () => {
            this.logLevel = LogLevel.DEBUG;
            this.logger = new Logger(this.logLevel, 'RUNNER');
        });

        this.program.on('option:log-level', (level) => {
            const levelMap: Record<string, LogLevel> = {
                'ERROR': LogLevel.ERROR,
                'WARN': LogLevel.WARN,
                'INFO': LogLevel.INFO,
                'DEBUG': LogLevel.DEBUG,
                'TRACE': LogLevel.TRACE
            };
            this.logLevel = levelMap[ level.toUpperCase() ] || LogLevel.INFO;
            this.logger = new Logger(this.logLevel, 'RUNNER');
        });

        this.program
            .command('all')
            .description('Run all test suites')
            .action(() => this.runAllTests());

        this.program
            .command('basic')
            .description('Run basic functionality tests')
            .action(() => this.runBasicTests());

        this.program
            .command('codec')
            .description('Run codec and format tests')
            .action(() => this.runCodecTests());

        this.program
            .command('backend')
            .description('Run writer backend tests')
            .action(() => this.runBackendTests());

        this.program
            .command('edge')
            .description('Run edge case tests')
            .action(() => this.runEdgeCaseTests());

        // Default command runs all tests
        this.program.action(() => this.runAllTests());
    }

    private getTestSuites (): TestSuite[] {
        const basicTests = new BasicTests(this.baseUrl, this.logLevel);
        const codecTests = new CodecTests(this.baseUrl, this.logLevel);
        const backendTests = new BackendTests(this.baseUrl, this.logLevel);
        const edgeCaseTests = new EdgeCaseTests(this.baseUrl, this.logLevel);

        return [
            {
                name: 'Basic Tests',
                description: 'Basic API functionality (health, upload, status, callbacks)',
                run: () => basicTests.runAll(),
                getResults: () => basicTests.getResults(),
            },
            {
                name: 'Codec Tests',
                description: 'Image codec and format testing (JPG, PNG, WebP, AVIF)',
                run: () => codecTests.runAll(),
                getResults: () => codecTests.getResults(),
            },
            {
                name: 'Backend Tests',
                description: 'Writer backend testing (S3, GCS, SFTP, direct hosting)',
                run: () => backendTests.runAll(),
                getResults: () => backendTests.getResults(),
            },
            {
                name: 'Edge Case Tests',
                description: 'Edge cases and error conditions (large files, invalid formats, network errors)',
                run: () => edgeCaseTests.runAll(),
                getResults: () => edgeCaseTests.getResults(),
            },
        ];
    }

    private async runAllTests (): Promise<void> {
        this.logger.info('Starting all Pixerve integration tests');
        this.logger.info(`Server URL: ${ this.baseUrl }`);
        this.logger.debug(`Log level: ${ LogLevel[ this.logLevel ] }`);

        const suites = this.getTestSuites();
        const results = [];

        for (const suite of suites) {
            this.logger.info(`Running ${ suite.name }...`);

            try {
                const startTime = Date.now();
                await suite.run();
                const duration = Date.now() - startTime;

                const suiteResults = suite.getResults();
                results.push({
                    suite: suite.name,
                    description: suite.description,
                    duration,
                    results: suiteResults,
                });

                this.logger.info(`${ suite.name } completed in ${ duration }ms`);

            } catch (error) {
                this.logger.error(`${ suite.name } failed`, { error: (error as Error).message });
                results.push({
                    suite: suite.name,
                    description: suite.description,
                    error: (error as Error).message,
                    results: null,
                });
            }
        }

        this.outputResults(results);
    }

    private async runBasicTests (): Promise<void> {
        const test = new BasicTests(this.baseUrl, this.logLevel);
        await test.runAll();
    }

    private async runCodecTests (): Promise<void> {
        const test = new CodecTests(this.baseUrl, this.logLevel);
        await test.runAll();
    }

    private async runBackendTests (): Promise<void> {
        const test = new BackendTests(this.baseUrl, this.logLevel);
        await test.runAll();
    }

    private async runEdgeCaseTests (): Promise<void> {
        const test = new EdgeCaseTests(this.baseUrl, this.logLevel);
        await test.runAll();
    }

    private outputResults (results: any[]): void {
        const options = this.program.opts();

        if (options.json) {
            console.log(JSON.stringify(results, null, 2));
            return;
        }

        if (options.junit) {
            this.outputJUnit(results, options.junit);
            return;
        }

        // Default human-readable output
        console.log('\n' + '='.repeat(80));
        console.log('PIXERVE INTEGRATION TEST RESULTS');
        console.log('='.repeat(80));

        let totalTests = 0;
        let totalPassed = 0;
        let totalFailed = 0;
        let totalDuration = 0;

        for (const result of results) {
            console.log(`\n${ result.suite }`);
            console.log('-'.repeat(result.suite.length));

            if (result.error) {
                console.log(`❌ FAILED: ${ result.error }`);
                totalFailed++;
            } else {
                const suiteResults = result.results.results;
                const passed = suiteResults.filter((r: any) => r.passed).length;
                const failed = suiteResults.filter((r: any) => !r.passed).length;

                console.log(`✅ Passed: ${ passed }`);
                console.log(`❌ Failed: ${ failed }`);
                console.log(`⏱️  Duration: ${ result.duration }ms`);

                totalTests += suiteResults.length;
                totalPassed += passed;
                totalFailed += failed;
                totalDuration += result.duration;
            }
        }

        console.log('\n' + '='.repeat(80));
        console.log('SUMMARY');
        console.log('='.repeat(80));
        console.log(`Total Test Suites: ${ results.length }`);
        console.log(`Total Tests: ${ totalTests }`);
        console.log(`Passed: ${ totalPassed }`);
        console.log(`Failed: ${ totalFailed }`);
        console.log(`Success Rate: ${ totalTests > 0 ? ((totalPassed / totalTests) * 100).toFixed(1) : 0 }%`);
        console.log(`Total Duration: ${ totalDuration }ms`);

        if (totalFailed === 0) {
            console.log('\n🎉 All tests passed!');
            process.exit(0);
        } else {
            console.log(`\n💥 ${ totalFailed } test(s) failed`);
            process.exit(1);
        }
    }

    private outputJUnit (results: any[], filePath: string): void {
        let xml = '<?xml version="1.0" encoding="UTF-8"?>\n';
        xml += '<testsuites>\n';

        for (const result of results) {
            xml += `  <testsuite name="${ result.suite }" tests="${ result.results?.results?.length || 0 }"`;

            if (!result.error) {
                const suiteResults = result.results.results;
                const failures = suiteResults.filter((r: any) => !r.passed).length;
                xml += ` failures="${ failures }" time="${ result.duration / 1000 }"`;
            }

            xml += '>\n';

            if (result.error) {
                xml += `    <testcase name="Suite Execution" classname="${ result.suite }">\n`;
                xml += `      <failure message="${ result.error }"/>\n`;
                xml += '    </testcase>\n';
            } else {
                for (const testResult of result.results.results) {
                    xml += `    <testcase name="${ testResult.name }" classname="${ result.suite }" time="${ testResult.duration / 1000 }">\n`;

                    if (!testResult.passed) {
                        xml += `      <failure message="${ testResult.error || 'Test failed' }"/>\n`;
                    }

                    xml += '    </testcase>\n';
                }
            }

            xml += '  </testsuite>\n';
        }

        xml += '</testsuites>\n';

        fs.writeFileSync(filePath, xml);
        this.logger.info(`JUnit results written to ${ filePath }`);
    }

    async run (): Promise<void> {
        try {
            await this.program.parseAsync();
        } catch (error) {
            this.logger.error('Test runner failed', { error: (error as Error).message });
            process.exit(1);
        }
    }
}

// CLI entry point
if (require.main === module) {
    const runner = new TestRunner();
    runner.run();
}

export default TestRunner;