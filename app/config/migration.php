<?php

declare(strict_types=1);

use Cycle\Schema\Generator\Migrations\Strategy\MultipleFilesStrategy;

return [
    'directory' => directory('app') . 'database/Migrations/',

    'table' => 'migrations',

    'strategy' => MultipleFilesStrategy::class,

    'safe' => true,

    'namespace' => 'Database\Migrations',
];
