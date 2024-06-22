<?php

declare(strict_types=1);

namespace Database\Migrations;

use Cycle\Migrations\Migration;

class OrmDefault2125b91024f8895d66c10f3ce9668b4f extends Migration
{
    protected const DATABASE = 'default';

    public function up(): void
    {
        $this->table('sentry_issues')
        ->addColumn('created_at', 'datetime', ['nullable' => false, 'defaultValue' => null, 'withTimezone' => false])
        ->addColumn('uuid', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 36])
        ->addColumn('trace_uuid', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 36])
        ->addColumn('title', 'text', ['nullable' => false, 'defaultValue' => null])
        ->addColumn('platform', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 32])
        ->addColumn('logger', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 32])
        ->addColumn('type', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 32])
        ->addColumn('transaction', 'string', ['nullable' => true, 'defaultValue' => null, 'size' => 255])
        ->addColumn('server_name', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 255])
        ->addColumn('payload', 'jsonb', ['nullable' => false, 'defaultValue' => null])
        ->addIndex(['trace_uuid'], ['name' => 'sentry_issues_index_trace_uuid_666ebc74b7900', 'unique' => false])
        ->addForeignKey(['trace_uuid'], 'sentry_traces', ['uuid'], [
            'name' => 'sentry_issues_foreign_trace_uuid_666ebc74b78fb',
            'delete' => 'CASCADE',
            'update' => 'CASCADE',
            'indexCreate' => true,
        ])
        ->setPrimaryKeys(['uuid'])
        ->create();
    }

    public function down(): void
    {
        $this->table('sentry_issues')->drop();
    }
}
