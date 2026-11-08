<?php
require __DIR__ . '/lib.php';

use App\UserRepo;

$repo = new UserRepo();
$names = [];
foreach ([1, 2, 9] as $id) {
    $u = $repo->find($id);
    $names[] = $u ? strtoupper($u['name']) : 'missing';
}
$f = App\fib(6);
$s = sprintf("%s|%d", implode(',', $names), $f);
$fn = static fn(string $x): string => str_repeat($x, 2);
$t = $fn("a\tb");
try {
    throw new \RuntimeException('boom');
} catch (\RuntimeException $e) {
    $msg = $e->getMessage();
}
echo strlen($s), "\n";
