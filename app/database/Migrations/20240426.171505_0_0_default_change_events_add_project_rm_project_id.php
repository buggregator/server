<?php

declare(strict_types=1);

namespace Database\Migrations;

use Cycle\Migrations\Migration;

class OrmDefault5b1ee362e1b19b19f535c6f9da36cb55 extends Migration
{
    protected const DATABASE = 'default';

    public function up(): void
    {
        $this->table('events')
            ->addColumn('project', 'string', ['nullable' => true, 'defaultValue' => null, 'size' => 255])
            ->dropColumn('project_id')
            ->update();
    }

    public function down(): void
    {
        $this->table('events')
            ->addColumn('project_id', 'integer', ['nullable' => true, 'defaultValue' => null])
            ->dropColumn('project')
            ->update();
    }
}
