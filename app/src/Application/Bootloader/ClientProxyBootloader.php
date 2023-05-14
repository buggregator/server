<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use App\Application\Service\ClientProxy\CoreHandlerInterface;
use App\Application\Service\ClientProxy\EventHandlerRegistry;
use App\Application\Service\ClientProxy\EventHandlerRegistryInterface;
use App\Application\Service\ClientProxy\EventHandlersListener;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Tokenizer\TokenizerListenerRegistryInterface;

final class ClientProxyBootloader extends Bootloader
{
    protected const SINGLETONS = [
        EventHandlerRegistry::class => EventHandlerRegistry::class,
        EventHandlerRegistryInterface::class => EventHandlerRegistry::class,
        CoreHandlerInterface::class => EventHandlerRegistry::class,
    ];

    public function boot(
        TokenizerListenerRegistryInterface $listenerRegistry,
        EventHandlersListener $listener,
    ): void {
        $listenerRegistry->addListener($listener);
    }
}
