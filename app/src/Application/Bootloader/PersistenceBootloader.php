<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use App\Application\Persistence\DriverEnum;
use App\Integration\CycleOrm\Persistence\CycleOrmAttachmentRepository;
use App\Integration\CycleOrm\Persistence\CycleOrmEventRepository;
use App\Integration\CycleOrm\Persistence\CycleOrmProjectRepository;
use App\Integration\CycleOrm\Persistence\CycleOrmWebhookDeliveryRepository;
use App\Integration\CycleOrm\Persistence\CycleOrmWebhookRepository;
use App\Integration\CycleOrm\Webhooks\CycleOrmWebhookRegistry;
use App\Integration\MongoDb\Persistence\MongoDBSmtpAttachmentRepository;
use App\Integration\MongoDb\Persistence\MongoDBEventRepository;
use App\Integration\MongoDb\Persistence\MongoDBProjectRepository;
use App\Interfaces\Console\RegisterModulesCommand;
use Cycle\ORM\EntityManagerInterface;
use Cycle\ORM\ORMInterface;
use Cycle\ORM\Select;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\EventRepositoryInterface;
use Modules\Projects\Domain\Project;
use Modules\Projects\Domain\ProjectRepositoryInterface;
use Modules\Smtp\Domain\Attachment;
use Modules\Smtp\Domain\AttachmentRepositoryInterface;
use Modules\Webhooks\Application\Locator\WebhookRegistryInterface;
use Modules\Webhooks\Domain as Webhooks;
use MongoDB\Database;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Console\Bootloader\ConsoleBootloader;
use Spiral\Core\FactoryInterface;
use Spiral\Cycle\Bootloader as CycleBridge;

final class PersistenceBootloader extends Bootloader
{
    public function defineDependencies(): array
    {
        return [
            // Databases
            CycleBridge\DatabaseBootloader::class,
            CycleBridge\MigrationsBootloader::class,

            // ORM
            CycleBridge\SchemaBootloader::class,
            CycleBridge\CycleOrmBootloader::class,
            CycleBridge\AnnotatedBootloader::class,
        ];
    }

    public function defineSingletons(): array
    {
        return [
            // Events
            EventRepositoryInterface::class => static fn(
                FactoryInterface $factory,
                DriverEnum $driver,
            ): EventRepositoryInterface => match ($driver) {
                DriverEnum::Database => $factory->make(CycleOrmEventRepository::class),
                DriverEnum::MongoDb => $factory->make(MongoDBEventRepository::class),
            },
            CycleOrmEventRepository::class => static fn(
                ORMInterface $orm,
                EntityManagerInterface $manager,
            ): CycleOrmEventRepository => new CycleOrmEventRepository($manager, new Select($orm, Event::class)),
            MongoDBEventRepository::class => static fn(
                Database $database,
                FactoryInterface $factory,
            ): MongoDBEventRepository => $factory->make(MongoDBEventRepository::class, [
                'collection' => $database->selectCollection('events'),
            ]),

            // Projects
            ProjectRepositoryInterface::class => static fn(
                FactoryInterface $factory,
                DriverEnum $driver,
            ): ProjectRepositoryInterface => match ($driver) {
                DriverEnum::Database => $factory->make(CycleOrmProjectRepository::class),
                DriverEnum::MongoDb => $factory->make(MongoDBProjectRepository::class),
            },
            CycleOrmProjectRepository::class => static fn(
                ORMInterface $orm,
                EntityManagerInterface $manager,
            ): ProjectRepositoryInterface => new CycleOrmProjectRepository($manager, new Select($orm, Project::class)),
            MongoDBProjectRepository::class => static fn(
                Database $database,
                FactoryInterface $factory,
            ): ProjectRepositoryInterface => $factory->make(MongoDBProjectRepository::class, [
                'collection' => $database->selectCollection('projects'),
            ]),

            // SMTP
            CycleOrmAttachmentRepository::class => static fn(
                ORMInterface $orm,
                EntityManagerInterface $manager,
            ): AttachmentRepositoryInterface => new CycleOrmAttachmentRepository(
                $manager,
                new Select($orm, Attachment::class),
            ),
            AttachmentRepositoryInterface::class => static fn(
                FactoryInterface $factory,
                DriverEnum $driver,
            ): AttachmentRepositoryInterface => match ($driver) {
                DriverEnum::Database => $factory->make(CycleOrmAttachmentRepository::class),
                DriverEnum::MongoDb => $factory->make(MongoDBSmtpAttachmentRepository::class),
            },
            MongoDBSmtpAttachmentRepository::class => static fn(
                Database $database,
                FactoryInterface $factory,
            ): AttachmentRepositoryInterface => $factory->make(MongoDBSmtpAttachmentRepository::class, [
                'collection' => $database->selectCollection('smtp_attachments'),
            ]),

            // Webhooks
            CycleOrmWebhookRepository::class => static fn(
                ORMInterface $orm,
                EntityManagerInterface $manager,
            ): Webhooks\WebhookRepositoryInterface => new CycleOrmWebhookRepository(
                $manager,
                new Select($orm, Webhooks\Webhook::class),
            ),
            CycleOrmWebhookDeliveryRepository::class => static fn(
                ORMInterface $orm,
                EntityManagerInterface $manager,
            ): Webhooks\DeliveryRepositoryInterface => new CycleOrmWebhookDeliveryRepository(
                $manager,
                new Select($orm, Webhooks\Delivery::class),
            ),
            Webhooks\WebhookRepositoryInterface::class => static fn(
                FactoryInterface $factory,
                DriverEnum $driver,
            ): Webhooks\WebhookRepositoryInterface => match ($driver) {
                DriverEnum::Database => $factory->make(CycleOrmWebhookRepository::class),
                default => throw new \RuntimeException('Unsupported driver'),
            },

            Webhooks\DeliveryRepositoryInterface::class => static fn(
                FactoryInterface $factory,
                DriverEnum $driver,
            ): Webhooks\DeliveryRepositoryInterface => match ($driver) {
                DriverEnum::Database => $factory->make(CycleOrmWebhookDeliveryRepository::class),
                default => throw new \RuntimeException('Unsupported driver'),
            },

            WebhookRegistryInterface::class => CycleOrmWebhookRegistry::class,
        ];
    }

    public function init(ConsoleBootloader $console, DriverEnum $driver): void
    {
        if ($driver === DriverEnum::Database) {
            $console->addSequence(
                name: RegisterModulesCommand::SEQUENCE,
                sequence: 'migrate',
                header: 'Migration',
            );
        }
    }
}
