<?php

declare(strict_types=1);

namespace App\Application\Domain\Entity;

use App\Domain\ValueObjects\Uuid;
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

    public function cast(array $values): array
    {
        foreach ($this->rules as $column => $rule) {
            if (!isset($values[$column])) {
                continue;
            }

            $values[$column] = match ($rule) {
                'uuid' => Uuid::fromString($values[$column]),
                'json' => new Json(\json_decode($values[$column], true)),
                default => $values[$column]
            };
        }

        return $values;
    }

    public function uncast(array $values): array
    {
        foreach ($this->rules as $column => $rule) {
            if (!isset($values[$column])) {
                continue;
            }

            $values[$column] = match ($rule) {
                'uuid' => $this->uncastUuid($values[$column]),
                'json' => $this->uncastJson($values[$column]),
                default => $values[$column]
            };
        }

        return $values;
    }

    private function uncastUuid(mixed $value): string
    {
        if ($value instanceof Uuid || $value instanceof UuidInterface) {
            return $value->toString();
        }

        return $value;
    }

    private function uncastJson(mixed $value): string
    {
        return (string)$value;
    }
}
