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

    public function toSearchableText(string $type, array|\JsonSerializable $payload): string
    {
        $data = $payload instanceof \JsonSerializable ? $payload->jsonSerialize() : $payload;

        $parts = \array_filter([
            $data['subject'] ?? null,
        ]);

        foreach (($data['from'] ?? []) as $address) {
            $parts[] = \is_array($address) ? ($address['email'] ?? $address['address'] ?? '') : (string) $address;
        }

        foreach (($data['to'] ?? []) as $address) {
            $parts[] = \is_array($address) ? ($address['email'] ?? $address['address'] ?? '') : (string) $address;
        }

        return \implode(' ', $parts);
    }
}
