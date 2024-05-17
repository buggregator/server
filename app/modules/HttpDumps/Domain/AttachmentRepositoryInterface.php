<?php

declare(strict_types=1);

namespace Modules\HttpDumps\Domain;

use App\Application\Domain\ValueObjects\Uuid;
use Cycle\ORM\RepositoryInterface;

/**
 * @extends RepositoryInterface<Attachment>
 */
interface AttachmentRepositoryInterface extends RepositoryInterface
{
    public function store(Attachment $attachment): bool;

    /**
     * @return iterable<Attachment>
     */
    public function findByEvent(Uuid $uuid): iterable;

    public function deleteByEvent(Uuid $uuid): void;
}
