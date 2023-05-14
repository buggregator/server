<?php

declare(strict_types=1);

namespace Modules\VarDumper\Application;

use App\Application\Commands\HandleReceivedEvent;
use Modules\VarDumper\Application\Dump\MessageParser;
use Spiral\Cqrs\CommandBusInterface;
use Symfony\Component\VarDumper\Cloner\Data;
use Symfony\Component\VarDumper\Dumper\HtmlDumper;

final class RequestHandler
{
    public function __construct(
        private readonly MessageParser $messageParser,
        private readonly CommandBusInterface $commandBus,
    ) {
    }

    public function handle(string $payload): void
    {
        $this->fireEvent(
            $this->messageParser->parse($payload)
        );
    }

    private function fireEvent(array $payload): void
    {
        $this->commandBus->dispatch(
            new HandleReceivedEvent(
                type: 'var-dump',
                payload: [
                    'payload' => [
                        'type' => $payload[0]->getType(),
                        'value' => $this->convertToPrimitive($payload[0]),
                    ],
                    'context' => $payload[1],
                ]
            )
        );
    }

    private function convertToPrimitive(Data $data): string|null
    {
        if (\in_array($data->getType(), ['string', 'boolean'])) {
            return (string)$data->getValue();
        }

        $dumper = new HtmlDumper();

        return $dumper->dump($data, true);
    }
}
