<?php

declare(strict_types=1);

namespace App\Integration\Kinde;

use Kinde\KindeSDK\Sdk\Enums\StorageEnums;
use Spiral\Session\SessionSectionInterface;

final readonly class SessionStorage
{
    public function __construct(
        private SessionSectionInterface $section,
    ) {}

    public function getToken($associative = true): array|object|null
    {
        $token = $this->get(StorageEnums::TOKEN);

        return empty($token) ? null : \json_decode($token, $associative);
    }

    public function setToken(string $token): void
    {
        $this->set(StorageEnums::TOKEN, $token);
    }

    public function getAccessToken(): ?string
    {
        $token = $this->getToken();
        return empty($token) ? null : $token['access_token'];
    }

    public function getIdToken(): ?string
    {
        $token = $this->getToken();
        return empty($token) ? null : $token['id_token'];
    }

    public function getRefreshToken(): ?string
    {
        $token = $this->getToken();

        return empty($token) ? null : $token['refresh_token'];
    }

    public function getState(): ?string
    {
        return $this->get(StorageEnums::STATE);
    }

    public function setState(string $newState): void
    {
        $this->set(StorageEnums::STATE, $newState);
    }

    public function getCodeVerifier(): ?string
    {
        return $this->get(StorageEnums::CODE_VERIFIER);
    }

    public function clear(): void
    {
        $this->purge();
    }

    public function removeItem(string $key): void
    {
        $this->delete($key);
    }

    private function delete(string $key): void
    {
        $this->section->delete($key);
    }

    private function get(string $key, mixed $default = null): mixed
    {
        return $this->section->get($key, $default);
    }

    private function purge(): void
    {
        $this->section->clear();
    }

    private function set(string $key, mixed $value): void
    {
        $this->section->set($key, $value);
    }
}
