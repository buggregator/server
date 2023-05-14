<?php

declare(strict_types=1);

namespace App\Application\Service\HttpHandler;

use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;

final class HandlerPipeline implements HandlerRegistryInterface, CoreHandlerInterface
{
    /** @var HandlerInterface[] */
    private array $handlers = [];
    private int $position = 0;
    private bool $isHandled = false;

    public function register(HandlerInterface $handler): void
    {
        if ($this->isHandled) {
            throw new \RuntimeException('Cannot register new handler after pipeline is handled.');
        }

        $this->handlers[$handler->priority()][] = $handler;
    }

    public function handle(ServerRequestInterface $request): ResponseInterface
    {
        $this->position = 0;

        if (!$this->isHandled) {
            \ksort($this->handlers);
            $newHandlers = [];
            foreach ($this->handlers as $handlers) {
                foreach ($handlers as $handler) {
                    $newHandlers[] = $handler;
                }
            }

            $this->handlers = $newHandlers;
            $this->isHandled = true;
        }

        return $this->handlePipeline($request);
    }

    private function handlePipeline(ServerRequestInterface $request): ResponseInterface
    {
        $handler = $this->handlers[$this->position] ?? null;
        $this->position++;

        if ($handler === null) {
            throw new \RuntimeException('No more handlers in the pipeline.');
        }

        return $handler->handle(
            $request,
            fn(ServerRequestInterface $request) => $this->handlePipeline($request)
        );
    }
}
