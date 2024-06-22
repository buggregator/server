<?php

declare(strict_types=1);

namespace Modules\Sentry\Application;

use App\Application\Event\EventTypeRegistryInterface;
use Cycle\Database\DatabaseInterface;
use Cycle\ORM\ORMInterface;
use Cycle\ORM\Select;
use Modules\Sentry\Application\Handlers\StoreEventHandler;
use Modules\Sentry\Application\Handlers\StoreTraceHandler;
use Modules\Sentry\Domain\Fingerprint;
use Modules\Sentry\Domain\FingerprintFactoryInterface;
use Modules\Sentry\Domain\FingerprintRepositoryInterface;
use Modules\Sentry\Domain\Issue;
use Modules\Sentry\Domain\IssueFactoryInterface;
use Modules\Sentry\Domain\IssueRepositoryInterface;
use Modules\Sentry\Domain\IssueTag;
use Modules\Sentry\Domain\IssueTagRepositoryInterface;
use Modules\Sentry\Domain\Trace;
use Modules\Sentry\Domain\TraceFactoryInterface;
use Modules\Sentry\Domain\TraceRepositoryInterface;
use Modules\Sentry\EventHandler;
use Modules\Sentry\Integration\CycleOrm\FingerprintFactory;
use Modules\Sentry\Integration\CycleOrm\FingerprintRepository;
use Modules\Sentry\Integration\CycleOrm\IssueFactory;
use Modules\Sentry\Integration\CycleOrm\IssueRepository;
use Modules\Sentry\Integration\CycleOrm\IssueTagRepository;
use Modules\Sentry\Integration\CycleOrm\TraceFactory;
use Modules\Sentry\Integration\CycleOrm\TraceRepository;
use Psr\Container\ContainerInterface;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Boot\EnvironmentInterface;

final class SentryBootloader extends Bootloader
{
    public function defineSingletons(): array
    {
        return [
            SecretKeyValidator::class => static fn(
                EnvironmentInterface $env,
            ): SecretKeyValidator => new SecretKeyValidator(
                secret: $env->get('SENTRY_SECRET_KEY'),
            ),

            EventHandlerInterface::class => static fn(
                ContainerInterface $container,
            ): EventHandlerInterface => new EventHandler($container, [
                StoreTraceHandler::class,
                StoreEventHandler::class,
            ]),

            // Persistence
            IssueTagRepositoryInterface::class => static fn(
                ORMInterface $orm,
            ): IssueTagRepositoryInterface => new IssueTagRepository(new Select($orm, IssueTag::class)),
            FingerprintRepositoryInterface::class => static fn(
                ORMInterface $orm,
                DatabaseInterface $database,
            ): FingerprintRepositoryInterface => new FingerprintRepository(
                new Select($orm, Fingerprint::class),
                $database,
            ),
            IssueRepositoryInterface::class => static fn(
                ORMInterface $orm,
            ): IssueRepositoryInterface => new IssueRepository(new Select($orm, Issue::class)),
            TraceRepositoryInterface::class => static fn(
                ORMInterface $orm,
            ): TraceRepositoryInterface => new TraceRepository(new Select($orm, Trace::class)),

            TraceFactoryInterface::class => TraceFactory::class,
            IssueFactoryInterface::class => IssueFactory::class,
            FingerprintFactoryInterface::class => FingerprintFactory::class,
        ];
    }

    public function boot(EventTypeRegistryInterface $registry): void
    {
        $registry->register('sentry', new Mapper\EventTypeMapper());
    }
}
