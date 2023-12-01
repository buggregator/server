<?php

declare(strict_types=1);

namespace Migration;

use Cycle\Migrations\Migration;

class OrmDefaultA580ac210e75d2560aaa96f96a407702 extends Migration
{
    protected const DATABASE = 'default';

    public function up(): void
    {
        $this->table('events')
        ->addColumn('uuid', 'string', ['nullable' => false, 'default' => null])
        ->addColumn('type', 'string', ['nullable' => false, 'default' => null])
        ->addColumn('payload', 'longText', ['nullable' => false, 'default' => null])
        ->addColumn('timestamp', 'float', ['nullable' => false, 'default' => null])
        ->addColumn('project_id', 'integer', ['nullable' => true, 'default' => null])
        ->setPrimaryKeys(['uuid'])
        ->create();
    }

    public function down(): void
    {
        $this->table('events')->drop();
    }
}
