<?php

namespace App\Domain;

use App\Domain\Traits\SoftDeletable;
use App\Domain\Enums\UserStatus;
use App\Domain\ValueObjects\Email;
use App\Domain\Contracts\Cacheable;

class User extends Entity implements \JsonSerializable, \Stringable, Cacheable
{
    use SoftDeletable;

    const CACHE_TTL = 3600;
    const ADMIN_ROLE = 'admin';

    private string $name;
    private Email $email;
    private UserStatus $status;
    private string $role;
    private array $permissions = [];
    private static array $instances = [];

    public function __construct(string $name, Email $email, UserStatus $status)
    {
        $this->name = $name;
        $this->email = $email;
        $this->status = $status;
        $this->role = 'user';
    }

    public static function create(string $name, string $rawEmail): self
    {
        $email = Email::fromString($rawEmail);
        $user = new self($name, $email, UserStatus::Active);
        self::$instances[$rawEmail] = $user;
        return $user;
    }

    public function getId(): ?int
    {
        return $this->id;
    }

    public function validate(): bool
    {
        return $this->name !== '' && !$this->isDeleted();
    }

    public function activate(): void
    {
        $this->status = UserStatus::Active;
    }

    public function suspend(): void
    {
        $this->status = UserStatus::Suspended;
    }

    public function promote(string $role): void
    {
        $this->role = $role;
        if ($role === self::ADMIN_ROLE) {
            $this->permissions[] = 'manage_users';
        }
    }

    public function isAdmin(): bool
    {
        return $this->role === self::ADMIN_ROLE;
    }

    public function hasPermission(string $permission): bool
    {
        return in_array($permission, $this->permissions);
    }

    public function getEmail(): Email
    {
        return $this->email;
    }

    public function getStatus(): UserStatus
    {
        return $this->status;
    }

    public function getCacheKey(): string
    {
        return 'user:' . $this->id;
    }

    public function getCacheTtl(): int
    {
        return self::CACHE_TTL;
    }

    public function clearCache(string $key): void
    {
        unset(self::$instances[$key]);
    }

    public function jsonSerialize(): mixed
    {
        return ['id' => $this->id, 'name' => $this->name, 'status' => $this->status->value];
    }

    public function __toString(): string
    {
        return $this->name . ' <' . $this->email->getValue() . '>';
    }
}
