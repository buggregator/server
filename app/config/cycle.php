<?php

declare(strict_types=1);
use Cycle\ORM\SchemaInterface;
use Cycle\ORM\Parser\Typecast;
use App\Application\Domain\Entity\ExtendedTypecast;
use Cycle\ORM\Collection\DoctrineCollectionFactory;

return [
    'schema' => [
        'cache' => env('CYCLE_SCHEMA_CACHE', true),
        'defaults' => [
            SchemaInterface::TYPECAST_HANDLER => [
                Typecast::class,
                ExtendedTypecast::class,
            ],
        ],
        'collections' => [
            'default' => 'doctrine',
            'factories' => ['doctrine' => new DoctrineCollectionFactory()],
        ],
        'generators' => null,
    ],
    'warmup' => env('CYCLE_SCHEMA_WARMUP', false),
];
