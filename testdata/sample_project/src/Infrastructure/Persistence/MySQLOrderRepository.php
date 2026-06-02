<?php

namespace App\Infrastructure\Persistence;

use App\Domain\Order;
use App\Domain\OrderRepository;
use App\Domain\OrderItem;
use App\Domain\Enums\OrderStatus;
use App\Domain\ValueObjects\Money;
use App\Infrastructure\Database\Connection;

class MySQLOrderRepository implements OrderRepository
{
    const TABLE = 'orders';
    const ITEMS_TABLE = 'order_items';

    private Connection $connection;

    public function __construct(Connection $connection)
    {
        $this->connection = $connection;
    }

    public function findById(int $id): mixed
    {
        $rows = $this->connection->query("SELECT * FROM " . self::TABLE . " WHERE id = ?", [$id])->fetchAll();
        return !empty($rows) ? $this->hydrate($rows[0]) : null;
    }

    public function findByUserId(int $userId): array
    {
        $rows = $this->connection->query(
            "SELECT * FROM " . self::TABLE . " WHERE user_id = ? ORDER BY created_at DESC",
            [$userId]
        )->fetchAll();
        return array_map([$this, 'hydrate'], $rows);
    }

    public function findByStatus(OrderStatus $status): array
    {
        $rows = $this->connection->query("SELECT * FROM " . self::TABLE . " WHERE status = ?", [$status->value])->fetchAll();
        return array_map([$this, 'hydrate'], $rows);
    }

    public function findAll(): array
    {
        $rows = $this->connection->query("SELECT * FROM " . self::TABLE)->fetchAll();
        return array_map([$this, 'hydrate'], $rows);
    }

    public function findPendingOlderThan(\DateTime $date): array
    {
        $rows = $this->connection->query(
            "SELECT * FROM " . self::TABLE . " WHERE status = ? AND created_at < ?",
            [OrderStatus::Pending->value, $date->format('Y-m-d H:i:s')]
        )->fetchAll();
        return array_map([$this, 'hydrate'], $rows);
    }

    public function save(mixed $entity): void
    {
        $data = [
            'user_id'  => $entity->getUserId(),
            'status'   => $entity->getStatus()->value,
            'total'    => $entity->getTotal()->getAmount(),
            'currency' => $entity->getTotal()->getCurrency(),
        ];
        if ($entity->isNew()) {
            $this->connection->insert(self::TABLE, $data);
        } else {
            $this->connection->update(self::TABLE, $data, ['id' => $entity->getId()]);
        }
    }

    public function delete(int $id): void
    {
        $this->connection->delete(self::TABLE, ['id' => $id]);
        $this->connection->delete(self::ITEMS_TABLE, ['order_id' => $id]);
    }

    public function getTotalRevenue(): int
    {
        $rows = $this->connection->query(
            "SELECT SUM(total) as revenue FROM " . self::TABLE . " WHERE status = ?",
            [OrderStatus::Delivered->value]
        )->fetchAll();
        return (int) ($rows[0]['revenue'] ?? 0);
    }

    public function countByUser(int $userId): int
    {
        $rows = $this->connection->query("SELECT COUNT(*) as cnt FROM " . self::TABLE . " WHERE user_id = ?", [$userId])->fetchAll();
        return (int) ($rows[0]['cnt'] ?? 0);
    }

    private function hydrate(array $row): Order
    {
        return new Order($row['user_id'], $row['currency'] ?? 'USD');
    }
}
