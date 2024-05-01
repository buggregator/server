<?php

declare(strict_types=1);

namespace Modules\Profiler\Application\Mapper;

use App\Application\Event\AbstractEventTypeMapper;

final readonly class EventTypeMapper extends AbstractEventTypeMapper
{
    public function toPreview(string $type, array|\JsonSerializable $payload): array|\JsonSerializable
    {
        $data = $payload instanceof \JsonSerializable ? $payload->jsonSerialize() : $payload;

        return [
            'peaks' => $data['peaks'],
            'tags' => $data['tags'],
            'app_name' => $data['app_name'],
            'hostname' => $data['hostname'],
            'date' => $data['date'],
        ];
    }
}
