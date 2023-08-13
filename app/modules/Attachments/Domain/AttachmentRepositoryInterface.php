<?php

declare(strict_types=1);

namespace Modules\Attachments\Domain;

use Cycle\ORM\RepositoryInterface;

/**
 * @template TEntity of Attachment
 */
interface AttachmentRepositoryInterface extends RepositoryInterface
{
    public function store(Attachment $event): bool;

    public function deleteAll(array $scope = []): void;

    public function deleteByPK(string $uuid): bool;
}
