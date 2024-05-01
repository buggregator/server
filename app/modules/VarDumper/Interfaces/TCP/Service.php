<?php

declare(strict_types=1);

namespace Modules\VarDumper\Interfaces\TCP;

use App\Application\Commands\HandleReceivedEvent;
use Modules\VarDumper\Application\Dump\BodyInterface;
use Modules\VarDumper\Application\Dump\DumpIdGeneratorInterface;
use Modules\VarDumper\Application\Dump\HtmlBody;
use Modules\VarDumper\Application\Dump\HtmlDumper;
use Modules\VarDumper\Application\Dump\MessageParser;
use Modules\VarDumper\Application\Dump\ParsedPayload;
use Modules\VarDumper\Application\Dump\PrimitiveBody;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\RoadRunner\Tcp\Request;
use Spiral\RoadRunner\Tcp\TcpEvent;
use Spiral\RoadRunnerBridge\Tcp\Response\ContinueRead;
use Spiral\RoadRunnerBridge\Tcp\Response\ResponseInterface;
use Spiral\RoadRunnerBridge\Tcp\Service\ServiceInterface;
use Symfony\Component\VarDumper\Cloner\Data;

final readonly class Service implements ServiceInterface
{
    public function __construct(
        private CommandBusInterface $bus,
        private DumpIdGeneratorInterface $idGenerator,
        private int $previewMaxDepth = 3,
    ) {}

    public function handle(Request $request): ResponseInterface
    {
        if ($request->event === TcpEvent::Connected) {
            return new ContinueRead();
        }

        $messages = \array_filter(\explode("\n", $request->body));

        foreach ($messages as $message) {
            $payload = (new MessageParser())->parse($message);

            $this->fireEvent($payload, $message);
        }

        return new ContinueRead();
    }

    private function fireEvent(ParsedPayload $payload, string $message): void
    {
        $this->bus->dispatch(
            new HandleReceivedEvent(
                type: 'var-dump',
                payload: [
                    'payload' => $this->prepareContent($payload),
                    'message' => $message,
                    'context' => $payload->context,
                ],
                project: $payload->context['project'] ?? null,
            ),
        );
    }

    private function convertToPrimitive(Data $data): BodyInterface|null
    {
        if (\in_array($data->getType(), ['string', 'boolean'])) {
            return new PrimitiveBody(
                type: $data->getType(),
                value: $data->getValue(),
            );
        }

        $dumper = new HtmlDumper(
            generator: $this->idGenerator,
            maxDepth: $this->previewMaxDepth,
        );

        return new HtmlBody(
            value: $dumper->dump($data, true),
        );
    }

    private function prepareContent(ParsedPayload $payload): array
    {
        $payloadContent = [
            'type' => $payload->data->getType(),
            'value' => $this->convertToPrimitive($payload->data),
        ];

        if ($payload->data->getType() === 'string' && isset($payload->context['language'])) {
            $payloadContent['type'] = 'code';
            $payloadContent['language'] = $payload->context['language'];
        }

        return $payloadContent;
    }
}
