<?php

declare(strict_types=1);

namespace Database\Migrations;

use Cycle\Migrations\Migration;

class OrmDefault010519bf047ccc1faaca91e6f526f378 extends Migration
{
    protected const DATABASE = 'default';

    public function up(): void
    {
        $this->table('sentry_issue_fingerprints')
            ->addColumn('created_at', 'datetime', ['nullable' => false, 'defaultValue' => null, 'withTimezone' => false],
            )
            ->addColumn('uuid', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 36])
            ->addColumn('issue_uuid', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 36])
            ->addColumn('fingerprint', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 50])
            ->addIndex(['issue_uuid'],
                ['name' => 'sentry_issue_fingerprints_index_issue_uuid_666ebc74b7929', 'unique' => false])
            ->addIndex(['issue_uuid', 'fingerprint'], ['name' => '9961459aa46305dec16ff24fb1284ae6', 'unique' => true])
            ->addForeignKey(['issue_uuid'], 'sentry_issues', ['uuid'], [
                'name' => 'bad38aad05e5c71fac6b43c8f4ef8066',
                'delete' => 'CASCADE',
                'update' => 'CASCADE',
                'indexCreate' => true,
            ])
            ->setPrimaryKeys(['uuid'])
            ->create();
    }

    public function down(): void
    {
        $this->table('sentry_issue_fingerprints')->drop();
    }
}
