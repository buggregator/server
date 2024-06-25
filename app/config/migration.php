<?php

declare(strict_types=1);

use Cycle\Schema\Generator\Migrations\Strategy\MultipleFilesStrategy;

return [
    /**
     * Directory to store migration files
     */
    'directory' => directory('app') . 'database/Migrations/',

    /**
     * Table name to store information about migrations status (per database)
     */
    'table' => 'migrations',

    'strategy' => MultipleFilesStrategy::class,

    /**
     * When set to true no confirmation will be requested on migration run.
     */
    'safe' => true,
];
