<?php

declare(strict_types=1);

namespace Migration;

use Cycle\Migrations\Migration;

class OrmDefault4d821d8599782e856feb0c9b8b26b42f extends Migration
{
    protected const DATABASE = 'default';

    public function up(): void
    {
        $this->table('events')
        ->alterColumn('timestamp', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 25])
        ->update();
    }

    public function down(): void
    {

    }
}
