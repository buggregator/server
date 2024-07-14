<?php

declare(strict_types=1);

namespace Modules\Smtp\Application;

use Modules\Smtp\Application\Mapper\EventTypeMapper;
use App\Application\Event\EventTypeRegistryInterface;
use App\Application\Persistence\DriverEnum;
use Cycle\ORM\EntityManagerInterface;
use Cycle\ORM\ORMInterface;
use Cycle\ORM\Select;
use Modules\Smtp\Application\Storage\AttachmentStorage;
use Modules\Smtp\Application\Storage\EmailBodyStorage;
use Modules\Smtp\Domain\Attachment;
use Modules\Smtp\Domain\AttachmentFactoryInterface;
use Modules\Smtp\Domain\AttachmentRepositoryInterface;
use Modules\Smtp\Domain\AttachmentStorageInterface;
use Modules\Smtp\Integration\CycleOrm\AttachmentRepository;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Cache\CacheStorageProviderInterface;
use Spiral\Core\FactoryInterface;
use Spiral\Storage\StorageInterface;

final class SmtpBootloader extends Bootloader
{
    public function defineSingletons(): array
    {
        return [
            EmailBodyStorage::class => static fn(
                CacheStorageProviderInterface $provider,
            ) => new EmailBodyStorage(
                cache: $provider->storage('smtp'),
            ),

            AttachmentStorageInterface::class => static fn(
                StorageInterface $storage,
                AttachmentRepositoryInterface $attachments,
                AttachmentFactoryInterface $factory,
            ): AttachmentStorageInterface => new AttachmentStorage(
                bucket: $storage->bucket('smtp'),
                attachments: $attachments,
                factory: $factory,
            ),

            AttachmentFactoryInterface::class => AttachmentFactory::class,

            AttachmentRepository::class => static fn(
                ORMInterface $orm,
                EntityManagerInterface $manager,
            ): AttachmentRepositoryInterface => new AttachmentRepository(
                $manager,
                new Select($orm, Attachment::class),
            ),
            AttachmentRepositoryInterface::class => static fn(
                FactoryInterface $factory,
                DriverEnum $driver,
            ): AttachmentRepositoryInterface => match ($driver) {
                DriverEnum::Database => $factory->make(AttachmentRepository::class),
                default => throw new \Exception('Unsupported database driver'),
            },
        ];
    }

    public function boot(EventTypeRegistryInterface $registry): void
    {
        $registry->register('smtp', new EventTypeMapper());
    }
}
