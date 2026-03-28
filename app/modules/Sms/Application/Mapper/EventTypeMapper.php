<?php

declare(strict_types=1);

namespace Modules\Sms\Application\Mapper;

use App\Application\Event\EventTypeMapperInterface;

final readonly class EventTypeMapper implements EventTypeMapperInterface
{
    public function toPreview(string $type, array|\JsonSerializable $payload): array|\JsonSerializable
    {
        $data = $payload instanceof \JsonSerializable ? $payload->jsonSerialize() : $payload;

        $preview = [
            'from' => $data['from'] ?? '',
            'to' => $data['to'] ?? '',
            'message' => $data['message'] ?? '',
            'gateway' => $data['gateway'] ?? '',
        ];

        if (!empty($data['warnings'])) {
            $preview['warnings'] = $data['warnings'];
        }

        return $preview;
    }

    public function toSearchableText(string $type, array|\JsonSerializable $payload): string
    {
        $data = $payload instanceof \JsonSerializable ? $payload->jsonSerialize() : $payload;

        return \implode(' ', \array_filter([
            $data['from'] ?? null,
            $data['to'] ?? null,
            $data['message'] ?? null,
        ]));
    }
}
