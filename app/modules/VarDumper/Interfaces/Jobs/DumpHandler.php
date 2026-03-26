<?php

declare(strict_types=1);

namespace Modules\VarDumper\Interfaces\Jobs;

use App\Application\Commands\HandleReceivedEvent;
use Modules\VarDumper\Application\Dump\BodyInterface;
use Modules\VarDumper\Application\Dump\DumpIdGeneratorInterface;
use Modules\VarDumper\Application\Dump\HtmlBody;
use Modules\VarDumper\Application\Dump\HtmlDumper;
use Modules\VarDumper\Application\Dump\MessageParser;
use Modules\VarDumper\Application\Dump\ParsedPayload;
use Modules\VarDumper\Application\Dump\PrimitiveBody;
use Spiral\Core\InvokerInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Queue\JobHandler;
use Symfony\Component\VarDumper\Cloner\Data;

final class DumpHandler extends JobHandler
{
    public function __construct(
        private readonly CommandBusInterface $commandBus,
        private readonly DumpIdGeneratorInterface $dumpId,
        InvokerInterface $invoker,
    ) {
        parent::__construct($invoker);
    }

    public function invoke(mixed $payload): void
    {
        if (\is_string($payload)) {
            $decoded = \json_decode($payload, true);
            // If JSON with 'payload' key — extract the base64 message from the Go plugin envelope
            if (\is_array($decoded) && isset($decoded['payload'])) {
                $message = $decoded['payload'];
            } else {
                // Raw base64 string (Spiral deserialized the payload field directly)
                $message = $payload;
            }
        } elseif (\is_array($payload)) {
            $message = $payload['payload'] ?? '';
        } else {
            return;
        }

        if ($message === '') {
            return;
        }

        $parsed = (new MessageParser())->parse($message);

        $this->fireEvent($parsed);
    }

    private function fireEvent(ParsedPayload $payload): void
    {
        $this->commandBus->dispatch(
            new HandleReceivedEvent(
                type: 'var-dump',
                payload: [
                    'payload' => $this->prepareContent($payload),
                    'context' => $payload->context,
                ],
                project: $payload->data->getContext()['project'] ?? null,
            ),
        );
    }

    private function convertToPrimitive(Data $data): BodyInterface
    {
        if (\in_array($data->getType(), ['string', 'boolean', 'integer', 'double'])) {
            return new PrimitiveBody(
                type: $data->getType(),
                value: $data->getValue(),
            );
        }

        $dumper = new HtmlDumper(
            generator: $this->dumpId,
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
            'label' => $payload->data->getContext()['label'] ?? null,
        ];

        $language = $payload->data->getContext()['language'] ?? null;

        if (
            $payload->data->getType() === 'string'
            && $language !== null
        ) {
            $payloadContent['type'] = 'code';
            $payloadContent['language'] = $language;
        }

        return $payloadContent;
    }
}
