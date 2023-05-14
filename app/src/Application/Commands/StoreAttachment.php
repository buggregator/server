<?php

declare(strict_types=1);

namespace App\Application\Commands;

use Psr\Http\Message\StreamInterface;
use Spiral\Cqrs\CommandInterface;

final class StoreAttachment implements CommandInterface
{
    public function __construct(
        public readonly string $type,
        public readonly string $filename,
        public readonly string|StreamInterface $content,
    ) {
    }
}
