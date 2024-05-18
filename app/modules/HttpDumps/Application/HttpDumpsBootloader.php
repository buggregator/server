<?php

declare(strict_types=1);

namespace Modules\HttpDumps\Application;

use App\Application\Persistence\DriverEnum;
use Cycle\ORM\EntityManagerInterface;
use Cycle\ORM\ORMInterface;
use Cycle\ORM\Select;
use Modules\HttpDumps\EventHandler;
use Modules\HttpDumps\Application\Storage\AttachmentStorage;
use Modules\HttpDumps\Domain\AttachmentFactoryInterface;
use Modules\HttpDumps\Domain\AttachmentRepositoryInterface;
use Modules\HttpDumps\Domain\AttachmentStorageInterface;
use Modules\HttpDumps\Domain\Attachment;
use Modules\HttpDumps\Integration\CycleOrm\AttachmentRepository;
use Psr\Container\ContainerInterface;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Core\FactoryInterface;
use Spiral\Storage\StorageInterface;

final class HttpDumpsBootloader extends Bootloader
{
    public function defineSingletons(): array
    {
        return [
            AttachmentStorageInterface::class => static fn(
                StorageInterface $storage,
                AttachmentRepositoryInterface $attachments,
                AttachmentFactoryInterface $factory,
            ): AttachmentStorageInterface => new AttachmentStorage(
                bucket: $storage->bucket('http_dumps'),
                attachments: $attachments,
                factory: $factory,
            ),

            AttachmentFactoryInterface::class => AttachmentFactory::class,

            EventHandlerInterface::class => static fn(
                ContainerInterface $container,
            ): EventHandlerInterface => new EventHandler($container, []),

            // Persistence
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
}
