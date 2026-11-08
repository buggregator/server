<?php
namespace App;

interface Repo { public function find(int $id): ?array; }

final class UserRepo implements Repo {
    private array $rows = [];
    public function __construct() {
        for ($i = 1; $i <= 5; $i++) {
            $this->rows[$i] = ['id' => $i, 'name' => "user\t$i\nline"];
        }
    }
    public function find(int $id): ?array { return $this->rows[$id] ?? null; }
}

function fib(int $n): int { return $n < 2 ? $n : fib($n - 1) + fib($n - 2); }
