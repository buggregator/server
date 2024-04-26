<?php

declare(strict_types=1);

namespace App\Application\Commands;

use App\Application\Domain\ValueObjects\Uuid;
use Spiral\Cqrs\CommandInterface;

final readonly class HandleReceivedEvent implements CommandInterface, \JsonSerializable
{
    public Uuid $uuid;
    public float $timestamp;

    public function __construct(
        public string $type,
        public array $payload,
        public ?string $project = null,
        ?Uuid $uuid = null,
    ) {
        $this->uuid = $uuid ?? Uuid::generate();
        $this->timestamp = \microtime(true);
    }

    public function jsonSerialize(): array
    {
        return [
            'project' => $this->project,
            'type' => $this->type,
            'payload' => $this->payload,
            'uuid' => (string) $this->uuid,
            'timestamp' => $this->timestamp,
        ];
    }
}
