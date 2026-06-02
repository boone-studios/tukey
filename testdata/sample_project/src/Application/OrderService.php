<?php

namespace App\Application;

use App\Application\UserService;
use App\Domain\Order;
use App\Domain\OrderRepository;
use App\Domain\Enums\OrderStatus;
use App\Domain\ValueObjects\Money;

class OrderService
{
    private OrderRepository $orderRepository;
    private UserService $userService;

    public function __construct(OrderRepository $orderRepository, UserService $userService)
    {
        $this->orderRepository = $orderRepository;
        $this->userService = $userService;
    }

    public function getOrdersByUser(int $userId): array
    {
        return $this->orderRepository->findByUserId($userId);
    }

    public function getTotalSpentByUser(int $userId): int
    {
        $orders = $this->orderRepository->findByUserId($userId);
        $total = 0;
        foreach ($orders as $order) {
            $total += $order->getTotal()->getAmount();
        }
        return $total;
    }

    public function processOrder(int $orderId): Order
    {
        $order = $this->orderRepository->findById($orderId);
        if ($order === null) {
            throw new \RuntimeException("Order {$orderId} not found");
        }
        // Circular: OrderService calls UserService
        $user = $this->userService->getUserById($order->getUserId());
        if ($user === null || !$user->getStatus()->isActive()) {
            throw new \DomainException("Cannot process order for inactive user");
        }
        $order->transition(OrderStatus::Processing);
        $this->orderRepository->save($order);
        return $order;
    }

    public function cancelOrder(int $orderId): void
    {
        $order = $this->orderRepository->findById($orderId);
        if ($order === null) {
            throw new \RuntimeException("Order not found");
        }
        $order->cancel();
        $this->orderRepository->save($order);
    }

    public function getPendingOrders(): array
    {
        return $this->orderRepository->findByStatus(OrderStatus::Pending);
    }

    public function getTotalRevenue(): Money
    {
        $cents = $this->orderRepository->getTotalRevenue();
        return Money::fromCents($cents);
    }

    public function getOrderSummary(int $orderId): array
    {
        $order = $this->orderRepository->findById($orderId);
        if ($order === null) {
            return [];
        }
        $user = $this->userService->getUserById($order->getUserId());
        return [
            'order'    => $order->getPayload(),
            'customer' => $user ? $user->jsonSerialize() : null,
            'items'    => $order->getItems(),
        ];
    }
}
