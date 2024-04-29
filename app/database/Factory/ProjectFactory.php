<?php

declare(strict_types=1);

namespace Database\Factory;

use Modules\Projects\Domain\Project;
use Modules\Projects\Domain\ValueObject\Key;
use Spiral\DatabaseSeeder\Factory\AbstractFactory;

final class ProjectFactory extends AbstractFactory
{
    public function entity(): string
    {
        return Project::class;
    }

    public function definition(): array
    {
        return [
            'key' => Key::create($this->faker->unique()->slug(2)),
            'name' => $this->faker->sentence(),
        ];
    }

    public function makeEntity(array $definition): object
    {
        return new Project(
            key: $definition['key'] instanceof Key ? $definition['key'] : Key::create($definition['key']),
            name: $definition['name'],
        );
    }
}
