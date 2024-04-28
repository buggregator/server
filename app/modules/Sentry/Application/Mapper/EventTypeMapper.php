<?php

declare(strict_types=1);

namespace Modules\Sentry\Application\Mapper;

use App\Application\Event\EventTypeMapperInterface;

final readonly class EventTypeMapper implements EventTypeMapperInterface
{
    public function toPreview(string $type, array|\JsonSerializable $payload): array|\JsonSerializable
    {
        $data = $payload instanceof \JsonSerializable ? $payload->jsonSerialize() : $payload;

        return [
            'exception' => $data['exception'],
            'platform' => $data['platform'],
            'environment' => $data['environment'],
            'server_name' => $data['server_name'],
            'event_id' => $data['event_id'] ?? null,
        ];
    }
}
