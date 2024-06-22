<?php

declare(strict_types=1);

namespace Modules\Sentry\Domain;

use App\Application\Domain\ValueObjects\Json;
use App\Application\Domain\ValueObjects\Uuid;
use Cycle\Annotated\Annotation\Column;
use Cycle\Annotated\Annotation\Entity;
use Cycle\Annotated\Annotation\Table\Index;
use Modules\Sentry\Application\DTO\Platform;
use Modules\Sentry\Domain\ValueObject\Sdk;

#[Entity(
    role: Trace::ROLE,
    repository: TraceRepositoryInterface::class,
    table: 'sentry_traces',
)]
#[Index(columns: ['trace_id'], unique: true)]
class Trace
{
    public const ROLE = 'sentry_trace';

    /**  @internal */
    public function __construct(
        #[Column(type: 'string(36)', primary: true, typecast: 'uuid')]
        private Uuid $uuid,
        #[Column(type: 'string(32)', name: 'trace_id')]
        private string $traceId,
        #[Column(type: 'string(255)', name: 'public_key')]
        private string $publicKey,
        #[Column(type: 'string(255)')]
        private string $environment,
        #[Column(type: 'boolean')]
        private bool $sampled,
        #[Column(type: 'float', name: 'sample_rate')]
        private float $sampleRate,
        #[Column(type: 'string', nullable: true, default: null)]
        private ?string $transaction,
        #[Column(type: 'jsonb', typecast: Sdk::class)]
        private Sdk $sdk,
        #[Column(type: 'string', typecast: Platform::class)]
        private Platform $language,
    ) {}

    public function getUuid(): Uuid
    {
        return $this->uuid;
    }

    public function getTraceId(): string
    {
        return $this->traceId;
    }

    public function getPublicKey(): string
    {
        return $this->publicKey;
    }

    public function getEnvironment(): string
    {
        return $this->environment;
    }

    public function isSampled(): bool
    {
        return $this->sampled;
    }

    public function getSampleRate(): float
    {
        return $this->sampleRate;
    }

    public function getTransaction(): string
    {
        return $this->transaction;
    }

    public function getSdk(): Json
    {
        return $this->sdk;
    }

    public function getLanguage(): string
    {
        return $this->language;
    }
}
