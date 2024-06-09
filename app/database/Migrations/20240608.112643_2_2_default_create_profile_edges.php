<?php

declare(strict_types=1);

namespace Database\Migrations;

use Cycle\Migrations\Migration;

class OrmDefault972859df19369e40b49f2fae46c0a310 extends Migration
{
    protected const DATABASE = 'default';

    public function up(): void
    {
        $this->table('profile_edges')
            ->addColumn('uuid', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 36])
            ->addColumn('profile_uuid', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 36])
            ->addColumn('order', 'integer', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('callee', 'text', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('caller', 'text', ['nullable' => true, 'defaultValue' => null])
            ->addColumn('parent_uuid', 'string', ['nullable' => true, 'defaultValue' => null, 'size' => 36])
            ->addColumn('cpu', 'integer', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('wt', 'integer', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('ct', 'integer', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('mu', 'integer', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('pmu', 'integer', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('d_cpu', 'integer', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('d_wt', 'integer', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('d_ct', 'integer', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('d_mu', 'integer', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('d_pmu', 'integer', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('p_cpu', 'float', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('p_wt', 'float', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('p_ct', 'float', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('p_mu', 'float', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('p_pmu', 'float', ['nullable' => false, 'defaultValue' => null])
            ->addIndex(['profile_uuid'], ['name' => 'profile_edges_index_profile_uuid_66643ff3139b6', 'unique' => false],
            )
            ->addIndex(['parent_uuid'], ['name' => 'profile_edges_index_parent_uuid_66643ff3139e7', 'unique' => false])
            ->addForeignKey(['profile_uuid'], 'profiles', ['uuid'], [
                'name' => 'profile_edges_foreign_profile_uuid_66643ff3139c9',
                'delete' => 'CASCADE',
                'update' => 'CASCADE',
                'indexCreate' => true,
            ])
            ->addForeignKey(['parent_uuid'], 'profile_edges', ['uuid'], [
                'name' => 'profile_edges_foreign_parent_uuid_66643ff3139f1',
                'delete' => 'CASCADE',
                'update' => 'CASCADE',
                'indexCreate' => true,
            ])
            ->setPrimaryKeys(['uuid'])
            ->create();
    }

    public function down(): void
    {
        $this->table('profile_edges')->drop();
    }
}
