<?php

declare(strict_types=1);

namespace Database\Migrations;

use Cycle\Migrations\Migration;

class OrmDefault7dc66c49f446f6ce1e8f1e0b52f70491 extends Migration
{
    protected const DATABASE = 'default';

    public function up(): void
    {
        $this->table('smtp_attachments')
            ->addColumn('uuid', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 36])
            ->addColumn('event_uuid', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 36])
            ->addColumn('name', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 255])
            ->addColumn('path', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 255])
            ->addColumn('size', 'integer', ['nullable' => false, 'defaultValue' => 0])
            ->addColumn('mime', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 32])
            ->addColumn('id', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 255])
            ->addIndex(['event_uuid'], ['name' => 'attachments_index_event_uuid_66348ada9b3e3', 'unique' => false])
            ->setPrimaryKeys(['uuid'])
            ->create();
    }

    public function down(): void
    {
        $this->table('smtp_attachments')->drop();
    }
}
