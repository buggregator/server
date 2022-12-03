<?php

declare(strict_types=1);

namespace Modules\VarDumper\Application\Dump;

use Symfony\Component\VarDumper\Cloner\Data;
use Symfony\Component\VarDumper\Cloner\Stub;

final class MessageParser
{
    /**
     * @throws \RuntimeException
     */
    public function parse(string $message): array
    {
        $payload = @\unserialize(\base64_decode($message), ['allowed_classes' => [Data::class, Stub::class]]);

        // Impossible to decode the message, give up.
        if (false === $payload) {
            throw new \RuntimeException("Unable to decode a message from var-dumper client.");
        }

        if (
            !\is_array($payload)
            || \count($payload) < 2
            || !$payload[0] instanceof Data
            || !\is_array($payload[1])
        ) {
            throw new \RuntimeException("Invalid var-dumper payload.");
        }

        return $payload;
    }
}
