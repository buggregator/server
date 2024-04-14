<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use App\Application\Service\HttpHandler\CoreHandlerInterface;
use App\Application\Service\HttpHandler\HandlerPipeline;
use App\Application\Service\HttpHandler\HandlerRegistryInterface;
use App\Interfaces\Http\Handler\FrontendRequest;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Boot\DirectoriesInterface;
use Spiral\Core\FactoryInterface;
use Spiral\Tokenizer\TokenizerListenerRegistryInterface;

final class HttpHandlerBootloader extends Bootloader
{
    public function defineSingletons(): array
    {
        return [
            HandlerPipeline::class => static fn(
                FactoryInterface $factory,
            ): HandlerPipeline => new HandlerPipeline(factory: $factory),
            HandlerRegistryInterface::class => HandlerPipeline::class,
            CoreHandlerInterface::class => HandlerPipeline::class,
            FrontendRequest::class => static fn(
                DirectoriesInterface $dirs,
            ): FrontendRequest => new FrontendRequest(
                $dirs->get('public'),
            ),
        ];
    }

    public function init(TokenizerListenerRegistryInterface $tokenizerRegistry, HandlerPipeline $pipeline): void
    {
        $tokenizerRegistry->addListener($pipeline);
    }
}
