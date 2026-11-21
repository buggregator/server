<?php
declare(strict_types=1);
namespace App\Service;

use App\Repo\UserRepo;
use App\Model\{User, Post as P};
use function App\Util\helper;

interface Saver extends \Countable { public function save(User $u): void; }
trait T { function t() {} }
enum Kind: string { case A = 'a'; public function label(): string { return 'x'; } }

final class UserService extends Base implements Saver, \JsonSerializable
{
    use T;
    public function __construct(private UserRepo $repo) {}
    public function save(User $u): void
    {
        $this->repo->persist($u);
        $this->log('x');
        $this->boot();
        self::make();
        parent::boot();
        static::make();
        UserRepo::find(1);
        $p = new P();
        $x = new \DateTime();
        helper(1);
        strlen('a');
        $f = fn() => $this->log('y');
    }
    private function log(string $m): void {}
    public static function make(): static { return new static(); }
}
function top(): void { (new UserService(new UserRepo()))->save(new User()); }
