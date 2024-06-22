<?php

declare(strict_types=1);

namespace Modules\Sentry\Domain;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Sentry\Application\DTO\JsonChunk;

interface IssueFactoryInterface
{
    public function createFromPayload(
        Uuid $traceUuid,
        JsonChunk $payload,
    ): Issue;
}
