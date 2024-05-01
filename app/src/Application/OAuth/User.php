<?php

declare(strict_types=1);

namespace App\Application\OAuth;

readonly class User implements \JsonSerializable
{
    public static function fromArray(array $data): self
    {
        return new self(
            provider: $data['provider'],
            username: $data['username'],
            avatar: $data['avatar'],
            email: $data['email'],
        );
    }

    public function __construct(
        public ?string $provider,
        public string $username,
        public string $avatar,
        public string $email,
    ) {}

    public function jsonSerialize(): array
    {
        return [
            'provider' => $this->provider,
            'username' => $this->username,
            'avatar' => $this->avatar,
            'email' => $this->email,
        ];
    }
}
