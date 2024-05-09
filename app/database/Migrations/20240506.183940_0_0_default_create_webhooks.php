<?php

declare(strict_types=1);

namespace Database\Migration;

use Cycle\Migrations\Migration;

class OrmDefaultFe65025ced69992a1d1d9088b9137d1d extends Migration
{
    protected const DATABASE = 'default';

    public function up(): void
    {
        $this->table('webhooks')
            ->addColumn('uuid', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 36])
            ->addColumn('key', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 50])
            ->addColumn('event', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 50])
            ->addColumn('url', 'text', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('headers', 'json', ['nullable' => false])
            ->addColumn('verify_ssl', 'boolean', ['nullable' => false, 'defaultValue' => false])
            ->addColumn('retry_on_failure', 'boolean', ['nullable' => false, 'defaultValue' => true])
            ->addIndex(['key'], ['name' => 'webhooks_index_key_663923eccdd6a', 'unique' => true])
            ->setPrimaryKeys(['uuid'])
            ->create();
    }

    public function down(): void
    {
        $this->table('webhooks')->drop();
    }
}
