<?php

declare(strict_types=1);

namespace Modules\Smtp\Application;

use App\Application\Event\EventTypeRegistryInterface;
use Modules\Smtp\Application\Storage\AttachmentStorage;
use Modules\Smtp\Application\Storage\EmailBodyStorage;
use Modules\Smtp\Domain\AttachmentFactoryInterface;
use Modules\Smtp\Domain\AttachmentRepositoryInterface;
use Modules\Smtp\Domain\AttachmentStorageInterface;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Cache\CacheStorageProviderInterface;
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
                bucket: $storage->bucket('attachments'),
                attachments: $attachments,
                factory: $factory,
            ),

            AttachmentFactoryInterface::class => AttachmentFactory::class,
        ];
    }

    public function boot(EventTypeRegistryInterface $registry): void
    {
        $registry->register('smtp', new Mapper\EventTypeMapper());
    }
}
