<?php

declare(strict_types=1);

namespace Modules\Smtp\Application\Mapper;

use App\Application\Event\AbstractEventTypeMapper;

final readonly class EventTypeMapper extends AbstractEventTypeMapper
{
    public function toPreview(string $type, array|\JsonSerializable $payload): array|\JsonSerializable
    {
        $data = $payload instanceof \JsonSerializable ? $payload->jsonSerialize() : $payload;

        return [
            'subject' => $data['subject'],
            'from' => $data['from'],
            'to' => $data['to'],
        ];
    }
}
