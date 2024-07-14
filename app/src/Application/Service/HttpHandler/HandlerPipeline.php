<?php

declare(strict_types=1);

namespace App\Application\Service\HttpHandler;

use App\Application\HTTP\Response\ValidationResource;
use Nyholm\Psr7\Response;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Spiral\Core\FactoryInterface;
use Spiral\Filters\Exception\ValidationException;
use Spiral\Tokenizer\Attribute\TargetClass;
use Spiral\Tokenizer\TokenizationListenerInterface;

#[TargetClass(class: HandlerInterface::class)]
final class HandlerPipeline implements HandlerRegistryInterface, CoreHandlerInterface, TokenizationListenerInterface
{
    /** @var HandlerInterface[] */
    private array $handlers = [];
    private int $position = 0;
    private bool $isHandled = false;

    public function __construct(
        private readonly FactoryInterface $factory,
    ) {}

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
            return new Response(404);
        }

        try {
            return $handler->handle(
                $request,
                fn(ServerRequestInterface $request) => $this->handlePipeline($request),
            );
        } catch (ValidationException $e) {
            return (new ValidationResource($e))->toResponse(new Response(422));
        }
    }

    public function listen(\ReflectionClass $class): void
    {
        $handler = $this->factory->make($class->getName());
        \assert($handler instanceof HandlerInterface);
        $this->register($handler);
    }

    public function finalize(): void
    {
        // nothing to do
    }
}
