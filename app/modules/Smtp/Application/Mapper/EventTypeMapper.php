<?php

declare(strict_types=1);

namespace Modules\Smtp\Application\Mapper;

use App\Application\Event\EventTypeMapperInterface;

final readonly class EventTypeMapper implements EventTypeMapperInterface
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
