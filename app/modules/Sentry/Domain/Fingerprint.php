<?php

declare(strict_types=1);

namespace Modules\Sentry\Domain;

use App\Application\Domain\ValueObjects\Uuid;
use Cycle\Annotated\Annotation\Column;
use Cycle\Annotated\Annotation\Entity;
use Cycle\Annotated\Annotation\ForeignKey;
use Cycle\Annotated\Annotation\Relation\BelongsTo;
use Cycle\Annotated\Annotation\Relation\HasOne;
use Cycle\Annotated\Annotation\Table\Index;

#[Entity(
    role: Fingerprint::ROLE,
    repository: FingerprintRepositoryInterface::class,
    table: 'sentry_issue_fingerprints',
)]
#[Index(columns: ['issue_uuid', 'fingerprint'], unique: true)]
#[ForeignKey(target: Issue::class, innerKey: 'issue_uuid', outerKey: 'uuid')]
class Fingerprint
{
    public const ROLE = 'sentry_issue_fingerprint';

    #[Column(type: 'datetime', name: 'created_at')]
    private \DateTimeInterface $createdAt;

    /**  @internal */
    public function __construct(
        #[Column(type: 'string(36)', primary: true, typecast: 'uuid')]
        private Uuid $uuid,
        #[Column(type: 'string(36)', name: 'issue_uuid', typecast: 'uuid')]
        private Uuid $issueUuid,
        #[Column(type: 'string(50)')]
        private string $fingerprint,
    ) {
        $this->createdAt = new \DateTimeImmutable();
    }

    public function getIssueUuid(): Uuid
    {
        return $this->issueUuid;
    }

    public function getFingerprint(): string
    {
        return $this->fingerprint;
    }

    public function getCreatedAt(): \DateTimeInterface
    {
        return $this->createdAt;
    }
}
