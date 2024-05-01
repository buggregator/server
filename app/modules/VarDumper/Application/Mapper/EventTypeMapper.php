<?php

declare(strict_types=1);

namespace Modules\VarDumper\Application\Mapper;

use App\Application\Event\AbstractEventTypeMapper;
use Modules\VarDumper\Application\Dump\HtmlDumper;
use Modules\VarDumper\Application\Dump\MessageParser;

final readonly class EventTypeMapper extends AbstractEventTypeMapper
{
    public function __construct(
        private HtmlDumper $dumper,
        private MessageParser $parser,
    ) {}

    public function toPreview(string $type, array|\JsonSerializable $payload): array|\JsonSerializable
    {
        $data = $payload instanceof \JsonSerializable ? $payload->jsonSerialize() : $payload;

        return [
            'payload' => $data['payload'],
            'context' => $data['context'],
        ];
    }

    public function toFull(string $type, array|\JsonSerializable $payload): array|\JsonSerializable
    {
        $data = $payload instanceof \JsonSerializable ? $payload->jsonSerialize() : $payload;

        $parsed = $this->parser->parse($data['message']);

        return [
            'payload' => [
                'payload' => $this->dumper->dump($parsed->data),
                'type' => $parsed->data->getType(),
            ],
            'context' => $data['context'],
        ];
    }
}
