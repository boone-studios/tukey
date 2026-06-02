<?php

namespace App\Infrastructure\Database;

final class Connection
{
    const DEFAULT_PORT = 3306;
    const MAX_RETRIES = 3;

    private static ?Connection $instance = null;
    private \PDO $pdo;

    private function __construct(string $dsn, string $username, string $password)
    {
        $this->pdo = new \PDO($dsn, $username, $password);
    }

    public static function getInstance(array $config = []): self
    {
        if (self::$instance === null) {
            $dsn = sprintf(
                'mysql:host=%s;port=%d;dbname=%s;charset=utf8mb4',
                $config['host'] ?? 'localhost',
                $config['port'] ?? self::DEFAULT_PORT,
                $config['database'] ?? 'app'
            );
            self::$instance = new self($dsn, $config['username'] ?? 'root', $config['password'] ?? '');
        }
        return self::$instance;
    }

    public function query(string $sql, array $params = []): \PDOStatement
    {
        $stmt = $this->pdo->prepare($sql);
        $stmt->execute($params);
        return $stmt;
    }

    public function insert(string $table, array $data): int
    {
        $cols = implode(', ', array_keys($data));
        $placeholders = implode(', ', array_fill(0, count($data), '?'));
        $sql = "INSERT INTO {$table} ({$cols}) VALUES ({$placeholders})";
        $this->query($sql, array_values($data));
        return (int) $this->pdo->lastInsertId();
    }

    public function update(string $table, array $data, array $where): int
    {
        $setParts = array_map(fn($col) => "{$col} = ?", array_keys($data));
        $whereParts = array_map(fn($col) => "{$col} = ?", array_keys($where));
        $sql = "UPDATE {$table} SET " . implode(', ', $setParts) . " WHERE " . implode(' AND ', $whereParts);
        $stmt = $this->query($sql, array_merge(array_values($data), array_values($where)));
        return $stmt->rowCount();
    }

    public function delete(string $table, array $where): int
    {
        $whereParts = array_map(fn($col) => "{$col} = ?", array_keys($where));
        $sql = "DELETE FROM {$table} WHERE " . implode(' AND ', $whereParts);
        $stmt = $this->query($sql, array_values($where));
        return $stmt->rowCount();
    }

    public function beginTransaction(): void
    {
        $this->pdo->beginTransaction();
    }

    public function commit(): void
    {
        $this->pdo->commit();
    }

    public function rollback(): void
    {
        $this->pdo->rollback();
    }

    public static function reset(): void
    {
        self::$instance = null;
    }
}
