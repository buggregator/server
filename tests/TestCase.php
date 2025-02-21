<?php

declare(strict_types=1);

namespace Tests;

use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Event\EventTypeMapperInterface;
use App\Application\Service\ErrorHandler\Handler;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\EventRepositoryInterface;
use Spiral\Core\Container;
use Spiral\Core\ContainerScope;
use Spiral\Core\InvokerInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Cqrs\CommandInterface;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Cqrs\QueryInterface;
use Spiral\Testing\TestableKernelInterface;
use Spiral\Testing\TestCase as BaseTestCase;
use Tests\App\Broadcasting\BroadcastFaker;
use Tests\App\Console\SpyConsoleInvoker;
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
            'public' => $root . '/frontend/assets',
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
        if (!$this->events instanceof EventsMocker) {
            $this->events = new EventsMocker(
                $this->mockContainer(EventRepositoryInterface::class),
            );
        }

        return $this->events;
    }

    protected function randomUuid(): Uuid
    {
        return Uuid::generate();
    }

    protected function dispatchCommand(CommandInterface $command): mixed
    {
        return $this->get(CommandBusInterface::class)->dispatch($command);
    }

    protected function dispatchQuery(QueryInterface $query): mixed
    {
        return $this->get(QueryBusInterface::class)->ask($query);
    }

    protected function mapEventTypeToPreview(Event $event): array|\JsonSerializable
    {
        return $this->get(EventTypeMapperInterface::class)->toPreview(
            $event->getType(),
            $event->getPayload(),
        );
    }

    protected function spyConsole(\Closure $closure, array $commandsToRun): SpyConsoleInvoker
    {
        $this->getContainer()->runScope([
            InvokerInterface::class => $fakeInvoker = new SpyConsoleInvoker(
                $this->get(InvokerInterface::class),
                $commandsToRun,
            ),
        ], $closure);

        return $fakeInvoker;
    }
}
