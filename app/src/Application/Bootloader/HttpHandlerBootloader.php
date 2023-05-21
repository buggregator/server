<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use App\Application\Service\HttpHandler\CoreHandlerInterface;
use App\Application\Service\HttpHandler\HandlerPipeline;
use App\Application\Service\HttpHandler\HandlerRegistryInterface;
use App\Interfaces\Http\FrontendRequest;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Boot\DirectoriesInterface;

final class HttpHandlerBootloader extends Bootloader
{
    protected const SINGLETONS = [
        HandlerPipeline::class => [self::class, 'initHandlerPipeline'],
        HandlerRegistryInterface::class => HandlerPipeline::class,
        CoreHandlerInterface::class => HandlerPipeline::class,
    ];

    private function initHandlerPipeline(DirectoriesInterface $dirs): HandlerPipeline
    {
        $pipeline = new HandlerPipeline();

        $pipeline->register(
            new FrontendRequest(
                $dirs->get('public')
            )
        );

        return $pipeline;
    }
}
