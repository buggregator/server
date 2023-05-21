<?php

declare(strict_types=1);

namespace App\Application\Service\ClientProxy;

use Buggregator\Client\Proto\Frame;
use Buggregator\Client\Sender;

final class InternalSender implements Sender
{
    public function __construct(
        private readonly EventHandlerRegistryInterface $handlerRegistry,
    ) {
    }

    public function send(iterable $frames): void
    {
        foreach ($frames as $event) {
            $this->handlePayload($event);
        }
    }

    private function handlePayload(Frame $payload): void
    {
        $this->handlerRegistry->handle($payload);
    }
}
