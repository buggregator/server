<?php

declare(strict_types=1);

namespace App\Application\Domain\Entity;

use App\Application\Domain\ValueObjects\Json;
use App\Application\Domain\ValueObjects\Uuid;
use Cycle\ORM\Parser\CastableInterface;
use Cycle\ORM\Parser\UncastableInterface;
use Ramsey\Uuid\UuidInterface;

final class ExtendedTypecast implements CastableInterface, UncastableInterface
{
    private array $rules = [];

    private array $availableRules = ['uuid', 'json'];

    public function setRules(array $rules): array
    {
        foreach ($rules as $key => $rule) {
            if (\in_array($rule, $this->availableRules, true)) {
                unset($rules[$key]);
                $this->rules[$key] = $rule;
            }
        }

        return $rules;
    }

    public function cast(array $data): array
    {
        foreach ($this->rules as $column => $rule) {
            if (!isset($data[$column])) {
                continue;
            }

            $data[$column] = match ($rule) {
                'uuid' => Uuid::fromString($data[$column]),
                'json' => new Json(\json_decode((string) $data[$column], true)),
                default => $data[$column],
            };
        }

        return $data;
    }

    public function uncast(array $data): array
    {
        foreach ($this->rules as $column => $rule) {
            if (!isset($data[$column])) {
                continue;
            }

            $data[$column] = match ($rule) {
                'uuid' => $this->uncastUuid($data[$column]),
                'json' => $this->uncastJson($data[$column]),
                default => $data[$column],
            };
        }

        return $data;
    }

    private function uncastUuid(mixed $value): string
    {
        if ($value instanceof Uuid || $value instanceof UuidInterface) {
            return (string) $value;
        }

        return $value;
    }

    private function uncastJson(mixed $value): string
    {
        return (string) $value;
    }
}
