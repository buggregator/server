<?php

declare(strict_types=1);

namespace Database\Migrations;

use Cycle\Migrations\Migration;
use Cycle\ORM\EntityManagerInterface;
use Modules\Projects\Domain\Project;
use Modules\Projects\Domain\ProjectRepositoryInterface;
use Modules\Projects\Domain\ValueObject\Key;
use Spiral\Core\ContainerScope;

class OrmDefaultD93e77c9f5556975e93bfbc969442732 extends Migration
{
    protected const DATABASE = 'default';

    public function up(): void
    {
        $this->getEntityManager()
            ->persist(new Project(Key::create(Project::DEFAULT_KEY), 'Default project'))
            ->run();
    }

    public function down(): void
    {
        $defaultProject = $this->getRepository()->findOne(['key' => Project::DEFAULT_KEY]);
        if ($defaultProject) {
            $this->getEntityManager()->delete($defaultProject)->run();
        }
    }

    private function getEntityManager(): EntityManagerInterface
    {
        return ContainerScope::getContainer()->get(EntityManagerInterface::class);
    }

    private function getRepository(): ProjectRepositoryInterface
    {
        return ContainerScope::getContainer()->get(ProjectRepositoryInterface::class);
    }
}
