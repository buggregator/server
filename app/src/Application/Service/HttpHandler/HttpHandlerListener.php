<?php

declare(strict_types=1);

namespace App\Application\Service\HttpHandler;

use Spiral\Core\FactoryInterface;
use Spiral\Tokenizer\Attribute\TargetClass;
use Spiral\Tokenizer\TokenizationListenerInterface;

#[TargetClass(HandlerInterface::class)]
final class HttpHandlerListener implements TokenizationListenerInterface
{
    /** @var class-string<HandlerInterface>[] */
    private array $handlers = [];

    public function __construct(
        private readonly HandlerRegistryInterface $registry,
        private readonly FactoryInterface $factory,
    ) {
    }

    public function listen(\ReflectionClass $class): void
    {
        $this->handlers[] = $class->getName();
    }

    public function finalize(): void
    {
        foreach ($this->handlers as $class) {
            $this->registry->register(
                $this->factory->make($class)
            );
        }
    }
}
