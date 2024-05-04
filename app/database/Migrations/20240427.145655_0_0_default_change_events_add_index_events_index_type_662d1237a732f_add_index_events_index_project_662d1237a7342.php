<?php

declare(strict_types=1);

namespace Database\Migrations;

use Cycle\Migrations\Migration;

class OrmDefaultB8ad031bf07a9610d0483c64b42db6a8 extends Migration
{
    protected const DATABASE = 'default';

    public function up(): void
    {
        $this->table('events')
            ->addIndex(['type'], ['name' => 'events_index_type_662d1237a732f', 'unique' => false])
            ->addIndex(['project'], ['name' => 'events_index_project_662d1237a7342', 'unique' => false])
            ->update();
    }

    public function down(): void
    {
        $this->table('events')
            ->dropIndex(['type'])
            ->dropIndex(['project'])
            ->update();
    }
}
