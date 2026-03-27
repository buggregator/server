<?php

declare(strict_types=1);

namespace Database\Migrations;

use Cycle\Migrations\Migration;

class OrmDefaultAddIsPinnedToEvents extends Migration
{
    protected const DATABASE = 'default';

    public function up(): void
    {
        $this->table('events')
            ->addColumn('is_pinned', 'boolean', ['nullable' => false, 'defaultValue' => false])
            ->update();
    }

    public function down(): void
    {
        $this->table('events')
            ->dropColumn('is_pinned')
            ->update();
    }
}
