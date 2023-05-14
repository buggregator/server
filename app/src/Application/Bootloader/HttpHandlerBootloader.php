<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use App\Application\Service\HttpHandler\CoreHandlerInterface;
use App\Application\Service\HttpHandler\HandlerPipeline;
use App\Application\Service\HttpHandler\HandlerRegistryInterface;
use App\Application\Service\HttpHandler\HttpHandlerListener;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Tokenizer\TokenizerListenerRegistryInterface;

final class HttpHandlerBootloader extends Bootloader
{
    protected const SINGLETONS = [
        HandlerPipeline::class => HandlerPipeline::class,
        HandlerRegistryInterface::class => HandlerPipeline::class,
        CoreHandlerInterface::class => HandlerPipeline::class,
    ];

    public function boot(
        TokenizerListenerRegistryInterface $listenerRegistry,
        HttpHandlerListener $listener,
    ): void {
        $listenerRegistry->addListener($listener);
    }
}
