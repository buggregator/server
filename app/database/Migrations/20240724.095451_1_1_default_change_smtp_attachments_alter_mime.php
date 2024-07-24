<?php

declare(strict_types=1);

namespace Migration;

use Cycle\Migrations\Migration;

class OrmDefaultD93e77c9f5556975e93bfbc969442731 extends Migration
{
    protected const DATABASE = 'default';

    public function up(): void
    {
        $this->table('smtp_attachments')
        ->alterColumn('mime', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 127])
        ->update();
    }

    public function down(): void
    {
        $this->table('smtp_attachments')
        ->alterColumn('mime', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 32])
        ->update();
    }
}
