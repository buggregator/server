<?php

declare(strict_types=1);

namespace Modules\VarDumper\Interfaces\TCP;

use App\Application\Commands\HandleReceivedEvent;
use Modules\VarDumper\Application\Dump\MessageParser;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\RoadRunner\Tcp\Request;
use Spiral\RoadRunner\Tcp\TcpWorkerInterface;
use Spiral\RoadRunnerBridge\Tcp\Response\ContinueRead;
use Spiral\RoadRunnerBridge\Tcp\Response\ResponseInterface;
use Spiral\RoadRunnerBridge\Tcp\Service\ServiceInterface;
use Symfony\Component\VarDumper\Cloner\Data;
use Symfony\Component\VarDumper\Dumper\HtmlDumper;

class Service implements ServiceInterface
{
    public function __construct(
        private readonly CommandBusInterface $commandBus,
    ) {
    }

    public function handle(Request $request): ResponseInterface
    {
        dump($request);
        if ($request->event === TcpWorkerInterface::EVENT_CONNECTED) {
            return new ContinueRead();
        }

        $messages = \array_filter(\explode("\n", $request->body));

        foreach ($messages as $message) {
            $payload = (new MessageParser())->parse($message);
            $this->fireEvent($payload);
        }

        return new ContinueRead();
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
