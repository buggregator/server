<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Sentry;

use App\Application\Broadcasting\Channel\EventsChannel;
use Modules\Projects\Domain\Project;
use Modules\Projects\Domain\ValueObject\Key;
use Nyholm\Psr7\Stream;
use Tests\App\Http\ResponseAssertions;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class SentryV4ActionTest extends ControllerTestCase
{
    protected const JSON = <<<'BODY'
{"event_id":"2b4f7918973f4371933dce5b3ac381bd","sent_at":"2023-12-01T18:30:35Z","dsn":"http:\/\/user@127.0.0.1:8082\/1","sdk":{"name":"sentry.php","version":"4.0.1"},"trace":{"trace_id":"143ef743ce184eb7abd0ae0891d33b7d","public_key":"user"}}
{"type":"event","content_type":"application\/json"}
{"event_id":"2b4f7918973f4371933dce5b3ac381bd","timestamp":1701455435.634665,"platform":"php","sdk":{"name":"sentry.php","version":"4.0.1"},"server_name":"Test","environment":"production","modules":{"amphp\/amp":"v2.6.2","amphp\/byte-stream":"v1.8.1","brick\/math":"0.11.0","buggregator\/app":"dev-master@818ea82","clue\/stream-filter":"v1.6.0","cocur\/slugify":"v3.2","codedungeon\/php-cli-colors":"1.12.2","composer\/pcre":"3.1.1","composer\/semver":"3.4.0","composer\/xdebug-handler":"3.0.3","cycle\/annotated":"v3.4.0","cycle\/database":"2.6.1","cycle\/migrations":"v4.2.1","cycle\/orm":"v2.5.0","cycle\/schema-builder":"v2.6.1","cycle\/schema-migrations-generator":"2.2.0","cycle\/schema-renderer":"1.2.0","defuse\/php-encryption":"v2.4.0","dnoegel\/php-xdg-base-dir":"v0.1.1","doctrine\/annotations":"2.0.1","doctrine\/collections":"1.8.0","doctrine\/deprecations":"1.1.2","doctrine\/inflector":"2.0.8","doctrine\/instantiator":"2.0.0","doctrine\/lexer":"3.0.0","egulias\/email-validator":"4.0.2","felixfbecker\/advanced-json-rpc":"v3.2.1","felixfbecker\/language-server-protocol":"v1.5.2","fidry\/cpu-core-counter":"0.5.1","google\/common-protos":"v4.5.0","google\/protobuf":"v3.25.1","graham-campbell\/result-type":"v1.1.2","grpc\/grpc":"1.57.0","guzzlehttp\/psr7":"2.6.1","hamcrest\/hamcrest-php":"v2.0.1","jean85\/pretty-package-versions":"2.0.5","league\/event":"3.0.2","league\/flysystem":"2.5.0","league\/mime-type-detection":"1.14.0","mockery\/mockery":"1.6.6","monolog\/monolog":"2.9.2","myclabs\/deep-copy":"1.11.1","nesbot\/carbon":"2.71.0","netresearch\/jsonmapper":"v4.2.0","nette\/php-generator":"v4.1.2","nette\/utils":"v4.0.3","nikic\/php-parser":"v4.17.1","nyholm\/psr7":"1.8.1","paragonie\/random_compat":"v9.99.100","phar-io\/manifest":"2.0.3","phar-io\/version":"3.2.1","php-http\/message":"1.16.0","phpdocumentor\/reflection-common":"2.2.0","phpdocumentor\/reflection-docblock":"5.3.0","phpdocumentor\/type-resolver":"1.7.3","phpoption\/phpoption":"1.9.2","phpstan\/phpdoc-parser":"1.24.4","phpunit\/php-code-coverage":"9.2.29","phpunit\/php-file-iterator":"3.0.6","phpunit\/php-invoker":"3.1.1","phpunit\/php-text-template":"2.0.4","phpunit\/php-timer":"5.0.3","phpunit\/phpunit":"9.6.15","pimple\/pimple":"v3.5.0","psr\/cache":"3.0.0","psr\/clock":"1.0.0","psr\/container":"2.0.2","psr\/event-dispatcher":"1.0.0","psr\/http-factory":"1.0.2","psr\/http-message":"2.0","psr\/http-server-handler":"1.0.2","psr\/http-server-middleware":"1.0.2","psr\/log":"3.0.0","psr\/simple-cache":"3.0.0","qossmic\/deptrac-shim":"1.0.2","ralouphie\/getallheaders":"3.0.3","ramsey\/collection":"2.0.0","ramsey\/uuid":"4.7.5","roadrunner-php\/app-logger":"1.1.0","roadrunner-php\/centrifugo":"2.0.0","roadrunner-php\/roadrunner-api-dto":"1.4.0","sebastian\/cli-parser":"1.0.1","sebastian\/code-unit":"1.0.8","sebastian\/code-unit-reverse-lookup":"2.0.3","sebastian\/comparator":"4.0.8","sebastian\/complexity":"2.0.2","sebastian\/diff":"4.0.5","sebastian\/environment":"5.1.5","sebastian\/exporter":"4.0.5","sebastian\/global-state":"5.0.6","sebastian\/lines-of-code":"1.0.3","sebastian\/object-enumerator":"4.0.4","sebastian\/object-reflector":"2.0.4","sebastian\/recursion-context":"4.0.5","sebastian\/resource-operations":"3.0.3","sebastian\/type":"3.2.1","sebastian\/version":"3.0.2","sentry\/sdk":"4.0.0","sentry\/sentry":"4.0.1","spatie\/array-to-xml":"3.2.2","spiral-packages\/cqrs":"v2.3.0","spiral-packages\/league-event":"1.0.1","spiral\/attributes":"v3.1.2","spiral\/composer-publish-plugin":"v1.1.2","spiral\/cycle-bridge":"v2.8.0","spiral\/data-grid":"v3.0.0","spiral\/data-grid-bridge":"v3.0.1","spiral\/framework":"3.10.0","spiral\/goridge":"4.1.0","spiral\/nyholm-bridge":"v1.3.0","spiral\/roadrunner":"v2023.3.7","spiral\/roadrunner-bridge":"3.0.2","spiral\/roadrunner-grpc":"3.2.0","spiral\/roadrunner-http":"3.2.0","spiral\/roadrunner-jobs":"4.3.0","spiral\/roadrunner-kv":"4.0.0","spiral\/roadrunner-metrics":"3.1.0","spiral\/roadrunner-services":"2.1.0","spiral\/roadrunner-tcp":"3.0.0","spiral\/roadrunner-worker":"3.2.0","spiral\/testing":"2.6.2","spiral\/validator":"1.5.0","symfony\/clock":"v7.0.0","symfony\/console":"v6.4.1","symfony\/deprecation-contracts":"v3.4.0","symfony\/event-dispatcher":"v7.0.0","symfony\/event-dispatcher-contracts":"v3.4.0","symfony\/filesystem":"v7.0.0","symfony\/finder":"v6.4.0","symfony\/mailer":"v6.4.0","symfony\/messenger":"v6.4.0","symfony\/mime":"v6.4.0","symfony\/options-resolver":"v7.0.0","symfony\/polyfill-ctype":"v1.28.0","symfony\/polyfill-iconv":"v1.28.0","symfony\/polyfill-intl-grapheme":"v1.28.0","symfony\/polyfill-intl-idn":"v1.28.0","symfony\/polyfill-intl-normalizer":"v1.28.0","symfony\/polyfill-mbstring":"v1.28.0","symfony\/polyfill-php72":"v1.28.0","symfony\/polyfill-php80":"v1.28.0","symfony\/polyfill-php83":"v1.28.0","symfony\/process":"v6.4.0","symfony\/service-contracts":"v3.4.0","symfony\/string":"v7.0.0","symfony\/translation":"v6.4.0","symfony\/translation-contracts":"v3.4.0","symfony\/var-dumper":"v6.4.0","symfony\/yaml":"v7.0.0","theseer\/tokenizer":"1.2.2","vimeo\/psalm":"5.16.0","vlucas\/phpdotenv":"v5.6.0","webmozart\/assert":"1.11.0","yiisoft\/friendly-exception":"1.1.0","zbateson\/mail-mime-parser":"2.4.0","zbateson\/mb-wrapper":"1.2.0","zbateson\/stream-decorators":"1.2.1","zentlix\/swagger-php":"1.x-dev@1f4927a","zircote\/swagger-php":"4.7.16"},"contexts":{"os":{"name":"Linux","version":"5.15.133.1-microsoft-standard-WSL2","build":"#1 SMP Thu Oct 5 21:02:42 UTC 2023","kernel_version":"Linux Test 5.15.133.1-microsoft-standard-WSL2 #1 SMP Thu Oct 5 21:02:42 UTC 2023 x86_64"},"runtime":{"name":"php","version":"8.2.5"},"trace":{"trace_id":"143ef743ce184eb7abd0ae0891d33b7d","span_id":"e4a276672c8a4a38"}},"exception":{"values":[{"type":"Exception","value":"test","stacktrace":{"frames":[{"filename":"\/vendor\/phpunit\/phpunit\/phpunit","lineno":107,"in_app":true,"abs_path":"\\/vendor\/phpunit\/phpunit\/phpunit","pre_context":["","unset($options);","","require PHPUNIT_COMPOSER_INSTALL;",""],"context_line":"PHPUnit\\TextUI\\Command::main();","post_context":[""]},{"filename":"\/vendor\/phpunit\/phpunit\/src\/TextUI\/Command.php","lineno":97,"in_app":true,"abs_path":"\\/vendor\/phpunit\/phpunit\/src\/TextUI\/Command.php","function":"PHPUnit\\TextUI\\Command::main","raw_function":"PHPUnit\\TextUI\\Command::main","pre_context":["     * @throws Exception","     *\/","    public static function main(bool $exit = true): int","    {","        try {"],"context_line":"            return (new static)->run($_SERVER['argv'], $exit);","post_context":["        } catch (Throwable $t) {","            throw new RuntimeException(","                $t->getMessage(),","                (int) $t->getCode(),","                $t,"]},{"filename":"\/vendor\/phpunit\/phpunit\/src\/TextUI\/Command.php","lineno":144,"in_app":true,"abs_path":"\\/vendor\/phpunit\/phpunit\/src\/TextUI\/Command.php","function":"PHPUnit\\TextUI\\Command::run","raw_function":"PHPUnit\\TextUI\\Command::run","pre_context":["        }","","        unset($this->arguments['test'], $this->arguments['testFile']);","","        try {"],"context_line":"            $result = $runner->run($suite, $this->arguments, $this->warnings, $exit);","post_context":["        } catch (Throwable $t) {","            print $t->getMessage() . PHP_EOL;","        }","","        $return = TestRunner::FAILURE_EXIT;"]},{"filename":"\/vendor\/phpunit\/phpunit\/src\/TextUI\/TestRunner.php","lineno":651,"in_app":true,"abs_path":"\\/vendor\/phpunit\/phpunit\/src\/TextUI\/TestRunner.php","function":"PHPUnit\\TextUI\\TestRunner::run","raw_function":"PHPUnit\\TextUI\\TestRunner::run","pre_context":["            if ($extension instanceof BeforeFirstTestHook) {","                $extension->executeBeforeFirstTest();","            }","        }",""],"context_line":"        $suite->run($result);","post_context":["","        foreach ($this->extensions as $extension) {","            if ($extension instanceof AfterLastTestHook) {","                $extension->executeAfterLastTest();","            }"]},{"filename":"\/vendor\/phpunit\/phpunit\/src\/Framework\/TestSuite.php","lineno":684,"in_app":true,"abs_path":"\\/vendor\/phpunit\/phpunit\/src\/Framework\/TestSuite.php","function":"PHPUnit\\Framework\\TestSuite::run","raw_function":"PHPUnit\\Framework\\TestSuite::run","pre_context":["                $test->setBackupGlobals($this->backupGlobals);","                $test->setBackupStaticAttributes($this->backupStaticAttributes);","                $test->setRunTestInSeparateProcess($this->runTestInSeparateProcess);","            }",""],"context_line":"            $test->run($result);","post_context":["        }","","        if ($this->testCase && class_exists($this->name, false)) {","            foreach ($hookMethods['afterClass'] as $afterClassMethod) {","                if (method_exists($this->name, $afterClassMethod)) {"]},{"filename":"\/vendor\/phpunit\/phpunit\/src\/Framework\/TestSuite.php","lineno":684,"in_app":true,"abs_path":"\\/vendor\/phpunit\/phpunit\/src\/Framework\/TestSuite.php","function":"PHPUnit\\Framework\\TestSuite::run","raw_function":"PHPUnit\\Framework\\TestSuite::run","pre_context":["                $test->setBackupGlobals($this->backupGlobals);","                $test->setBackupStaticAttributes($this->backupStaticAttributes);","                $test->setRunTestInSeparateProcess($this->runTestInSeparateProcess);","            }",""],"context_line":"            $test->run($result);","post_context":["        }","","        if ($this->testCase && class_exists($this->name, false)) {","            foreach ($hookMethods['afterClass'] as $afterClassMethod) {","                if (method_exists($this->name, $afterClassMethod)) {"]},{"filename":"\/vendor\/phpunit\/phpunit\/src\/Framework\/TestCase.php","lineno":968,"in_app":true,"abs_path":"\\/vendor\/phpunit\/phpunit\/src\/Framework\/TestCase.php","function":"PHPUnit\\Framework\\TestCase::run","raw_function":"PHPUnit\\Framework\\TestCase::run","pre_context":["            $template->setVar($var);","","            $php = AbstractPhpProcess::factory();","            $php->runTestJob($template->render(), $this, $result, $processResultFile);","        } else {"],"context_line":"            $result->run($this);","post_context":["        }","","        $this->result = null;","","        return $result;"]},{"filename":"\/vendor\/phpunit\/phpunit\/src\/Framework\/TestResult.php","lineno":728,"in_app":true,"abs_path":"\\/vendor\/phpunit\/phpunit\/src\/Framework\/TestResult.php","function":"PHPUnit\\Framework\\TestResult::run","raw_function":"PHPUnit\\Framework\\TestResult::run","pre_context":["                        $_timeout = $this->defaultTimeLimit;","                }","","                $invoker->invoke([$test, 'runBare'], [], $_timeout);","            } else {"],"context_line":"                $test->runBare();","post_context":["            }","        } catch (TimeoutException $e) {","            $this->addFailure(","                $test,","                new RiskyTestError("]},{"filename":"\/vendor\/phpunit\/phpunit\/src\/Framework\/TestCase.php","lineno":1218,"in_app":true,"abs_path":"\\/vendor\/phpunit\/phpunit\/src\/Framework\/TestCase.php","function":"PHPUnit\\Framework\\TestCase::runBare","raw_function":"PHPUnit\\Framework\\TestCase::runBare","pre_context":["","            foreach ($hookMethods['preCondition'] as $method) {","                $this->{$method}();","            }",""],"context_line":"            $this->testResult = $this->runTest();","post_context":["            $this->verifyMockObjects();","","            foreach ($hookMethods['postCondition'] as $method) {","                $this->{$method}();","            }"]},{"filename":"\/vendor\/phpunit\/phpunit\/src\/Framework\/TestCase.php","lineno":1612,"in_app":true,"abs_path":"\\/vendor\/phpunit\/phpunit\/src\/Framework\/TestCase.php","function":"PHPUnit\\Framework\\TestCase::runTest","raw_function":"PHPUnit\\Framework\\TestCase::runTest","pre_context":["        $testArguments = array_merge($this->data, $this->dependencyInput);","","        $this->registerMockObjectsFromTestArguments($testArguments);","","        try {"],"context_line":"            $testResult = $this->{$this->name}(...array_values($testArguments));","post_context":["        } catch (Throwable $exception) {","            if (!$this->checkExceptionExpectations($exception)) {","                throw $exception;","            }",""]},{"filename":"\/tests\/Feature\/Interfaces\/Http\/Sentry\/SentryV4ActionTest.php","lineno":14,"in_app":true,"abs_path":"\\/tests\/Feature\/Interfaces\/Http\/Sentry\/SentryV4ActionTest.php","function":"Interfaces\\Http\\Sentry\\SentryV4ActionTest::testSend","raw_function":"Interfaces\\Http\\Sentry\\SentryV4ActionTest::testSend","pre_context":["final class SentryV4ActionTest extends ControllerTestCase","{","    public function testSend(): void","    {","        \\Sentry\\init(['dsn' => 'http:\/\/user@127.0.0.1:8082\/1']);"],"context_line":"        \\Sentry\\captureException(new \\Exception('test'));","post_context":["","\/\/        $this->http","\/\/            ->postJson(","\/\/                uri: '\/api\/1\/store\/',","\/\/                data: Stream::create("]}]},"mechanism":{"type":"generic","handled":true,"data":{"code":0}}}]}}
BODY;

    private Project $project;

    protected function setUp(): void
    {
        parent::setUp();

        $this->project = $this->createProject('default');
    }

    public function testSendWithoutGzip(): void
    {
        $this->makeRequest(project: $this->project->getKey())->assertOk();

        $this->broadcastig->assertPushed(new EventsChannel($this->project->getKey()), function (array $data) {
            $this->assertSame('event.received', $data['event']);
            $this->assertSame('sentry', $data['data']['type']);
            $this->assertSame('default', $data['data']['project']);

            $this->assertSame('Test', $data['data']['payload']['server_name']);
            $this->assertSame('production', $data['data']['payload']['environment']);

            $this->assertCount(3, $data['data']['payload']['exception']['values'][0]['stacktrace']['frames']);

            $this->assertNotEmpty($data['data']['uuid']);
            $this->assertNotEmpty($data['data']['timestamp']);

            return true;
        });
    }

    public function testSendGzipped(): void
    {
        $this->http
            ->postJson(
                uri: '/api/default/envelope/',
                data: Stream::create(\gzcompress(self::JSON, -1, \ZLIB_ENCODING_GZIP)),
                headers: [
                    'Content-Encoding' => 'gzip',
                    'X-Buggregator-Event' => 'sentry',
                    'Content-Type' => 'application/x-sentry-envelope',
                    'X-Sentry-Auth' => 'Sentry sentry_version=7, sentry_client=sentry.php/4.0.1, sentry_key=user',
                ],
            )->assertOk();

        $this->broadcastig->assertPushed(new EventsChannel('default'), function (array $data) {
            $this->assertSame('event.received', $data['event']);
            $this->assertSame('sentry', $data['data']['type']);
            $this->assertSame('default', $data['data']['project']);

            $this->assertSame('Test', $data['data']['payload']['server_name']);
            $this->assertSame('production', $data['data']['payload']['environment']);

            $this->assertNotEmpty($data['data']['uuid']);
            $this->assertNotEmpty($data['data']['timestamp']);


            return true;
        });
    }

    public function testSendGzippedSpiral(): void
    {
        $this->http
            ->postJson(
                uri: '/api/default/envelope/',
                data: Stream::create(\gzcompress(self::JSON, -1, \ZLIB_ENCODING_GZIP)),
                headers: [
                    'Content-Encoding' => 'gzip',
                    'X-Buggregator-Event' => 'sentry',
                    'Content-Type' => 'application/json',
                    'X-Sentry-Auth' => 'Sentry sentry_version=7, sentry_client=sentry.php.spiral/3.1.2, sentry_key=sentry',
                ],
            )->assertOk();

        $this->broadcastig->assertPushed(new EventsChannel('default'), function (array $data) {
            $this->assertSame('event.received', $data['event']);
            $this->assertSame('sentry', $data['data']['type']);

            $this->assertSame('Test', $data['data']['payload']['server_name']);
            $this->assertSame('production', $data['data']['payload']['environment']);

            $this->assertNotEmpty($data['data']['uuid']);
            $this->assertNotEmpty($data['data']['timestamp']);

            return true;
        });
    }

    private function makeRequest(string $secret = 'secret', string|Key $project = 'default'): ResponseAssertions
    {
        return $this->http
            ->postJson(
                uri: '/api/' . $project . '/envelope/',
                data: Stream::create(self::JSON),
                headers: [
                    'X-Buggregator-Event' => 'sentry',
                    'Content-Type' => 'application/x-sentry-envelope',
                    'X-Sentry-Auth' => 'Sentry sentry_version=7, sentry_client=sentry.php/4.0.1, sentry_key=' . $secret,
                ],
            );
    }
}
