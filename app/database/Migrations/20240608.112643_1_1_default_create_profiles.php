<?php

declare(strict_types=1);

namespace Database\Migrations;

use Cycle\Migrations\Migration;

class OrmDefault178280ff82db54b14ccc25ce1028b465 extends Migration
{
    protected const DATABASE = 'default';

    public function up(): void
    {
        $this->table('profiles')
            ->addColumn('uuid', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 36])
            ->addColumn('name', 'string', ['nullable' => false, 'defaultValue' => null, 'size' => 255])
            ->addColumn('cpu', 'integer', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('wt', 'integer', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('ct', 'integer', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('mu', 'integer', ['nullable' => false, 'defaultValue' => null])
            ->addColumn('pmu', 'integer', ['nullable' => false, 'defaultValue' => null])
            ->setPrimaryKeys(['uuid'])
            ->create();
    }

    public function down(): void
    {
        $this->table('profiles')->drop();
    }
}
