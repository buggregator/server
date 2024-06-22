<?php

declare(strict_types=1);

namespace Modules\Sentry\Domain;

use App\Application\Domain\ValueObjects\Uuid;
use Cycle\Annotated\Annotation\Column;
use Cycle\Annotated\Annotation\Entity;
use Cycle\Annotated\Annotation\ForeignKey;

#[Entity(
    role: IssueTag::ROLE,
    repository: IssueTagRepositoryInterface::class,
    table: 'sentry_issue_tag',
)]
#[ForeignKey(target: Issue::class, innerKey: 'issue_uuid', outerKey: 'uuid')]
class IssueTag
{
    public const ROLE = 'sentry_issue_tag';

    /**  @internal */
    public function __construct(
        #[Column(type: 'string(36)', name: 'issue_uuid', primary: true, typecast: 'uuid')]
        private Uuid $issueUuid,
        #[Column(type: 'string', primary: true)]
        private string $tag,
        #[Column(type: 'string')]
        private string $value,
    ) {}

    public function getIssueUuid(): Uuid
    {
        return $this->issueUuid;
    }

    public function getTag(): string
    {
        return $this->tag;
    }

    public function getValue(): string
    {
        return $this->value;
    }
}
