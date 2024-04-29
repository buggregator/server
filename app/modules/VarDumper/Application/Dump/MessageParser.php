<?php

declare(strict_types=1);

namespace Modules\VarDumper\Application\Dump;

use Modules\VarDumper\Exception\InvalidPayloadException;
use Symfony\Component\VarDumper\Cloner\Data;
use Symfony\Component\VarDumper\Cloner\Stub;

final class MessageParser
{
    /**
     * @throws \RuntimeException
     */
    public function parse(string $message): ParsedPayload
    {
        $payload = @\unserialize(\base64_decode($message), ['allowed_classes' => [Data::class, Stub::class]]);

        // Impossible to decode the message, give up.
        if (false === $payload) {
            throw new InvalidPayloadException("Unable to decode the message.");
        }

        if (
            !\is_array($payload)
            || \count($payload) < 2
            || !$payload[0] instanceof Data
            || !\is_array($payload[1])
        ) {
            throw new InvalidPayloadException("Invalid payload structure.");
        }

        return new ParsedPayload(data: $payload[0], context: $payload[1]);
    }
}
