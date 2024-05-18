<?php

declare(strict_types=1);

namespace Database\Migrations;

use Cycle\Migrations\Migration;

class OrmDefaultDca21e486cc30724624c30812f0e24df extends Migration
{
    protected const DATABASE = 'default';

    public function up(): void
    {
        $this->table('http_dump_attachments')
            ->addColumn('uuid', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 36])
            ->addColumn('event_uuid', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 36])
            ->addColumn('name', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 255])
            ->addColumn('path', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 255])
            ->addColumn('size', 'integer', ['nullable' => false, 'defaultValue' => 0])
            ->addColumn('mime', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 32])
            ->addIndex(['event_uuid'],
                ['name' => 'http_dump_attachments_index_event_uuid_6647a54986782', 'unique' => false])
            ->setPrimaryKeys(['uuid'])
            ->create();
    }

    public function down(): void
    {
        $this->table('http_dump_attachments')->drop();
    }
}
