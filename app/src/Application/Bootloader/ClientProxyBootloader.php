<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use App\Application\Service\ClientProxy\CoreHandlerInterface;
use App\Application\Service\ClientProxy\EventHandlerRegistry;
use App\Application\Service\ClientProxy\EventHandlerRegistryInterface;
use App\Application\Service\ClientProxy\EventHandlersListener;
use App\Application\Service\ClientProxy\InternalSender;
use Buggregator\Client\Application;
use Buggregator\Client\Config\SocketServer;
use Buggregator\Client\Proto\Server\Decoder;
use Buggregator\Client\Proto\Server\Version\V1;
use Buggregator\Client\Sender\FileSender;
use Buggregator\Client\Sender\SenderRegistry;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Boot\DirectoriesInterface;
use Spiral\Boot\EnvironmentInterface;
use Spiral\Tokenizer\TokenizerListenerRegistryInterface;

final class ClientProxyBootloader extends Bootloader
{
    protected const SINGLETONS = [
        Decoder::class => [self::class, 'initDecoder'],
        Application::class => [self::class, 'initApplication'],
        EventHandlerRegistry::class => EventHandlerRegistry::class,
        EventHandlerRegistryInterface::class => EventHandlerRegistry::class,
        CoreHandlerInterface::class => EventHandlerRegistry::class,
        SenderRegistry::class => [self::class, 'createRegistry']
    ];

    public function boot(
        TokenizerListenerRegistryInterface $listenerRegistry,
        EventHandlersListener $listener,
    ): void {
        $listenerRegistry->addListener($listener);
    }

    private function initApplication(EnvironmentInterface $env): Application
    {
        return new Application([
            new SocketServer(9912, $env->get('DUMP_SERVER_HOST', '127.0.0.1'))
        ]);
    }

    private function createRegistry(DirectoriesInterface $dirs, InternalSender $sender): SenderRegistry
    {
        $registry = new SenderRegistry();
        $registry->register('file', new FileSender($dirs->get('runtime') . '/dumps'));
        $registry->register('internal', $sender);

        return $registry;
    }

    private function initDecoder(): Decoder
    {
        return new Decoder([
            new V1()
        ]);
    }
}
