<?php

declare(strict_types=1);

return [
    /**
     * Directory to store migration files
     */
    'directory' => directory('app') . 'migrations/',

    /**
     * Table name to store information about migrations status (per database)
     */
    'table' => 'migrations',

    'strategy' => \Cycle\Schema\Generator\Migrations\Strategy\MultipleFilesStrategy::class,

    /**
     * When set to true no confirmation will be requested on migration run.
     */
    'safe' => true,
];
