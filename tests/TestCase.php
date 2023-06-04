<?php

declare(strict_types=1);

namespace Tests;

use App\Application\Service\ErrorHandler\Handler;
use Buggregator\Client\Test\Mock\StreamClientMock;
use Buggregator\Client\Traffic\StreamClient;
use Spiral\Core\Container;
use Spiral\Testing\TestableKernelInterface;
use Spiral\Testing\TestCase as BaseTestCase;
use Tests\App\TestKernel;
use Buggregator\Client\Proto\Frame;
use Buggregator\Client\Traffic\Message;
use Buggregator\Client\Traffic\Parser;

class TestCase extends BaseTestCase
{
    public function createAppInstance(Container $container = new Container()): TestableKernelInterface
    {
        return TestKernel::create(
            directories: $this->defineDirectories(
                $this->rootDirectory()
            ),
            exceptionHandler: Handler::class,
            container: $container
        );
    }

    protected function tearDown(): void
    {
        // Uncomment this line if you want to clean up runtime directory.
        // $this->cleanUpRuntimeDirectory();
    }

    public function rootDirectory(): string
    {
        return __DIR__ . '/..';
    }

    public function defineDirectories(string $root): array
    {
        return [
            'root' => $root,
            'modules' => __DIR__ . '/../app/modules',
            'public' => __DIR__ . '/../frontend/.output/public',
        ];
    }

    public function mockStreamClient(array|string $body): StreamClient
    {
        return StreamClientMock::createFromGenerator(
            (static function () use ($body) {
                if (\is_string($body)) {
                    yield $body;
                    return;
                }
                yield from $body;
            })()
        );
    }

    public function createSmtpFrame(array|string $body): Frame\Smtp
    {
        return new Frame\Smtp(
            $this->runInFiber(fn() => (new Parser\Smtp)->parseStream([], $this->mockStreamClient($body)))
        );
    }

    /**
     * @template T
     *
     * @param \Closure(): T $callback
     *
     * @return T
     * @throws \Throwable
     */
    public function runInFiber(\Closure $callback): mixed
    {
        $fiber = new \Fiber($callback);
        $fiber->start();
        do {
            if ($fiber->isTerminated()) {
                return $fiber->getReturn();
            }
            $fiber->resume();
        } while (true);
    }
}
