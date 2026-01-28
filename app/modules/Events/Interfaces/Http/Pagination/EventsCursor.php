<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Pagination;

use Modules\Events\Domain\Event;
use Ramsey\Uuid\Uuid;
use Spiral\Filters\Exception\ValidationException;

final readonly class EventsCursor
{
    public function __construct(
        private string $timestamp,
        private string $uuid,
    ) {}

    public static function fromEvent(Event $event): self
    {
        return new self(
            (string) $event->getTimestamp(),
            (string) $event->getUuid(),
        );
    }

    public static function fromOpaque(string $cursor): self
    {
        $decoded = self::decode($cursor);
        $payload = \json_decode($decoded, true);

        if (!\is_array($payload)) {
            throw self::invalid('Invalid cursor payload.');
        }

        $timestamp = $payload['ts'] ?? null;
        $uuid = $payload['uuid'] ?? null;

        if (!\is_string($timestamp) || !self::isValidTimestamp($timestamp)) {
            throw self::invalid('Invalid cursor timestamp.');
        }

        if (!\is_string($uuid) || !Uuid::isValid($uuid)) {
            throw self::invalid('Invalid cursor UUID.');
        }

        return new self($timestamp, $uuid);
    }

    public function toOpaque(): string
    {
        $payload = \json_encode(
            [
                'ts' => $this->timestamp,
                'uuid' => $this->uuid,
            ],
            \JSON_UNESCAPED_SLASHES,
        );

        if ($payload === false) {
            throw self::invalid('Unable to encode cursor.');
        }

        return self::encode($payload);
    }

    public function getTimestamp(): string
    {
        return $this->timestamp;
    }

    public function getUuid(): string
    {
        return $this->uuid;
    }

    private static function encode(string $value): string
    {
        return \rtrim(\strtr(\base64_encode($value), '+/', '-_'), '=');
    }

    private static function decode(string $value): string
    {
        $decoded = \base64_decode(self::padBase64Url($value), true);
        if ($decoded === false) {
            throw self::invalid('Invalid cursor encoding.');
        }

        return $decoded;
    }

    private static function padBase64Url(string $value): string
    {
        $value = \strtr($value, '-_', '+/');
        $padding = \strlen($value) % 4;

        return $padding === 0 ? $value : $value . \str_repeat('=', 4 - $padding);
    }

    private static function isValidTimestamp(string $value): bool
    {
        return (bool) \preg_match('/^\d+(?:\.\d+)?$/', $value);
    }

    private static function invalid(string $message): ValidationException
    {
        return new ValidationException([
            'cursor' => [$message],
        ], 'Invalid cursor.');
    }
}
