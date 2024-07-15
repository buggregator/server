<?php

declare(strict_types=1);

namespace Modules\Sentry\Domain;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Sentry\Application\DTO\MetaChunk;

interface TraceFactoryInterface
{
    public function createFromMeta(
        Uuid $uuid,
        MetaChunk $meta,
    ): Trace;
}
