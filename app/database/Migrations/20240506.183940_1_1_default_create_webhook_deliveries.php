<?php

declare(strict_types=1);

namespace Migration;

use Cycle\Migrations\Migration;

class OrmDefault7d099ecd872a405640cbb3ca5af6b869 extends Migration
{
    protected const DATABASE = 'default';

    public function up(): void
    {
        $this->table('webhook_deliveries')
            ->addColumn('created_at', 'datetime', ['nullable' => false, 'defaultValue' => null, 'withTimezone' => false])
            ->addColumn('uuid', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 36])
            ->addColumn('webhook_uuid', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 36])
            ->addColumn('payload', 'text', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('response', 'text', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('status', 'integer', ['nullable' => false, 'defaultValue' => null])
            ->addIndex(['webhook_uuid'], ['name' => 'webhook_deliveries_index_webhook_uuid_663923eccdcd5', 'unique' => false])
            ->addForeignKey(['webhook_uuid'], 'webhooks', ['uuid'], [
                'name' => 'webhook_deliveries_foreign_webhook_uuid_663923eccdbe1',
                'delete' => 'CASCADE',
                'update' => 'CASCADE',
                'indexCreate' => true,
            ])
            ->setPrimaryKeys(['uuid'])
            ->create();
    }

    public function down(): void
    {
        $this->table('webhook_deliveries')->drop();
    }
}
