<?php

declare(strict_types=1);

namespace App\Integration\CycleOrm\Persistence;

use App\Application\Domain\ValueObjects\Uuid;
use Cycle\ORM\EntityManagerInterface;
use Cycle\ORM\Select;
use Cycle\ORM\Select\Repository;
use Modules\Smtp\Domain\Attachment;
use Modules\Smtp\Domain\AttachmentRepositoryInterface;

/**
 * @template TEntity of Attachment
 * @extends Repository<Attachment>
 */
final class CycleOrmAttachmentRepository extends Repository implements AttachmentRepositoryInterface
{
    public function __construct(
        private readonly EntityManagerInterface $em,
        Select $select,
    ) {
        parent::__construct($select);
    }

    public function store(Attachment $attachment): bool
    {
        $this->em->persist($attachment)->run();

        return true;
    }

    public function findByEvent(Uuid $uuid): iterable
    {
        return $this->select()->where('eventUuid', $uuid)->fetchAll();
    }

    public function deleteByEvent(Uuid $uuid): void
    {
        foreach ($this->findByEvent($uuid) as $attachment) {
            $this->em->delete($attachment);
        }

        $this->em->run();
    }
}
