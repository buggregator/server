<?php

declare(strict_types=1);

namespace Modules\Inspector\Application\Mapper;

use App\Application\Event\AbstractEventTypeMapper;

final readonly class EventTypeMapper extends AbstractEventTypeMapper
{
    public function toPreview(string $type, array|\JsonSerializable $payload): array|\JsonSerializable
    {
        $data = $payload instanceof \JsonSerializable ? $payload->jsonSerialize() : $payload;

        $transaction = [];

        foreach ($data as $block) {
            if ($block['model'] === 'transaction') {
                $transaction = $block;
                break;
            }
        }

        return [
            $transaction,
        ];
    }
}
