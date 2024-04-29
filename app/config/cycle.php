<?php

declare(strict_types=1);

use Cycle\ORM\Collection\ArrayCollectionFactory;
use Cycle\ORM\Collection\DoctrineCollectionFactory;
use Cycle\ORM\Collection\IlluminateCollectionFactory;

return [
    'schema' => [
        'cache' => env('CYCLE_SCHEMA_CACHE', true),
        'defaults' => [
            \Cycle\ORM\SchemaInterface::TYPECAST_HANDLER => [
                \Cycle\ORM\Parser\Typecast::class,
                \App\Application\Domain\Entity\ExtendedTypecast::class,
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
