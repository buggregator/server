<?php

declare(strict_types=1);

namespace App\Application\Service\ClientProxy;

use Spiral\Core\FactoryInterface;
use Spiral\Tokenizer\Attribute\TargetClass;
use Spiral\Tokenizer\TokenizationListenerInterface;

#[TargetClass(EventHandlerInterface::class)]
final class EventHandlersListener implements TokenizationListenerInterface
{
    /** @var class-string<EventHandlerInterface>[] */
    private array $handlers = [];

    public function __construct(
        private readonly EventHandlerRegistryInterface $registry,
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

