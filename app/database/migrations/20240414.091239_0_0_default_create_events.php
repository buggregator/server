<?php

declare(strict_types=1);

namespace Migration;

use Cycle\Migrations\Migration;

class OrmDefault1df1e263c7c7d0a163f630de97e0b78c extends Migration
{
    protected const DATABASE = 'default';

    public function up(): void
    {
        $this->table('events')
        ->addColumn('uuid', 'string', ['nullable' => false, 'size' => 36])
        ->addColumn('type', 'string', ['nullable' => false, 'size' => 50])
        ->addColumn('payload', 'jsonb', ['nullable' => false])
        ->addColumn('timestamp', 'float', ['nullable' => false])
        ->addColumn('project_id', 'integer', ['nullable' => true])
        ->setPrimaryKeys(['uuid'])
        ->create();
    }

    public function down(): void
    {
        $this->table('events')->drop();
    }
}
