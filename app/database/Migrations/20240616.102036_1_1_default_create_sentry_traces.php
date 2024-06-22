<?php

declare(strict_types=1);

namespace Database\Migrations;

use Cycle\Migrations\Migration;

class OrmDefault706076e16879158d6f8d79771cf7e44c extends Migration
{
    protected const DATABASE = 'default';

    public function up(): void
    {
        $this->table('sentry_traces')
        ->addColumn('uuid', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 36])
        ->addColumn('trace_id', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 32])
        ->addColumn('public_key', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 255])
        ->addColumn('environment', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 255])
        ->addColumn('sampled', 'boolean', ['nullable' => false, 'defaultValue' => null])
        ->addColumn('sample_rate', 'float', ['nullable' => false, 'defaultValue' => null])
        ->addColumn('transaction', 'string', ['nullable' => true, 'defaultValue' => null, 'size' => 255])
        ->addColumn('sdk', 'jsonb', ['nullable' => false, 'defaultValue' => null])
        ->addColumn('language', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 255])
        ->addIndex(['trace_id'], ['name' => 'sentry_traces_index_trace_id_666ebc74b7a32', 'unique' => true])
        ->setPrimaryKeys(['uuid'])
        ->create();
    }

    public function down(): void
    {
        $this->table('sentry_traces')->drop();
    }
}
