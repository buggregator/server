<?php

declare(strict_types=1);

namespace Database\Migrations;

use Cycle\Migrations\Migration;

class OrmDefaultB85343121e5fbd05ec587256dfb444e9 extends Migration
{
    protected const DATABASE = 'default';

    public function up(): void
    {
        $this->table('events')
        ->addColumn('group_id', 'string', ['nullable' => true, 'defaultValue' => null, 'size' => 255])
        ->addIndex(['group_id'], ['name' => 'events_index_group_id_667079a9c5a94', 'unique' => false])
        ->update();
    }

    public function down(): void
    {
        $this->table('events')
        ->dropIndex(['group_id'])
        ->dropColumn('group_id')
        ->update();
    }
}
