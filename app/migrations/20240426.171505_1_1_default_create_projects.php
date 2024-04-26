<?php

declare(strict_types=1);

namespace Migration;

use Cycle\Migrations\Migration;

class OrmDefault2a897d68b0647d8ad446d9074be5a172 extends Migration
{
    protected const DATABASE = 'default';

    public function up(): void
    {
        $this->table('projects')
        ->addColumn('key', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 36])
        ->addColumn('name', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 255])
        ->setPrimaryKeys(['key'])
        ->create();
    }

    public function down(): void
    {
        $this->table('projects')->drop();
    }
}
