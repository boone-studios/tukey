<?php

namespace App\Http\Controllers;

use App\Application\OrderService;
use App\Application\Commands\PlaceOrderCommand;
use App\Application\Handlers\PlaceOrderHandler;
use App\Domain\Enums\OrderStatus;

class OrderController
{
    private OrderService $orderService;

    public function __construct(OrderService $orderService)
    {
        $this->orderService = $orderService;
    }

    public function index(): array
    {
        $orders = $this->orderService->getPendingOrders();
        return ['data' => array_map(fn($o) => $o->getPayload(), $orders), 'code' => 200];
    }

    public function show(int $id): array
    {
        $summary = $this->orderService->getOrderSummary($id);
        if (empty($summary)) {
            return ['error' => 'Order not found', 'code' => 404];
        }
        return ['data' => $summary, 'code' => 200];
    }

    public function cancel(int $id): array
    {
        try {
            $this->orderService->cancelOrder($id);
            return ['code' => 200];
        } catch (\RuntimeException $e) {
            return ['error' => $e->getMessage(), 'code' => 404];
        }
    }

    public function process(int $id): array
    {
        try {
            $order = $this->orderService->processOrder($id);
            return ['data' => $order->getPayload(), 'code' => 200];
        } catch (\Exception $e) {
            return ['error' => $e->getMessage(), 'code' => 500];
        }
    }

    public function revenue(): array
    {
        $money = $this->orderService->getTotalRevenue();
        return ['data' => ['amount' => $money->getAmount(), 'currency' => $money->getCurrency()], 'code' => 200];
    }
}
