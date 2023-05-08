<?php

declare(strict_types=1);

return [

    /**
     * -------------------------------------------------------------------------
     *  Default Distribution Resolver Name
     * -------------------------------------------------------------------------
     *
     * Here you can specify which of the resolvers you want to use in the
     * default for all work with URI generation. Of course, you can use
     * multiple resolvers at the same time using the distribution library.
     *
     */

    'default' => env('DISTRIBUTION_RESOLVER', 'local'),

    'resolvers' => [
        'local' => [
            'type' => 'static',
            'uri' => '/',
        ],
    ],

];
