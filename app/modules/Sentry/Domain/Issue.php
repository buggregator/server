<?php

declare(strict_types=1);

namespace Modules\Sentry\Domain;

use App\Application\Domain\ValueObjects\Json;
use App\Application\Domain\ValueObjects\Uuid;
use Cycle\Annotated\Annotation\Column;
use Cycle\Annotated\Annotation\Entity;
use Cycle\Annotated\Annotation\ForeignKey;
use Cycle\Annotated\Annotation\Relation\HasMany;
use Cycle\Annotated\Annotation\Relation\HasOne;
use Cycle\Annotated\Annotation\Relation\ManyToMany;
use Doctrine\Common\Collections\ArrayCollection;
use Modules\Sentry\Domain\ValueObject\Sdk;

#[Entity(
    role: Issue::ROLE,
    repository: IssueRepositoryInterface::class,
    table: 'sentry_issues',
)]
#[ForeignKey(target: Trace::class, innerKey: 'trace_uuid', outerKey: 'uuid')]
class Issue
{
    public const ROLE = 'sentry_issue';

    #[Column(type: 'datetime')]
    private \DateTimeInterface $createdAt;

    #[HasOne(
        target: Fingerprint::class,
        innerKey: 'uuid',
        outerKey: 'issue_uuid'
    )]
    private Fingerprint $fingerprint;

    #[HasMany(
        target: IssueTag::class,
        innerKey: 'uuid',
        outerKey: 'issue_uuid',
    )]
    private ArrayCollection $tags;

    /**  @internal */
    public function __construct(
        #[Column(type: 'string(36)', primary: true, typecast: 'uuid')]
        private Uuid $uuid,
        #[Column(type: 'string(36)', name: 'trace_uuid', typecast: 'uuid')]
        private Uuid $traceUuid,
        #[Column(type: 'text')]
        private string $title,
        #[Column(type: 'string(32)')]
        private string $platform,
        #[Column(type: 'string(32)')]
        private string $logger,
        #[Column(type: 'string(32)')]
        private string $type,
        #[Column(type: 'string', nullable: true, default: null)]
        private ?string $transaction,
        #[Column(type: 'string', name: 'server_name')]
        private string $serverName,
        #[Column(type: 'jsonb', typecast: Json::class)]
        private Json $payload,
    ) {
        $this->tags = new ArrayCollection();
        $this->createdAt = new \DateTimeImmutable();
    }

    public function getUuid(): Uuid
    {
        return $this->uuid;
    }

    public function getTraceUuid(): Uuid
    {
        return $this->traceUuid;
    }

    public function getTitle(): string
    {
        return $this->title;
    }

    public function getPlatform(): string
    {
        return $this->platform;
    }

    public function getLogger(): string
    {
        return $this->logger;
    }

    public function getType(): string
    {
        return $this->type;
    }

    public function getSdk(): Sdk
    {
        return $this->sdk;
    }

    public function getTransaction(): string
    {
        return $this->transaction;
    }

    public function getServerName(): string
    {
        return $this->serverName;
    }

    public function getPayload(): Json
    {
        return $this->payload;
    }

    public function getCreatedAt(): \DateTimeInterface
    {
        return $this->createdAt;
    }

    public function getTags(): ArrayCollection
    {
        return $this->tags;
    }
}
