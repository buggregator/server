<?php

declare(strict_types=1);

namespace App\Application\Auth;

use Spiral\Auth\TokenInterface;

final class Token implements TokenInterface
{
    public function __construct(
        private readonly string $id,
        private readonly array $token,
        private readonly array $payload,
        private readonly \DateTimeImmutable $issuedAt,
        private readonly \DateTimeImmutable $expiresAt,
    ) {
    }

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
