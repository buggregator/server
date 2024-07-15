<?php

declare(strict_types=1);

namespace Modules\Sentry\Application\DTO;

use App\Application\Domain\ValueObjects\Uuid;

class Payload implements \JsonSerializable
{
    public readonly Uuid $uuid;
    private string $fingerprint;
    private bool $isExists = false;

    /**
     * @param PayloadChunkInterface[] $chunks
     */
    public function __construct(
        public readonly array $chunks,
    ) {
        $this->uuid = Uuid::generate();
    }

    public function withFingerprint(string $fingerprint): self
    {
        $self = clone $this;
        $self->fingerprint = $fingerprint;
        return $self;
    }

    public function markAsExists(): self
    {
        $self = clone $this;
        $self->isExists = true;

        return $self;
    }

    public function eventId(): string
    {
        return $this->getMeta()->eventId();
    }

    public function traceId(): string
    {
        return $this->getMeta()->traceId();
    }

    public function getMeta(): MetaChunk
    {
        if (isset($this->chunks[0]) && $this->chunks[0] instanceof MetaChunk) {
            return $this->chunks[0];
        }

        throw new \InvalidArgumentException('Meta chunk not found');
    }

    public function getPayload(): PayloadChunkInterface
    {
        return $this->chunks[2];
    }

    public function type(): Type
    {
        foreach ($this->chunks as $chunk) {
            if ($chunk instanceof TypeChunk) {
                return $chunk->type();
            }
        }

        throw new \InvalidArgumentException('Type chunk not found');
    }

    public function tags(): array
    {
        $serverName = $this->getPayload()['server_name'] ?? null;

        $tags = [
            'platform' => $this->getMeta()->platform()->name,
            'environment' => $this->getMeta()->environment(),
        ];

        if ($serverName !== null) {
            $tags['server_name'] = $serverName;
        }

        return $tags;
    }

    public function jsonSerialize(): array
    {
        return $this->chunks;
    }

    public function getFingerprint(): string
    {
        return $this->fingerprint;
    }

    public function isExists(): bool
    {
        return $this->isExists;
    }
}
