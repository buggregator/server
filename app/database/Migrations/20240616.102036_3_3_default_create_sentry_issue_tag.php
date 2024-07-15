<?php

declare(strict_types=1);

namespace Database\Migrations;

use Cycle\Migrations\Migration;

class OrmDefaultEaf9d271c936d277c83848f39607bce4 extends Migration
{
    protected const DATABASE = 'default';

    public function up(): void
    {
        $this->table('sentry_issue_tag')
        ->addColumn('issue_uuid', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 36])
        ->addColumn('tag', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 255])
        ->addColumn('value', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 255])
        ->addIndex(['issue_uuid'], ['name' => 'sentry_issue_tag_index_issue_uuid_666ebc74b7863', 'unique' => false])
        ->addForeignKey(['issue_uuid'], 'sentry_issues', ['uuid'], [
            'name' => 'sentry_issue_tag_foreign_issue_uuid_666ebc74b7870',
            'delete' => 'CASCADE',
            'update' => 'CASCADE',
            'indexCreate' => true,
        ])
        ->setPrimaryKeys(['issue_uuid', 'tag'])
        ->create();
    }

    public function down(): void
    {
        $this->table('sentry_issue_tag')->drop();
    }
}
