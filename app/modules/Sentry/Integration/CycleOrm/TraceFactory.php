<?php

declare(strict_types=1);

namespace Modules\Sentry\Integration\CycleOrm;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Sentry\Application\DTO\MetaChunk;
use Modules\Sentry\Domain\Trace;
use Modules\Sentry\Domain\TraceFactoryInterface;
use Modules\Sentry\Domain\ValueObject\Sdk;

final class TraceFactory implements TraceFactoryInterface
{
    public function createFromMeta(Uuid $uuid, MetaChunk $meta): Trace
    {
        return new Trace(
            uuid: $uuid,
            traceId: $meta->traceId(),
            publicKey: $meta->publicKey(),
            environment: $meta->environment(),
            sampled: $meta->sampled(),
            sampleRate: $meta->sampleRate(),
            transaction: $meta->transaction(),
            sdk: new Sdk($meta->sdk()),
            language: $meta->platform(),
        );
    }
}
