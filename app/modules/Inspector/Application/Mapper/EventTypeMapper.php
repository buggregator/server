<?php

declare(strict_types=1);

namespace Modules\Inspector\Application\Mapper;

use App\Application\Event\EventTypeMapperInterface;

final readonly class EventTypeMapper implements EventTypeMapperInterface
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
