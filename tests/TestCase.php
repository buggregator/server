<?php

declare(strict_types=1);

namespace Tests;

use App\Application\Service\ErrorHandler\Handler;
use Modules\Events\Domain\EventRepositoryInterface;
use Psr\SimpleCache\CacheInterface;
use Spiral\Cache\Storage\ArrayStorage;
use Spiral\Core\Container;
use Spiral\Core\ContainerScope;
use Spiral\Testing\TestableKernelInterface;
use Spiral\Testing\TestCase as BaseTestCase;
use Tests\App\Broadcasting\BroadcastFaker;
use Tests\App\Events\EventsMocker;
use Tests\App\TestKernel;

class TestCase extends BaseTestCase
{
    protected BroadcastFaker $broadcastig;
    private ?EventsMocker $events = null;

    protected function setUp(): void
    {
        parent::setUp();

        // Bind container to ContainerScope
        (new \ReflectionClass(ContainerScope::class))->setStaticPropertyValue('container', $this->getContainer());
        $this->broadcastig = new BroadcastFaker($this->getContainer());
    }

    public function createAppInstance(Container $container = new Container()): TestableKernelInterface
    {
        return TestKernel::create(
            directories: $this->defineDirectories(
                $this->rootDirectory(),
            ),
            exceptionHandler: Handler::class,
            container: $container,
        );
    }

    protected function tearDown(): void
    {
        parent::tearDown();

        // Uncomment this line if you want to clean up runtime directory.
        // $this->cleanUpRuntimeDirectory();

        (new \ReflectionClass(ContainerScope::class))->setStaticPropertyValue('container', null);
        $this->broadcastig->reset();
    }

    public function rootDirectory(): string
    {
        return __DIR__ . '/..';
    }

    public function defineDirectories(string $root): array
    {
        return [
            'root' => $root,
            'modules' => $root . '/app/modules',
            'public' => $root . '/frontend/.output/public',
        ];
    }

    /**
     * @template T
     *
     * @param class-string<T>|string $id
     *
     * @return T|mixed
     * @psalm-return ($id is class-string ? T : mixed)
     *
     * @throws \Throwable
     */
    public function get(string $id): mixed
    {
        return $this->getApp()->getContainer()->get($id);
    }

    public function fakeEvents(): EventsMocker
    {
        if ($this->events === null) {
            $this->events = new EventsMocker(
                $this->mockContainer(EventRepositoryInterface::class),
            );
        }

        return $this->events;
    }
}
