<?php

declare(strict_types=1);

namespace App\Integration\MongoDb\Persistence;

use App\Application\Domain\ValueObjects\Uuid;
use App\Integration\MongoDb\Mappers\SmtpAttachmentMapper;
use Modules\Smtp\Domain\Attachment;
use Modules\Smtp\Domain\AttachmentRepositoryInterface;
use MongoDB\Collection;
use MongoDB\Model\BSONDocument;

/**
 * todo: cover with tests
 */
final readonly class MongoDBSmtpAttachmentRepository implements AttachmentRepositoryInterface
{
    use HelpersTrait;

    public function __construct(
        private Collection $collection,
        private SmtpAttachmentMapper $mapper,
    ) {}

    public function store(Attachment $attachment): bool
    {
        $result = $this->collection->insertOne($this->mapper->toDocument($attachment));

        return $result->getInsertedCount() > 0;
    }

    public function findByEvent(Uuid $uuid): iterable
    {
        $cursor = $this->collection->find([
            SmtpAttachmentMapper::EVENT_UUID => (string) $uuid,
        ]);

        foreach ($cursor as $document) {
            yield $this->mapper->toAttachment($document);
        }
    }

    public function deleteByEvent(Uuid $uuid): void
    {
        $this->collection->deleteMany([SmtpAttachmentMapper::EVENT_UUID => (string) $uuid]);
    }

    public function findByPK(mixed $id): ?object
    {
        return $this->findOne(['_id' => $id]);
    }

    /**
     * @return Attachment|null
     */
    public function findOne(array $scope = []): ?object
    {
        /** @var BSONDocument|null $document */
        $document = $this->collection->findOne($scope);

        if ($document === null) {
            return null;
        }

        return $this->mapper->toAttachment($document);
    }

    public function findAll(array $scope = [], array $orderBy = [], int $limit = 30, int $offset = 0): iterable
    {
        $cursor = $this->collection->find(
            $scope,
            $this->makeOptions(
                orderBy: $orderBy,
                limit: $limit,
                offset: $offset,
            ),
        );

        foreach ($cursor as $document) {
            yield $this->mapper->toAttachment($document);
        }
    }
}
