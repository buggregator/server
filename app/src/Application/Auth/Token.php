<?php

declare(strict_types=1);

namespace App\Application\Auth;

use Spiral\Auth\TokenInterface;

final readonly class Token implements TokenInterface
{
    public function __construct(
        private string $id,
        private array $token,
        private array $payload,
        private \DateTimeImmutable $issuedAt,
        private \DateTimeImmutable $expiresAt,
    ) {}

    public function getID(): string
    {
        return $this->id;
    }

    public function getToken(): array
    {
        return $this->token;
    }

    public function getPayload(): array
    {
        return $this->payload;
    }

    public function getIssuedAt(): \DateTimeImmutable
    {
        return $this->issuedAt;
    }

    public function getExpiresAt(): \DateTimeInterface
    {
        return $this->expiresAt;
    }
}
