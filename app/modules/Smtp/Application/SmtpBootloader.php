<?php

declare(strict_types=1);

namespace Modules\Smtp\Application;

use App\Application\Event\EventTypeRegistryInterface;
use Modules\Smtp\Application\Storage\EmailBodyStorage;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Cache\CacheStorageProviderInterface;

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
        ];
    }

    public function boot(EventTypeRegistryInterface $registry): void
    {
        $registry->register('smtp', new Mapper\EventTypeMapper());
    }
}
