<?php

declare(strict_types=1);

namespace App\Integration\Kinde;

use Firebase\JWT\JWK;
use Firebase\JWT\JWT;

final readonly class JwtTokenParser
{
    public function __construct(
        private string $jwksUrl,
    ) {}

    public function parse(string $token): ?array
    {
        try {
            $jwksJson = \file_get_contents($this->jwksUrl);
            $jwks = \json_decode($jwksJson, true);

            return \json_decode(\json_encode(JWT::decode($token, JWK::parseKeySet($jwks))), true);
        } catch (\Exception) {
            return null;
        }
    }
}
