<?php

declare(strict_types=1);

namespace Modules\Sentry\Application\DTO;

use App\Application\Domain\ValueObjects\Uuid;

final readonly class MetaChunk extends JsonChunk
{
    public function eventId(): string
    {
        return $this->data['event_id'] ?? (string) Uuid::generate();
    }

    public function traceId(): string
    {
        return $this->data['trace']['trace_id'] ?? (string) Uuid::generate();
    }

    public function publicKey(): string
    {
        return $this->data['trace']['public_key'] ?? '';
    }

    public function environment(): string
    {
        return $this->data['trace']['environment'] ?? '';
    }

    public function platform(): Platform
    {
        $sdk = $this->data['sdk'];

        return Platform::detect($sdk['name']);
    }

    public function sampled(): bool
    {
        if (!isset($this->data['trace']['sampled'])) {
            return false;
        }

        $value = $this->data['trace']['sampled'];

        if (\is_bool($value)) {
            return $value;
        }

        if (\is_string($value)) {
            return $value === 'true';
        }

        return false;
    }

    public function sampleRate(): float
    {
        return (float) ($this->data['trace']['sample_rate'] ?? 0.0);
    }

    public function transaction(): ?string
    {
        return $this->data['trace']['transaction'] ?? null;
    }

    public function sdk(): array
    {
        return $this->data['sdk'] ?? [];
    }
}
