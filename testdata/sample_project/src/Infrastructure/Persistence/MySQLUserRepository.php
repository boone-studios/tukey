<?php

namespace App\Infrastructure\Persistence;

use App\Domain\User;
use App\Domain\UserRepository;
use App\Domain\ValueObjects\Email;
use App\Domain\Enums\UserStatus;
use App\Infrastructure\Database\Connection;

class MySQLUserRepository implements UserRepository
{
    const TABLE = 'users';

    private Connection $connection;
    private array $cache = [];

    public function __construct(Connection $connection)
    {
        $this->connection = $connection;
    }

    public function findById(int $id): mixed
    {
        if (isset($this->cache[$id])) {
            return $this->cache[$id];
        }
        $rows = $this->connection->query("SELECT * FROM " . self::TABLE . " WHERE id = ?", [$id])->fetchAll();
        if (empty($rows)) {
            return null;
        }
        $user = $this->hydrate($rows[0]);
        $this->cache[$id] = $user;
        return $user;
    }

    public function findByEmail(string $email): ?User
    {
        $rows = $this->connection->query("SELECT * FROM " . self::TABLE . " WHERE email = ?", [$email])->fetchAll();
        return !empty($rows) ? $this->hydrate($rows[0]) : null;
    }

    public function findByStatus(UserStatus $status): array
    {
        $rows = $this->connection->query("SELECT * FROM " . self::TABLE . " WHERE status = ?", [$status->value])->fetchAll();
        return array_map([$this, 'hydrate'], $rows);
    }

    public function findActiveUsers(int $limit = 100): array
    {
        $rows = $this->connection->query(
            "SELECT * FROM " . self::TABLE . " WHERE status = ? LIMIT ?",
            [UserStatus::Active->value, $limit]
        )->fetchAll();
        return array_map([$this, 'hydrate'], $rows);
    }

    public function findAll(): array
    {
        $rows = $this->connection->query("SELECT * FROM " . self::TABLE)->fetchAll();
        return array_map([$this, 'hydrate'], $rows);
    }

    public function save(mixed $entity): void
    {
        $data = ['name' => $entity->name ?? '', 'email' => $entity->getEmail()->getValue(), 'status' => $entity->getStatus()->value];
        if ($entity->isNew()) {
            $this->connection->insert(self::TABLE, $data);
        } else {
            $this->connection->update(self::TABLE, $data, ['id' => $entity->getId()]);
            unset($this->cache[$entity->getId()]);
        }
    }

    public function delete(int $id): void
    {
        $this->connection->delete(self::TABLE, ['id' => $id]);
        unset($this->cache[$id]);
    }

    public function existsByEmail(string $email): bool
    {
        $rows = $this->connection->query("SELECT id FROM " . self::TABLE . " WHERE email = ?", [$email])->fetchAll();
        return !empty($rows);
    }

    public function countByStatus(UserStatus $status): int
    {
        $rows = $this->connection->query("SELECT COUNT(*) as cnt FROM " . self::TABLE . " WHERE status = ?", [$status->value])->fetchAll();
        return (int) ($rows[0]['cnt'] ?? 0);
    }

    private function hydrate(array $row): User
    {
        $email = Email::fromString($row['email']);
        $status = UserStatus::from($row['status']);
        return new User($row['name'], $email, $status);
    }
}
