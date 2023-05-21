<?php

declare(strict_types=1);

namespace Modules\VarDumper\Application;

use App\Application\Commands\HandleReceivedEvent;
use App\Application\Service\ClientProxy\EventHandlerInterface;
use Buggregator\Client\Proto\Frame;
use Modules\VarDumper\Application\Dump\MessageParser;
use Spiral\Cqrs\CommandBusInterface;
use Symfony\Component\VarDumper\Cloner\Data;
use Symfony\Component\VarDumper\Dumper\HtmlDumper;

final class RequestHandler implements EventHandlerInterface
{
    public function __construct(
        private readonly MessageParser $messageParser,
        private readonly CommandBusInterface $commandBus,
    ) {
    }

    /**
     * @param Frame\VarDumper $frame
     */
    public function handle(Frame $frame): void
    {
        $this->fireEvent(
            $this->messageParser->parse($frame->dump)
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

    public function isSupported(Frame $frame): bool
    {
        return $frame instanceof Frame\VarDumper;
    }
}
