<?php

declare(strict_types=1);

namespace Modules\Projects\Application;

use Modules\Projects\Domain\ProjectFactoryInterface;
use Modules\Projects\Domain\ProjectLocatorInterface;
use Modules\Projects\Domain\ValueObject\Key;
use Psr\Log\LoggerInterface;
use Symfony\Component\Finder\Finder;
use Symfony\Component\Yaml\Yaml;

final readonly class YamlFileProjectLocator implements ProjectLocatorInterface
{
    public function __construct(
        private Finder $finder,
        private LoggerInterface $logger,
        private ProjectFactoryInterface $projectFactory,
        private string $directory,
    ) {
    }

    public function findAll(): iterable
    {
        $this->finder->files()->in($this->directory)->name('*.project.yaml');

        foreach ($this->finder as $file) {
            try {
                /**
                 * @var array{project: array{
                 *     key: string,
                 *     name: string,
                 * }} $data
                 */
                $data = Yaml::parseFile($file->getPathname());
                $this->validateData($data);

                $project = $this->projectFactory->create(
                    key: Key::create($data['project']['key']),
                    name: $data['project']['name'],
                );

                yield $project;
            } catch (\Throwable $e) {
                $this->logger->error(
                    'Failed to parse project file',
                    ['file' => $file->getPathname(), 'error' => $e->getMessage()],
                );
            }
        }
    }

    private function validateData(array $data): void
    {
        $errors = [];
        // Yaml structure validation
        if (!isset($data['project'])) {
            throw new \RuntimeException('Missing "project" key');
        }

        if (!\is_array($data['project'])) {
            throw new \RuntimeException('Invalid "project" key');
        }

        if (!isset($data['project']['key'])) {
            $errors[] = 'Missing "key" key';
        } elseif (!\is_string($data['project']['key'])) {
            $errors[] = 'Invalid "key" key';
        }

        if (!isset($data['project']['name'])) {
            $errors[] = 'Missing "name" key';
        } elseif (!\is_string($data['project']['name'])) {
            $errors[] = 'Invalid "name" key';
        }

        if ($errors !== []) {
            throw new \RuntimeException(\implode(', ', $errors));
        }
    }
}
